# Stack Docker unificado

El archivo `../docker-compose.yml` es el punto de entrada único para el backend
y RouterEase Admin. Reutiliza los archivos de `API/` para no duplicar servicios,
redes, volúmenes ni configuración de Traefik.

## Primer arranque de desarrollo

Desde la raíz del repositorio:

```sh
cp API/.env.example API/.env
# Editar API/.env: reemplazar todos los valores de ejemplo por secretos reales
# y configurar OIDC_ISSUER y OIDC_AUDIENCE para el realm/cliente de Keycloak.
docker compose config --quiet
docker compose up --build
```

`API/.env` no se versiona. Los volúmenes `policy_db_data` y
`keycloak_db_data` conservan las políticas y la configuración de Keycloak entre
reinicios. No use `docker compose down -v` salvo que quiera eliminar los datos
del entorno local.

El perfil opcional del túnel se inicia explícitamente:

```sh
docker compose --profile cloudflare up --build
```

## Servicios y accesos

Traefik expone HTTP/HTTPS en los puertos 80 y 443. Con el host configurado en
las reglas de Traefik, las rutas son:

- `/routerease/admin/`: panel React RouterEase Admin.
- `/routerease/dns/`: API de políticas DNS.
- `/routerease/user/`: Keycloak.

PostgreSQL y PowerDNS permanecen en la red interna de Docker; la resolución DNS
no se publica como si fuera una ruta HTTP de Traefik. Para el laboratorio DNS,
use los contratos y pruebas documentados en `docs/OPERATIONS-DNS-POLICY.md`.

Antes de publicar el stack, configure TLS, el hostname de Traefik/Keycloak y un
cliente público OIDC con Authorization Code + PKCE. Los valores de ejemplo no
son aptos para producción.
