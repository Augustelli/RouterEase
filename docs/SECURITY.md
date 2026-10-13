# Security baseline and remediation plan

**Current status**: audit indicates credentials and runtime state are committed in public `dev-api`. Treat values as compromised. Avoid quoting existing secrets in issues, docs and chat transcripts.

## Mandatory P0 actions
1. Inventory all secrets: Keycloak admin and database passwords, policy DB password, router admin/Wi-Fi example credentials, ACME private key/certificate material, token/client secrets, environment files and embedded URLs. Rotate/revoke originals after migrating to protected injection. Test old values no longer grant access.
2. Remove tracked runtime directories/files from future commits: `API/Authentication/keycloak-data/`, `API/GO-Proxie/dns-volumen/`, `API/ReverseProxie/acme.json`, `API/ReverseProxie/logs/`, any generated certificates, logs, tokens and database snapshots. Use named Docker volumes and secured backups.
3. This alone **does not clean Git history**. Decide history rewrite/force-push after coordination and backup; rotate secrets irrespective of history cleanup. Coordinate CI, deployments and collaborators after rewrite.
4. Lock down Traefik dashboard/API; remove/limit Docker socket access; define specific CORS origins and never wildcard origins with credentialed requests; enable HTTPS consistently.
5. Restrict PowerDNS `allow-from` and upstream access; ensure public resolver abuse and open-recursion are not possible. Restrict database and observability service binds.
6. Keycloak production mode, secured bootstrap, issuer/hostname configuration and separate OIDC client setup. Frontend must not use resource-owner-password grant (`grant_type=password`) or embed admin CLI credentials.
7. Store router API keys only at secure router config and server-side hashed representation; per-router scopes, rotation, revocation, audit. Provide safe recovery for lost router credentials.
8. Remove logging of entire DNS request bodies and HTTP headers; sanitize personal device identifiers and sensitive domain/query data; bound retention.

## Authorization and DNS
- JWT validation uses trusted issuer/JWKS, expected audience/client IDs, signature algorithm, expiry, nbf, roles/ownership; no UI-only auth checks.
- Router API keys authenticate infrastructure; do not infer client identity from one router key. Metadata such as MAC/IP must be corroborated with local router observations; preserve router scope and device lease recency.
- Define clear rule precedence, exception behavior and deliberate DNS fail-open/fail-closed strategy. Test bypass limitations (external DoH/DoT, apps with hardcoded resolvers, VPNs, IPv6).
- Network/FW/UCI changes on real routers need backup/rollback. Test isolated before enabling on users' LAN.

## Development process
- No actual production secret in `.env.example`.
- Add `gitleaks` or equivalent secret scan, static analysis and dependency checks to CI.
- Review generated runtime directories and `git status --ignored` before committing.
- Do not claim P0 is complete until credentials have been rotated and runtime artifacts are untracked and no longer served.
