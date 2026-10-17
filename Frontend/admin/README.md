# RouterEase Admin — DNS Management

SPA React para administrar políticas DNS. Requiere el backend RouterEase, un
cliente OIDC público Keycloak con Authorization Code + PKCE y el rol
`routerease-admin` en el access token.

```sh
cp .env.example .env.local
npm ci
npm run dev
```

Para integrarla al stack, desde `API/`:

```sh
docker compose -f docker-compose.yml -f docker-compose.admin.yml up -d --build
```

La UI queda en `/routerease/admin/`. Se eligió esa ruta porque `/admin/` está
reservada por el dashboard de Traefik en la configuración actual. Las URLs
OIDC se inyectan al compilar: son configuración pública, no secretos.

Pruebas: `npm run test` y `npm run build`.
