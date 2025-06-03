package DNS

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var (
	keycloakIssuer = os.Getenv("KEYCLOAK_ISSUER") // e.g. http://keycloak:8080/realms/yourrealm
	clientID       = os.Getenv("KEYCLOAK_CLIENT_ID")
	clientSecret   = os.Getenv("KEYCLOAK_CLIENT_SECRET")
	dnsdistAPIURL  = os.Getenv("DNSDIST_API_URL") // e.g. http://dnsdist:8083/api
	dnsdistAPIKey  = os.Getenv("DNSDIST_API_KEY")
)

type PolicyRequest struct {
	UserToken string `json:"user_token"`
}

type DnsdistPolicyUpdate struct {
	ClientIP string `json:"client_ip"`
	Policy   string `json:"policy"` // "filtered" or "default"
}

func getUserProfile(token string) (string, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, keycloakIssuer)
	if err != nil {
		return "", err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return "", err
	}
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return "", err
	}
	// Example: parental_control attribute
	if parental, ok := claims["parental_control"].(bool); ok && parental {
		return "filtered", nil
	}
	return "default", nil
}

func updateDnsdistPolicy(clientIP, policy string) error {
	update := DnsdistPolicyUpdate{ClientIP: clientIP, Policy: policy}
	body, _ := json.Marshal(update)
	req, err := http.NewRequest("POST", dnsdistAPIURL+"/policy", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", dnsdistAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("dnsdist API error: %v", resp.Status)
	}
	return nil
}

func getClientIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func policyHandler(w http.ResponseWriter, r *http.Request) {
	var req PolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	clientIP := getClientIP(r)
	policy, err := getUserProfile(req.UserToken)
	if err != nil {
		http.Error(w, "Authentication failed", 401)
		return
	}
	if err := updateDnsdistPolicy(clientIP, policy); err != nil {
		http.Error(w, "Failed to update dnsdist", 500)
		return
	}
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("Policy %s applied for %s", policy, clientIP)))
}

func main() {
	http.HandleFunc("/apply-policy", policyHandler)
	log.Println("DNS Policy Service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
