# API v1 — políticas DNS

Todas las rutas administrativas requieren `Authorization: Bearer <OIDC JWT>`.
El token se verifica contra `OIDC_ISSUER`/JWKS, audiencia `OIDC_AUDIENCE`,
firma, emisor, expiración y rol `routerease-admin`. Las rutas de router no
aceptan JWT humano.

| Operación | Ruta |
|---|---|
| Routers | `GET,POST /api/v1/routers`; `GET,PUT,DELETE /api/v1/routers/{id}` |
| API keys | `POST /api/v1/routers/{id}/keys`; `DELETE /api/v1/routers/{id}/keys/{keyId}` |
| Dispositivos | `GET,POST /api/v1/routers/{id}/devices`; `GET,PUT,DELETE /api/v1/devices/{id}` |
| Grupos | `GET,POST /api/v1/routers/{id}/groups`; `GET,PUT,DELETE /api/v1/groups/{id}` |
| Membresía | `POST,DELETE /api/v1/groups/{id}/devices/{deviceId}` |
| Políticas | `GET,POST /api/v1/policies`; `GET,PUT,DELETE /api/v1/policies/{id}` |
| Reglas | `GET,POST /api/v1/policies/{id}/rules`; `PUT,DELETE /api/v1/policies/{id}/rules/{ruleId}` |
| Asignaciones | `GET,POST /api/v1/policies/{id}/assignments`; `DELETE /api/v1/policies/{id}/assignments/{assignmentId}` |
| Política efectiva | `GET /api/v1/devices/{id}/effective-policy?domain=example.org` |
| Estado | `GET /api/v1/dns/status` |

El secreto de una API key de router aparece sólo en la respuesta de creación.
Rutas de router:

- `POST /api/v1/router/leases`: `X-Router-API-Key`; scope `inventory:write`;
  cuerpo `{ "leases": [{"device_id":"…","ip":"192.0.2.10","expires_in_seconds":300}] }`.
- `POST /api/v1/router/dns-query`: `X-Router-API-Key`; scope `dns:query`;
  `X-Router-Client-IP`; `Content-Type: application/dns-message`.

Las rutas legadas `/users/*`, `/groups/*`, `/domains/*` y `/dns-query` retornan
`410 Gone`: eran CRUD sin autenticación y no son compatibles de forma segura.

Las reglas son exactas o incluyen subdominios: `example.org` alcanza
`example.org` y `a.example.org`, nunca `badexample.org`. Entre reglas explícitas
un `ALLOW` es excepción y vence a `BLOCK`; después gana el sufijo más específico,
prioridad numérica menor y, finalmente, asignación directa a dispositivo antes
que grupo. Si no hay regla, se aplica la política con menor prioridad (directa
antes que grupo); si no hay asignación, se permite. Un bloqueo produce NXDOMAIN
con el ID DNS original preservado.
