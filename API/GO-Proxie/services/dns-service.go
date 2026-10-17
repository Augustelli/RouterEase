package services

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	"gorm.io/gorm"
)

const maxDNSMessageBytes = 65535

type DNSHandler struct {
	DB       *gorm.DB
	Engine   *PolicyEngine
	Upstream string
}

func (h DNSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/dns-message") {
		http.Error(w, "POST application/dns-message required", http.StatusBadRequest)
		return
	}
	rawKey := r.Header.Get("X-Router-API-Key")
	if rawKey == "" {
		rawKey = strings.TrimPrefix(r.Header.Get("Authorization"), "Router ")
	}
	key, err := AuthenticateRouterKey(h.DB, rawKey, "dns:query")
	if err != nil {
		http.Error(w, "router authorization required", http.StatusUnauthorized)
		return
	}
	clientIP := r.Header.Get("X-Router-Client-IP")
	if net.ParseIP(clientIP) == nil {
		http.Error(w, "valid X-Router-Client-IP required", http.StatusBadRequest)
		return
	}
	device, err := h.Engine.ResolveDevice(r.Context(), key.RouterID, clientIP)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "unknown or expired router lease", http.StatusForbidden)
		return
	}
	if err != nil {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxDNSMessageBytes))
	if err != nil {
		http.Error(w, "invalid DNS message", http.StatusBadRequest)
		return
	}
	request := new(dns.Msg)
	if err := request.Unpack(body); err != nil || len(request.Question) != 1 {
		http.Error(w, "one valid DNS question required", http.StatusBadRequest)
		return
	}
	decision, evaluateErr := h.Engine.Evaluate(r.Context(), device.ID, request.Question[0].Name)
	if evaluateErr == nil && decision.Decision == DecisionBlock {
		response := new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError // documented NXDOMAIN policy response
		h.writeDNS(w, response)
		return
	}
	// DB/cache errors intentionally fail open after authenticated identity has
	// been established; the resolver itself still returns a clear failure.
	response, err := h.queryPowerDNS(r.Context(), request)
	if err != nil {
		http.Error(w, "resolver unavailable", http.StatusServiceUnavailable)
		return
	}
	h.writeDNS(w, response)
}

func (h DNSHandler) queryPowerDNS(ctx context.Context, request *dns.Msg) (*dns.Msg, error) {
	upstream := h.Upstream
	if upstream == "" {
		upstream = "dns:53"
	}
	response, _, err := (&dns.Client{Timeout: 2 * time.Second}).ExchangeContext(ctx, request, upstream)
	return response, err
}
func (h DNSHandler) writeDNS(w http.ResponseWriter, response *dns.Msg) {
	body, err := response.Pack()
	if err != nil {
		http.Error(w, "failed to encode DNS response", 500)
		return
	}
	w.Header().Set("Content-Type", "application/dns-message")
	_, _ = w.Write(body)
}

// DNSResolveHandler is intentionally retained only as a compatibility stub.
func DNSResolveHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "use authenticated /api/v1/router/dns-query", http.StatusGone)
}
