# Thesis ↔ implementation traceability

Reference: `TIF-4-Mancuso,Augusto Tomas.docx`, section 2.1.19 and figures 3–5 (~pages 55–59), conclusions and results 4.1–4.6 (~pages 86–90). Interpret the thesis as scope/intended architecture; prefer executable source and tests for current operational state.

| Thesis feature/claim | Evidence in `dev-api` | Gap to target MVP |
|---|---|---|
| Lightweight router/client, LuCI and installable package | `Package/package/controller/` and views; Docker OpenWrt harness | Native supported `.ipk`/`.apk`, supervised agent and physical testing remain to be built/verified |
| Traefik centralized entry | `API/ReverseProxie/`, `API/docker-compose.yml` | Secure edge config, routes and protocol-specific DNS exposures |
| Keycloak OIDC auth | `API/Authentication/` | JWT/roles on actual admin API; secure SPA auth and router enrollment |
| DNS filter/parental groups by device | `API/GO-Proxie/services/dns-service.go`, GORM CRUD, PowerDNS | Filtering/JWT blocks commented out; actual per-device identity and policy enforcement unimplemented |
| Recursor DNS | `API/DNS/powerdns-docker-compose.yaml` | Harden allow-list, define forwarding/recursion, readiness and correct policy interception |
| MAC manufacturer enrichment | `API/MACLookUp/` | Tests, durable OUI update and randomized-MAC handling; do not conflate OUI with authentication |
| Observability metrics/tracing | Prometheus and Jaeger side compose definitions | Canonical integration, correct targets and secure Grafana dashboards |
| Simple installation / onboarding | `Package/HOW-TO-RUN.md` lab | Physical install instructions and frontend guided enrollment |

## Terminology and cautions
The thesis refers to a `routerease-agent` with `procd`, but the current branch does not yet show a complete released package including this functionality. Document it as target, not done. The thesis discusses per-device parental filtering as a result; the active `go-proxie` handler currently does not enforce its commented filter logic. These distinctions are required in issue descriptions and demos.
