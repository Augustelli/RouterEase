package services

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"strings"

	//      "github.com/golang-jwt/jwt/v5"
	"github.com/miekg/dns"
	"io"
	"net/http"
	//      "strings"
	"bytes"
	"log"
)

func DoHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	log.Println("Entering DoHandler")
	// Only accept POST with correct content type
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/dns-message" {
		log.Printf("DoHandler: Unsupported method or content type. Method: %s, Content-Type: %s", r.Method, r.Header.Get("Content-Type"))
		http.Error(w, "Unsupported method or content type", http.StatusBadRequest)
		return
	}
	log.Println("DoHandler: Method and content type check passed.")

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

	// Read DNS query from body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("DoHandler: Failed to read request body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	log.Println("DoHandler: Successfully read request body.")

	log.Printf("Handling DoH request: method=%s, path=%s, params=%v, headers=%v, body=%s", r.Method, r.URL.Path, r.URL.Query(), r.Header, string(body))

	dnsReq := new(dns.Msg)
	if err := dnsReq.Unpack(body); err != nil {
		log.Printf("DoHandler: Failed to unpack DNS message from body: %v", err)
		http.Error(w, "Invalid DNS message", http.StatusBadRequest)
		return
	}
	log.Printf("DoHandler: Successfully unpacked DNS message. Request: %+v", dnsReq)

	if len(dnsReq.Question) == 0 {
		log.Println("DoHandler: No DNS question found in the request.")
		http.Error(w, "No DNS question", http.StatusBadRequest)
		return
	}
	log.Printf("DoHandler: DNS Question: %v", dnsReq.Question[0])
	queryName := dnsReq.Question[0].Name

	var resp *dns.Msg

	// Blocklist filtering
	blockedDomain, err := IsDomainBlockedForUUID(db, uuid, queryName)
	if err != nil {
		log.Printf("DoHandler: Error checking if domain is blocked: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if blockedDomain {
		resp = new(dns.Msg)
		resp.SetReply(dnsReq)
		resp.Rcode = dns.RcodeNameError // NXDOMAIN

		// Pack the DNS message back into binary format for the response.
		out, err := resp.Pack()
		if err != nil {
			log.Printf("DoHandler: Failed to pack DNS response: %v", err)
			http.Error(w, "Failed to pack DNS response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/dns-message")
		w.Write(out)
		log.Println("DoHandler: NXDOMAIN response sent. Exiting handler.")
		return
	}

	// If not blocked, forward to PowerDNS
	if resp == nil {
		log.Println("DoHandler: Forwarding DNS query to PowerDNS.")
		var queryErr error
		resp, queryErr = queryPowerDNSMsg(dnsReq)
		if queryErr != nil {
			log.Printf("DoHandler: PowerDNS query failed: %v", queryErr)
			http.Error(w, "DNS query failed", http.StatusInternalServerError)
			return
		}
		log.Printf("DoHandler: Received response from PowerDNS: %+v", resp)
	}

	// Pack the DNS message back into binary format for the response.
	out, err := resp.Pack()
	if err != nil {
		log.Printf("DoHandler: Failed to pack DNS response: %v", err)
		http.Error(w, "Failed to pack DNS response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/dns-message")
	w.Write(out)
	log.Println("DoHandler: Response sent. Exiting handler.")
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

type DNSResolveRequest struct {
	Name string `json:"name"`
}

func DNSResolveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Log the request details including the body
	log.Printf("Handling request: method=%s, path=%s, headers=%v, body=%s", r.Method, r.URL.Path, r.Header, string(body))

	var req DNSResolveRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Missing domain name in request body", http.StatusBadRequest)
		return
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(req.Name), dns.TypeA)
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
