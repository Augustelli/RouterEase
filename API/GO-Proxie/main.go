package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miekg/dns"
	"log"
	"net/http"
	"strings"
)

// Blocklists for each group
var blocklists = map[string][]string{
	"ParentalControl": {"adultsite.com", "explicitcontent.com"},
	"Adblocking":      {"tracker.com", "ads.com"},
	"Common":          {}, // No filtering
}

// Group mapping (UUID -> Group)
var userGroups = map[string]string{
	"07a5b831-2aab-4d57-8517-41fc29195f78": "ParentalControl",
	"54eed05e-2947-42cd-a519-9fe14455aade": "Adblocking",
	"4089f0e0-cbf2-4652-bedb-cddad94a9448": "Common",
}

// Middleware to handle DoH requests
func dohHandler(w http.ResponseWriter, r *http.Request) {
	// Extract JWT from Authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse JWT without validation
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return nil, nil // No validation, return nil for the key
	})

	// Extract UUID (sub claim)
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

	// Map UUID to group
	group, exists := userGroups[uuid]
	if !exists {
		http.Error(w, "User group not found", http.StatusForbidden)
		return
	}

	// Apply filtering based on group
	blocklist := blocklists[group]
	query := r.URL.Query().Get("name") // Extract DNS query name
	for _, blockedDomain := range blocklist {
		if strings.Contains(query, blockedDomain) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf("%s\t0\tIN\tA\tNXDOMAIN\n", query)))
			return
		}
	}

	// Forward request to PowerDNS
	answers, err := queryPowerDNS(query)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}

	// Relay DNS response back to client
	w.WriteHeader(http.StatusOK)
	for _, answer := range answers {
		_, _ = w.Write([]byte(answer.String() + "\n"))
	}
}

func queryPowerDNS(domain string) ([]dns.RR, error) {
	// Create a new DNS message
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA) // Query for A record

	// Set up the DNS client
	c := new(dns.Client)

	// Send the query to the PowerDNS server
	r, _, err := c.Exchange(m, "localhost:5301") // PowerDNS server address
	if err != nil {
		return nil, fmt.Errorf("failed to query PowerDNS: %v", err)
	}

	// Return the response answers
	return r.Answer, nil
}

func main() {
	http.HandleFunc("/dns-query", dohHandler)
	log.Println("Starting proxy on :8080")
	log.Fatal(http.ListenAndServe(":8087", nil))
}
