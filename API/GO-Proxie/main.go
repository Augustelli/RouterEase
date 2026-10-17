package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"GO-Proxie/controllers"
	"GO-Proxie/services"
)

func main() {
	db := services.InitializeDB()
	engine := services.NewPolicyEngine(db)
	admin := services.NewAdminAuthorizerFromEnv()
	mux := http.NewServeMux()
	mux.Handle("/api/v1/", admin.Middleware(controllers.V1API{DB: db, Engine: engine}))
	mux.Handle("/api/v1/router/leases", controllers.RouterAPI{DB: db, Engine: engine})
	mux.Handle("/api/v1/router/dns-query", services.DNSHandler{DB: db, Engine: engine, Upstream: os.Getenv("DNS_UPSTREAM")})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	// Retire the previous unauthenticated CRUD paths rather than retaining a
	// privilege-escalation path alongside the versioned API.
	mux.HandleFunc("/users/", retired)
	mux.HandleFunc("/groups/", retired)
	mux.HandleFunc("/domains/", retired)
	mux.HandleFunc("/dns-query", retired)
	server := &http.Server{Addr: ":8080", Handler: securityHeaders(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Println("RouterEase policy API listening on :8080")
	log.Fatal(server.ListenAndServe())
}

func retired(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "endpoint retired; use authenticated /api/v1", http.StatusGone)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
