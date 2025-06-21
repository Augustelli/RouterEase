package services

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miekg/dns"
	"io"
	"net/http"
	"strings"
)

// TODO REFACTORING: Move blocklists and userGroups to a configuration file or database
var blocklists = map[string][]string{
	"ParentalControl": {"adultsite.com", "explicitcontent.com"},
	"Adblocking":      {"tracker.com", "ads.com"},
	"Common":          {},
}

var userGroups = map[string][]string{
	"07a5b831-2aab-4d57-8517-41fc29195f78": {"ParentalControl", "Adblocking"},
	"54eed05e-2947-42cd-a519-9fe14455aade": {"Adblocking"},
	"2bca9ebc-b438-438f-8cd7-002e09e0dca6": {"Common"},
}

func DoHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST with correct content type
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/dns-message" {
		http.Error(w, "Unsupported method or content type", http.StatusBadRequest)
		return
	}

	// Extract JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}
	uuid, ok := claims["sub"].(string)
	if !ok {
		http.Error(w, "UUID not found in token", http.StatusUnauthorized)
		return
	}
	groups, exists := userGroups[uuid]
	if !exists {
		http.Error(w, "User groups not found", http.StatusForbidden)
		return
	}
	blocklist := []string{}
	for _, group := range groups {
		blocklist = append(blocklist, blocklists[group]...)
	}

	// Read DNS query from body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	dnsReq := new(dns.Msg)
	if err := dnsReq.Unpack(body); err != nil {
		http.Error(w, "Invalid DNS message", http.StatusBadRequest)
		return
	}
	if len(dnsReq.Question) == 0 {
		http.Error(w, "No DNS question", http.StatusBadRequest)
		return
	}
	queryName := dnsReq.Question[0].Name

	// Blocklist filtering
	for _, blockedDomain := range blocklist {
		if strings.Contains(queryName, blockedDomain) {
			resp := new(dns.Msg)
			resp.SetReply(dnsReq)
			resp.Rcode = dns.RcodeNameError // NXDOMAIN
			packed, _ := resp.Pack()
			w.Header().Set("Content-Type", "application/dns-message")
			w.WriteHeader(http.StatusOK)
			w.Write(packed)
			return
		}
	}

	// Forward to PowerDNS
	resp, err := queryPowerDNSMsg(dnsReq)
	if err != nil {
		http.Error(w, "DNS query failed", http.StatusInternalServerError)
		return
	}
	packed, _ := resp.Pack()
	w.Header().Set("Content-Type", "application/dns-message")
	w.WriteHeader(http.StatusOK)
	w.Write(packed)
}

func queryPowerDNSMsg(msg *dns.Msg) (*dns.Msg, error) {
	c := new(dns.Client)
	resp, _, err := c.Exchange(msg, "dns:53")
	return resp, err
}

type DNSQuestion struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
}

type DNSAnswer struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
	TTL  uint32 `json:"TTL"`
	Data string `json:"data"`
}

type DNSResponse struct {
	Status   int           `json:"Status"`
	TC       bool          `json:"TC"`
	RD       bool          `json:"RD"`
	RA       bool          `json:"RA"`
	AD       bool          `json:"AD"`
	CD       bool          `json:"CD"`
	Question []DNSQuestion `json:"Question"`
	Answer   []DNSAnswer   `json:"Answer"`
}

func DNSResolveHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("name")
	if domain == "" {
		http.Error(w, "Missing domain parameter", http.StatusBadRequest)
		return
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	resp, err := queryPowerDNSMsg(msg)
	if err != nil {
		http.Error(w, "DNS query failed", http.StatusInternalServerError)
		return
	}

	// Build Question section
	questions := make([]DNSQuestion, len(resp.Question))
	for i, q := range resp.Question {
		questions[i] = DNSQuestion{
			Name: q.Name,
			Type: q.Qtype,
		}
	}

	// Build Answer section
	answers := []DNSAnswer{}
	for _, ans := range resp.Answer {
		if a, ok := ans.(*dns.A); ok {
			answers = append(answers, DNSAnswer{
				Name: a.Hdr.Name,
				Type: a.Hdr.Rrtype,
				TTL:  a.Hdr.Ttl,
				Data: a.A.String(),
			})
		}
	}

	result := DNSResponse{
		Status:   resp.Rcode,
		TC:       resp.Truncated,
		RD:       resp.RecursionDesired,
		RA:       resp.RecursionAvailable,
		AD:       resp.AuthenticatedData,
		CD:       resp.CheckingDisabled,
		Question: questions,
		Answer:   answers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
