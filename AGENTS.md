# AGENTS.md — RouterEase

## Mission and scope
RouterEase is a home-network management platform for OpenWrt. Finish a **fully functional, reproducible MVP**, preserving real router operation, per-device DNS/parental-control policies, strong security, and usable management UIs. Work from the `dev-api` baseline, commit work in a dedicated feature branch, and merge only after review and tests. Do **not** treat the thesis's aspirational statements as proof of implemented behavior.

## Architecture invariants
1. **Two execution environments**: a real OpenWrt router runs the installable RouterEase package (and built-in network services); a separate single backend host runs **all backend services in Docker Compose**. The Docker OpenWrt environment is a development/test harness, not the production router package.
2. **Router stays lightweight**: LuCI integration, UCI/ubus, leases/neighbors, local DNS forwarding, network changes and a supervised `procd` agent. Do not make the router depend on Python, Docker, Keycloak, or backend databases for normal operation.
3. **Backend owns policies**: management/control plane owns users, devices, policy CRUD, blocklists, allowlists, groups, schedules, auditing and configuration. The DNS data plane MUST apply policies to real DNS queries (not only store them). Separate admin JWT authorization from router-to-backend API-key authentication.
4. **Device-specific DNS is non-negotiable**: a single router API key identifies the router, **not** the origin device. Design and test a trustworthy way to associate queries with a router and LAN device (IP/lease/MAC mapping with bounded validity, or an explicit on-router policy forwarder). Never trust arbitrary user-supplied `client_ip` or MAC as authority without validating its authenticated router and source.
5. **Gateway**: Traefik terminates TLS and routes API/management web traffic. DNS protocol traffic is not automatically HTTP traffic: do not assert that Traefik's HTTP routers serve UDP/TCP 53. Specify, test, and document the separate DNS, DoH and management paths and exposure.
6. **Identity**: keep Keycloak for user auth, roles and OIDC; use individually provisioned, revocable and rotatable router API keys, stored hashed at rest where applicable. Avoid embedding administrator credentials or long-lived static bearer tokens in LuCI JS/browser bundles.
7. **Persistence**: use PostgreSQL for app policies/devices and a distinct logical database for Keycloak; durable named volumes and migrations. Database data directories are never committed to Git. Keep database ports internal unless specifically justified.
8. **Observability**: backend instrumentation, metrics and logs via Prometheus, Grafana and Jaeger/OpenTelemetry as applicable. Scrape targets must match real Compose service names; avoid sensitive DNS domains, JWTs, API keys or Wi-Fi passwords in logs/traces. Observability must not become a dependency for DNS resolution.
9. **Two UIs**: LuCI for on-router local operations; a separate management frontend (framework to be chosen) for users, devices, filters, schedules, rules, diagnostics and dashboards. Both use versioned API contracts. Management changes require verified authorization.
10. **MVP-first**: no SaaS/multi-host orchestration, Kubernetes, speculative dependencies or cloud-only requirements unless explicitly approved. Keep the backend on one host and make deployment documented and repeatable.

## Existing repository layout — baseline (`dev-api`)
- `API/docker-compose.yml`: current Compose aggregator using `extends`; currently references reverse proxy, whoami, PowerDNS, GO-Proxie, MACLookup, Keycloak and two DBs. Prometheus and Jaeger have separate Compose definitions and are not included in this aggregator.
- `API/ReverseProxie/`: Traefik configs; currently includes risky Docker socket and dashboard exposure; tighten.
- `API/Authentication/`: Keycloak and PostgreSQL. Existing plaintext credentials and committed runtime data must be treated as exposed.
- `API/GO-Proxie/`: Go HTTP API, GORM CRUD for users/groups/domains, and DoH forwarding. Existing JWT and DNS blocklist checks in `services/dns-service.go` are commented out. Treat enforcement as **unimplemented**.
- `API/DNS/`: PowerDNS Recursor configuration and an experimental Go `main.go` not part of the active Compose. Decide whether to integrate or remove only after tests.
- `API/MACLookUp/`: OUI manufacturer identification service. Does NOT establish authoritative device identity by itself; locally administered/randomized MAC addresses need special handling.
- `API/Observability/`: Prometheus and Jaeger definitions; reconcile with real runtime.
- `Package/`: OpenWrt-in-Docker test harness, Lua LuCI controller and views; not yet evidence of a complete cross-device `.ipk`/`.apk` package with `procd`. `Package/monitoring` holds separate legacy InfluxDB/Grafana files; not part of the chosen canonical observability stack.

## Source of truth, in priority order
1. Approved architectural decisions in `docs/ARCHITECTURE.md` and accepted ADRs.
2. Executable code + verified tests and Compose effective configuration.
3. `docs/ROADMAP.md` for incomplete work and acceptance criteria.
4. TIF-4 thesis as **product intent/context**, not necessarily current implementation. `docs/THESIS-TRACEABILITY.md` documents alignment and gaps.
If these conflict, report the discrepancy instead of silently assuming functionality exists.

## Standards and guardrails
- Prefer the **smallest coherent change**. Before edits, inspect affected services, Compose dependencies, ports, networks, persistent paths, current API paths, and on-router behavior.
- Never hardcode production secrets, router admin/Wi-Fi credentials, public IPs, passwords, API keys, static JWTs or deployment domains. Use `.env.example` with nonsecret placeholders and validated required configuration. Do not use `latest` or `start-dev` for production.
- Never commit `.env`, `.idea`, databases, `pg_wal`, ACME private material, generated logs, router configuration credentials, or runtime volumes. `.gitignore` is not retroactive: removing tracked assets requires a separate deliberate commit, and historical exposures require key rotation/history remediation.
- Input validation, permission checks and resource ownership on every management endpoint. API keys are scoped to registered routers. JWT audience/issuer/signature/expiry must be validated. Enforce strict HTTP methods/statuses and bounded request sizes.
- Changes to DNS resolution, DHCP, firewall or network UCI are HIGH RISK: provide config backup, rollback and a failsafe; never run destructive rules on the developer host by default. DNS failover behavior (fail-open vs fail-closed) must be specified and tested.
- Don't expose backend databases, Prometheus, Grafana or Jaeger publicly by default. Protect Traefik dashboard; avoid permissive CORS with credentials. Bind Docker socket read-only only if absolutely needed; prefer a restricted socket proxy.
- Avoid logging raw DNS wire payloads, Authorization headers, cookies or passwords.
- Explicitly distinguish `dev`, `test/lab` and `production` topology. The OpenWrt emulator is optional and isolated in dev; production uses a physical router with supported OpenWrt firmware.
- Existing source uses Go and Lua. Use idiomatic Go modules, `gofmt`, table-driven tests and context/timeouts. For Lua/LuCI ensure OpenWrt compatibility, UCI atomicity and safe shell-command construction. Add smoke/integration tests for APIs, DB migrations, DNS policy decisions and container health.

## Key behaviors that define "fully functional"
1. Clean host can run the backend with documented prerequisites, appropriate secrets and health checks.
2. Router can install, configure, start/stop and upgrade RouterEase package on a supported physical OpenWrt device; agent survives reboots and reports health.
3. An admin can sign in through Keycloak and manage household/router/device/user/group and parental DNS policies in a separate frontend.
4. A registered router connects using a scoped revocable API key. Device identity is reliably derived and mapped to rules, even as DHCP leases change.
5. An actual DNS query from device A can be blocked while the same domain for device B is allowed; allowlist precedence, schedule boundaries, default behavior, and policy updates are tested.
6. Restarting individual backend containers does not destroy configuration and does not cause uncontrolled router configuration changes.
7. Prometheus and Jaeger ingest working backend telemetry; Grafana can show relevant dashboard panels, and observability outages do not break DNS.
8. Security checks, backup/restore instructions and a recovery procedure are documented.

## Work protocol for coding agents
For each change: **inspect → plan → implement → test → document**. Give changed paths, risks, backward compatibility, commands run and results. Never claim tests passed unless executed. On changes spanning multiple services, draw/update sequence diagrams and APIs. Consult `API/AGENTS.md`, `Package/AGENTS.md`, `docs/ARCHITECTURE.md`, `docs/SECURITY.md` and ADRs before changing the stack.

## Initial audit tasks — highest priority
1. Remove tracked secrets/runtime state in a dedicated remediation PR; rotate exposed credentials, move runtime data to Docker named volumes and document history cleanup separately.
2. Normalize Compose effective configuration, networks, volumes, mounted file paths and startup health dependencies; integrate Prometheus/Grafana/Jaeger for dev and protected ops.
3. Separate DNS data plane from policy management plane, implement authenticated router registration and trustworthy per-device attribution. Replace hardcoded blocklists/tokens with DB-backed policy CRUD and tested enforcement.
4. Build real OpenWrt package with `procd`, UCI settings, LuCI views, install/upgrade/uninstall and supported firmware target, while preserving emulator for testing.
5. Create independent management web frontend and end-to-end test matrix.

## Reference commands (run in the appropriate directory and environment; do not assume success)
- `docker compose -f API/docker-compose.yml config` (baseline config audit; may fail because current definitions reference external networks and relative files).
- `docker compose -f API/docker-compose.yml ps` / `logs --tail=100 <service>`.
- `(cd API/GO-Proxie && go test ./...)` and `(cd API/MACLookUp && go test ./...)` after Go is available.
- `(cd Package && make build)` and `make run` **ONLY inside a dedicated lab host** after reviewing bridge/interface manipulations. This is **not** the physical router packaging process.

## Document boundaries
`AGENTS.md` defines durable agent rules; `docs/ARCHITECTURE.md` describes current vs target design; `docs/ROADMAP.md` tracks delivery; `docs/SECURITY.md` defines remediations; ADRs capture finalized tradeoffs. Keep them synchronized.
