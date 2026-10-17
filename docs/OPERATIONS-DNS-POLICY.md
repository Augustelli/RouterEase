# Operación del motor DNS de políticas

1. Copiar `API/.env.example` a `API/.env` y reemplazar todos los valores de
   ejemplo con secretos nuevos. Configurar un cliente Keycloak con audiencia
   `OIDC_AUDIENCE` y asignar el rol `routerease-admin`.
2. Desde `API/`, ejecutar `docker compose up -d --build`. PostgreSQL conserva
   los datos en `policy_db_data`; las migraciones SQL se aplican al iniciar Go.
3. Exponer la API por Traefik sólo como HTTPS. El endpoint es DoH HTTP/HTTPS;
   Traefik no publica DNS UDP/TCP 53. PowerDNS permanece en la red interna.

Pruebas de laboratorio:

```sh
docker run --rm -v "$PWD/GO-Proxie:/src" -w /src golang:1.24-alpine go test ./...
```

Las pruebas PostgreSQL y DoH requieren `TEST_DATABASE_URL`; el test
`TestAuthenticatedDoHBlocksOneDeviceAndForwardsAnother` inicia un resolver UDP
efímero y comprueba una respuesta NXDOMAIN para el dispositivo infantil y A
para el otro dispositivo, con el mismo nombre. No registra dominios, payloads
DNS ni cabeceras de autorización.

No es protección universal contra DNS externo, DoH/DoT propio, VPN, caché DNS
o conexiones directas por IP. Antes de producción física se necesita el agente
OpenWrt descrito en ADR-0001 y una remediación separada de secretos históricos.
