package services

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/miekg/dns"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDomainMatches(t *testing.T) {
	cases := []struct {
		query, rule string
		want        bool
	}{
		{"example.org", "example.org", true},
		{"cdn.example.org", "example.org", true},
		{"badexample.org", "example.org", false},
		{"example.org.evil.test", "example.org", false},
	}
	for _, tc := range cases {
		if got := domainMatches(tc.query, tc.rule); got != tc.want {
			t.Errorf("domainMatches(%q,%q)=%v, want %v", tc.query, tc.rule, got, tc.want)
		}
	}
}

func TestAuthenticatedDoHBlocksOneDeviceAndForwardsAnother(t *testing.T) {
	db := integrationDB(t)
	router := Router{Name: "dns-lab", Status: "active"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	a := Device{RouterID: router.ID, Name: "child", MACAddress: "02:00:00:00:00:11"}
	b := Device{RouterID: router.ID, Name: "adult", MACAddress: "02:00:00:00:00:12"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	group := DeviceGroup{RouterID: router.ID, Name: "kids"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO device_group_members(device_id,group_id) VALUES (?,?)", a.ID, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	policy := DNSPolicy{Name: "filter", DefaultAction: "ALLOW", Enabled: true}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&DNSRule{PolicyID: policy.ID, Domain: "example.org", Action: "BLOCK", Enabled: true}).Error; err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	if err := db.Create(&PolicyAssignment{PolicyID: policy.ID, GroupID: &groupID, Priority: 1}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, lease := range []DeviceLease{{RouterID: router.ID, DeviceID: a.ID, IP: "192.0.2.10", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}, {RouterID: router.ID, DeviceID: b.ID, IP: "192.0.2.11", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}} {
		if err := db.Create(&lease).Error; err != nil {
			t.Fatal(err)
		}
	}
	_, key, err := IssueRouterKey(db, router.ID, []string{"dns:query"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer packet.Close()
	resolver := &dns.Server{PacketConn: packet, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		reply := new(dns.Msg)
		reply.SetReply(r)
		reply.Answer = []dns.RR{&dns.A{Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 1}, A: net.ParseIP("203.0.113.9")}}
		_ = w.WriteMsg(reply)
	})}
	go func() { _ = resolver.ActivateAndServe() }()
	defer resolver.Shutdown()
	h := DNSHandler{DB: db, Engine: NewPolicyEngine(db), Upstream: packet.LocalAddr().String()}
	query := new(dns.Msg)
	query.SetQuestion("example.org.", dns.TypeA)
	query.Id = 4242
	wire, err := query.Pack()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ip    string
		rcode int
	}{{"192.0.2.10", dns.RcodeNameError}, {"192.0.2.11", dns.RcodeSuccess}} {
		req := httptest.NewRequest(http.MethodPost, "http://policy/api/v1/router/dns-query", bytes.NewReader(wire))
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("X-Router-API-Key", key)
		req.Header.Set("X-Router-Client-IP", tc.ip)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s HTTP=%d", tc.ip, rec.Code)
		}
		answer := new(dns.Msg)
		if err := answer.Unpack(rec.Body.Bytes()); err != nil {
			t.Fatal(err)
		}
		if answer.Id != 4242 || answer.Rcode != tc.rcode {
			t.Fatalf("%s response id=%d rcode=%d", tc.ip, answer.Id, answer.Rcode)
		}
	}
}

func integrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`TRUNCATE policy_decision_audits, device_leases, router_api_keys, policy_assignments, dns_rules, dns_policies, device_group_members, device_groups, devices, routers CASCADE`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPolicyEnginePerDeviceAndInvalidation(t *testing.T) {
	db := integrationDB(t)
	router := Router{Name: "lab-router", Status: "active"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	a := Device{RouterID: router.ID, Name: "child", MACAddress: "02:00:00:00:00:01"}
	b := Device{RouterID: router.ID, Name: "adult", MACAddress: "02:00:00:00:00:02"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	group := DeviceGroup{RouterID: router.ID, Name: "children"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO device_group_members(device_id,group_id) VALUES (?,?)", a.ID, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	policy := DNSPolicy{Name: "child-policy", DefaultAction: "ALLOW", Enabled: true}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	rule := DNSRule{PolicyID: policy.ID, Domain: "example.org", Action: "BLOCK", Enabled: true}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	groupID := group.ID
	if err := db.Create(&PolicyAssignment{PolicyID: policy.ID, GroupID: &groupID, Priority: 10}).Error; err != nil {
		t.Fatal(err)
	}
	engine := NewPolicyEngine(db)
	got, err := engine.Evaluate(context.Background(), a.ID, "www.example.org.")
	if err != nil || got.Decision != DecisionBlock {
		t.Fatalf("child decision=%+v err=%v", got, err)
	}
	got, err = engine.Evaluate(context.Background(), b.ID, "www.example.org.")
	if err != nil || got.Decision != DecisionAllow {
		t.Fatalf("adult decision=%+v err=%v", got, err)
	}
	if err := db.Model(&DNSRule{}).Where("id=?", rule.ID).Update("action", "ALLOW").Error; err != nil {
		t.Fatal(err)
	}
	engine.Invalidate()
	got, err = engine.Evaluate(context.Background(), a.ID, "www.example.org")
	if err != nil || got.Decision != DecisionAllow {
		t.Fatalf("updated decision=%+v err=%v", got, err)
	}
}

func TestRouterAPIKeyLifecycle(t *testing.T) {
	db := integrationDB(t)
	router := Router{Name: "router", Status: "active"}
	if err := db.Create(&router).Error; err != nil {
		t.Fatal(err)
	}
	key, raw, err := IssueRouterKey(db, router.ID, []string{"dns:query"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateRouterKey(db, raw, "dns:query"); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateRouterKey(db, raw, "inventory:write"); err == nil {
		t.Fatal("scope escalation was accepted")
	}
	if err := RevokeRouterKey(db, key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateRouterKey(db, raw, "dns:query"); err == nil {
		t.Fatal("revoked key was accepted")
	}
	expires := time.Now().Add(-time.Minute)
	_, expired, err := IssueRouterKey(db, router.ID, nil, &expires)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateRouterKey(db, expired, "dns:query"); err == nil {
		t.Fatal("expired key was accepted")
	}
}
