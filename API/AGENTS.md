# Backend agents — `API/`

Read root `AGENTS.md` and `docs/ARCHITECTURE.md` first. The API is deployed on **one host** as Docker Compose services, not on the OpenWrt router.

- Preserve domain separation: Traefik (TLS/edge), Keycloak (human auth), Core Go API (policy/data management), per-device DNS enforcement/data plane, PowerDNS (resolver), MACLookup (OUI), PostgreSQL (durable app and Keycloak databases), Prometheus/Grafana/Jaeger (diagnostics). An HTTP reverse proxy is not itself a UDP DNS server.
- Rename/reshape `GO-Proxie` only in an intentional migration with backward-compatible endpoints, tests and Compose updates. Its current CRUD routes lack per-route authorization and its DNS filtering code is commented out.
- Human sessions: OIDC via Keycloak, validate JWT cryptography, issuer, audience, expiry and roles; prefer Authorization Code + PKCE for web frontend. Router credentials: independently provisioned scoped API keys with revocation/rotation; API key alone does not identify individual LAN hosts.
- Design canonical data models/migrations for household, router, device, identity mappings, user, group, policy, policy assignment, blocked domains, allowlist exception, schedule and policy decision audit. Resolve precedence deterministically and test it.
- DNS policy enforcement must run on every relevant query, not on an occasional web request. Decide and document whether on-router forwarding or a per-client-aware resolver is the authoritative lookup boundary. Preserve client identity across NAT; verify DHCP/IP mappings against the authenticated router.
- Avoid `:latest`, plaintext passwords in Compose, `start-dev`, exposed admin panels, overly broad `ALLOW_FROM`, permissive CORS + credentials, and log bodies containing DNS queries/headers. No generated PostgreSQL directories or ACME secrets in Git.
- Production-readiness gate: pinned image versions, healthchecks, safe restart behavior, config validation, resource limits, migrations, least-privileged networking, TLS, backup/restore and curl/dig smoke tests for both allow and deny.
- Add Go unit/integration tests; run `go test ./...`, `go vet ./...` and `gofmt`; validate effective Compose via `docker compose config` before claiming success. Recording expected commands isn't proof of passing tests.
- DNS policy schema changes use immutable SQL files under `GO-Proxie/services/migrations/`; do not reintroduce production `AutoMigrate`. Router DoH requires a scoped API key plus a router-scoped, unexpired lease identity; see `docs/ADR-0001-device-dns-attribution.md`.
- Observability targets must match actual service names and exposed endpoints. Do not couple DNS availability to Prometheus, Grafana or Jaeger availability.
