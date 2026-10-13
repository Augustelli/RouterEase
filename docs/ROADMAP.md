# RouterEase MVP implementation roadmap

**Rule**: a checkbox is complete only with executed tests and evidence. Current entries are TODOs, not verified failures of every deployment environment.

## P0 — safety and repeatability
- [ ] Create feature branch from `dev-api` (blocked for this automation by GitHub integration 403; no branch created as of drafting).
- [ ] Inventory and rotate *all* publicly exposed credentials/cert material and disable the old values. Clean tracked logs/DB volumes/ACME files and add ignores. Plan repository-history remediation; removing in HEAD does not erase history.
- [ ] Pin image and OpenWrt versions; select dependency-update policy and licensing.
- [ ] Replace insecure Traefik dashboard, permissive CORS with credentials, public Keycloak/DB/metrics exposure, and weak database credentials.
- [ ] Confirm effective Compose config and volume mount paths; use named volumes, `.env.example`, secret injection and healthchecks. No hardcoded host domain/IP or external Docker networks without provisioning.
- [ ] Add GitHub CI and safe smoke checks for Go and Compose config.

## P1 — first complete vertical slice of parental DNS
- [ ] Design and sign off per-device DNS identity ADR (DHCP lease churn; multiple routers; NAT; IPv6).
- [ ] Define policy domain and Postgres migrations: router, device, user, group, policy, rule, assignment, schedule, exception, API key, audit.
- [ ] Implement Keycloak admin OIDC/JWT validation and admin RBAC.
- [ ] Implement router enrollment, per-router API key hashing, revocation, rotation, rate limits and scopes.
- [ ] Implement CRUD for group/device/domain policy, schedules, allowlists, blocklists and auditable override.
- [ ] Implement policy evaluation in actual DNS request path, integrating recursor and caching/invalidation.
- [ ] Automated e2e: device A blocked and device B allowed for same name; schedule/precedence; ownership isolation; identity spoofing, lease renewal and backend outage.

## P2 — physical OpenWrt package
- [ ] Select reference hardware and exact supported OpenWrt version/arch; record firmware checksum and package manager.
- [ ] Split test emulator from installable package source; build native `.ipk` or `.apk` and verify package metadata/dependencies.
- [ ] Implement UCI config, `procd` daemon, LuCI settings; secure bootstrap with backend, heartbeat, device inventory and DNS config.
- [ ] Transactional config apply and rollback; restore previous DNS/network settings on uninstall/failure.
- [ ] Test on physical router: cold boot, reboot, upgrade, uninstall, LAN Wi-Fi, leases, IPv4/IPv6, TLS, WAN outage, DNS outage.

## P3 — separate management frontend
- [ ] Choose framework/UX design; implement accessible responsive SPA behind Traefik.
- [ ] Keycloak login via Authorization Code + PKCE, short-lived tokens and protected routes.
- [ ] Screens for routers/devices/groups, filter lists/domains, schedules, temporary exceptions, per-device policy assignments and audit/status.
- [ ] Separate basic dashboard for households from operator-level Grafana.
- [ ] E2E workflow from saving web policy to enforced DNS on a real client.

## P4 — observability and delivery quality
- [ ] Integrate Prometheus/Grafana/Jaeger into canonical Compose with correct scrape targets, sensible retention and protected access.
- [ ] Backend metrics for DNS request latency, allowed/blocked counts, errors, Keycloak/auth failure, API readiness, DB and router heartbeat.
- [ ] Traces for authenticated HTTP requests, with privacy scrubbing. DNS telemetry may use metrics rather than full per-query trace.
- [ ] Load/soak tests and recovery tests; document performance constraints for chosen router.
- [ ] Complete README, OPERATIONS, troubleshooting, backups/restores, compatibility matrix, release and rollback process.

## MVP acceptance definition
- Backend starts on **one Docker host** from sanitized documented configuration, no secrets in Git.
- Two authenticated control paths: human admin via Keycloak and registered router via revocable API key.
- Distinct devices demonstrably receive distinct parental DNS policies, including schedules and exceptions, from the independent frontend.
- Tested router package installs/reboots/upgrades/uninstalls on selected physical OpenWrt model without disabling basic connectivity.
- Prometheus/Grafana/Jaeger give meaningful protected backend observability; no monitoring outage breaks DNS.
- Automated unit/integration/e2e and manual physical-hardware smoke matrix pass.
