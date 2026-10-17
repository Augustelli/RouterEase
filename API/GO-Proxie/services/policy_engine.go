package services

import (
	"context"
	"errors"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const policyCacheTTL = 5 * time.Minute

type Decision string

const (
	DecisionAllow       Decision = "ALLOW"
	DecisionBlock       Decision = "BLOCK"
	DecisionUnavailable Decision = "UNAVAILABLE"
)

type PolicyDecision struct {
	Decision Decision
	Reason   string
}

type cachedPolicy struct {
	decision PolicyDecision
	expires  time.Time
}

// PolicyEngine caches decisions rather than raw DNS data. A mutation calls
// Invalidate, so changes take effect on the next query without a restart.
type PolicyEngine struct {
	db    *gorm.DB
	mu    sync.RWMutex
	cache map[string]cachedPolicy
}

func NewPolicyEngine(db *gorm.DB) *PolicyEngine {
	return &PolicyEngine{db: db, cache: make(map[string]cachedPolicy)}
}

func (e *PolicyEngine) Invalidate() {
	e.mu.Lock()
	e.cache = make(map[string]cachedPolicy)
	e.mu.Unlock()
}

func (e *PolicyEngine) ResolveDevice(ctx context.Context, routerID, clientIP string) (Device, error) {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return Device{}, errors.New("invalid client IP")
	}
	var lease DeviceLease
	if err := e.db.WithContext(ctx).Where("router_id = ? AND ip = ? AND expires_at > now()", routerID, ip.String()).First(&lease).Error; err != nil {
		return Device{}, err
	}
	var device Device
	if err := e.db.WithContext(ctx).Where("id = ? AND router_id = ?", lease.DeviceID, routerID).First(&device).Error; err != nil {
		return Device{}, err
	}
	return device, nil
}

func (e *PolicyEngine) Evaluate(ctx context.Context, deviceID, qname string) (PolicyDecision, error) {
	domain, err := normalizeDomain(qname)
	if err != nil {
		return PolicyDecision{}, err
	}
	key := deviceID + "|" + domain
	e.mu.RLock()
	cached, ok := e.cache[key]
	e.mu.RUnlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.decision, nil
	}

	decision, err := e.evaluateDB(ctx, deviceID, domain)
	if err != nil {
		// A valid cached value remains usable during a transient DB outage.
		if ok && time.Now().Before(cached.expires) {
			return cached.decision, nil
		}
		return PolicyDecision{Decision: DecisionUnavailable, Reason: "policy-store-unavailable"}, err
	}
	e.mu.Lock()
	e.cache[key] = cachedPolicy{decision: decision, expires: time.Now().Add(policyCacheTTL)}
	e.mu.Unlock()
	return decision, nil
}

type applicableRule struct {
	DNSRule
	Priority     int
	DeviceScoped bool
}

func (e *PolicyEngine) evaluateDB(ctx context.Context, deviceID, domain string) (PolicyDecision, error) {
	var rules []applicableRule
	err := e.db.WithContext(ctx).Raw(`
SELECT r.id, r.policy_id, r.domain, r.action, r.enabled, r.created_at, r.updated_at,
       a.priority, (a.device_id IS NOT NULL) AS device_scoped
FROM policy_assignments a
JOIN dns_policies p ON p.id = a.policy_id AND p.enabled
JOIN dns_rules r ON r.policy_id = p.id AND r.enabled
WHERE a.device_id = ? OR a.group_id IN (
  SELECT group_id FROM device_group_members WHERE device_id = ?
)`, deviceID, deviceID).Scan(&rules).Error
	if err != nil {
		return PolicyDecision{}, err
	}
	matching := make([]applicableRule, 0, len(rules))
	for _, rule := range rules {
		if domainMatches(domain, rule.Domain) {
			matching = append(matching, rule)
		}
	}
	if len(matching) > 0 {
		// Explicit ALLOW is an exception and wins over BLOCK. Within the same
		// action select the narrowest suffix, then lowest priority, then device.
		sort.Slice(matching, func(i, j int) bool {
			ai, aj := matching[i], matching[j]
			if ai.Action != aj.Action {
				return ai.Action == string(DecisionAllow)
			}
			if len(ai.Domain) != len(aj.Domain) {
				return len(ai.Domain) > len(aj.Domain)
			}
			if ai.Priority != aj.Priority {
				return ai.Priority < aj.Priority
			}
			if ai.DeviceScoped != aj.DeviceScoped {
				return ai.DeviceScoped
			}
			return ai.ID < aj.ID
		})
		winner := matching[0]
		return PolicyDecision{Decision: Decision(winner.Action), Reason: "explicit-rule"}, nil
	}

	type fallback struct {
		Action       string
		Priority     int
		DeviceScoped bool
		ID           string
	}
	var defaults []fallback
	err = e.db.WithContext(ctx).Raw(`
SELECT p.default_action AS action, a.priority, (a.device_id IS NOT NULL) AS device_scoped, p.id
FROM policy_assignments a JOIN dns_policies p ON p.id = a.policy_id AND p.enabled
WHERE a.device_id = ? OR a.group_id IN (SELECT group_id FROM device_group_members WHERE device_id = ?)
`, deviceID, deviceID).Scan(&defaults).Error
	if err != nil {
		return PolicyDecision{}, err
	}
	if len(defaults) == 0 {
		return PolicyDecision{Decision: DecisionAllow, Reason: "no-assignment"}, nil
	}
	sort.Slice(defaults, func(i, j int) bool {
		if defaults[i].Priority != defaults[j].Priority {
			return defaults[i].Priority < defaults[j].Priority
		}
		if defaults[i].DeviceScoped != defaults[j].DeviceScoped {
			return defaults[i].DeviceScoped
		}
		return defaults[i].ID < defaults[j].ID
	})
	return PolicyDecision{Decision: Decision(defaults[0].Action), Reason: "policy-default"}, nil
}

func normalizeDomain(value string) (string, error) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if domain == "" || len(domain) > 253 || net.ParseIP(domain) != nil {
		return "", errors.New("invalid DNS name")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", errors.New("invalid DNS name")
		}
		for _, r := range label {
			if !(r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return "", errors.New("invalid DNS name")
			}
		}
	}
	return domain, nil
}

// domainMatches implements exact-or-descendant matching, never substring matching.
func domainMatches(query, configured string) bool {
	configured = strings.TrimSuffix(strings.ToLower(configured), ".")
	return query == configured || strings.HasSuffix(query, "."+configured)
}
