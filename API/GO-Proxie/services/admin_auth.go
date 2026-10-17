package services

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

type adminClaims struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type AdminAuthorizer struct {
	issuer, audience string
	once             sync.Once
	verifier         *oidc.IDTokenVerifier
	err              error
}

func NewAdminAuthorizerFromEnv() *AdminAuthorizer {
	return &AdminAuthorizer{issuer: os.Getenv("OIDC_ISSUER"), audience: os.Getenv("OIDC_AUDIENCE")}
}
func (a *AdminAuthorizer) init(ctx context.Context) {
	if a.issuer == "" || a.audience == "" {
		a.err = errors.New("OIDC_ISSUER and OIDC_AUDIENCE must be configured")
		return
	}
	provider, err := oidc.NewProvider(ctx, a.issuer)
	if err != nil {
		a.err = err
		return
	}
	a.verifier = provider.Verifier(&oidc.Config{ClientID: a.audience})
}
func (a *AdminAuthorizer) Authorize(ctx context.Context, authorization string) error {
	if !strings.HasPrefix(authorization, "Bearer ") {
		return errors.New("missing bearer token")
	}
	a.once.Do(func() { a.init(ctx) })
	if a.err != nil {
		return a.err
	}
	token, err := a.verifier.Verify(ctx, strings.TrimPrefix(authorization, "Bearer "))
	if err != nil {
		return err
	}
	var claims adminClaims
	if err := token.Claims(&claims); err != nil {
		return err
	}
	for _, role := range claims.RealmAccess.Roles {
		if role == "routerease-admin" {
			return nil
		}
	}
	for _, resource := range claims.ResourceAccess {
		for _, role := range resource.Roles {
			if role == "routerease-admin" {
				return nil
			}
		}
	}
	return errors.New("routerease-admin role required")
}
func (a *AdminAuthorizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Authorize(r.Context(), r.Header.Get("Authorization")); err != nil {
			http.Error(w, "admin authorization required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
