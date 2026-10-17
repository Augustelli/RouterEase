# Arranque del entorno de desarrollo

Esta guía inicia el backend Docker de RouterEase en un único host. Es un entorno
de desarrollo: Keycloak usa `start-dev` y la API existente aún no implementa
autorización completa para sus CRUD. No lo expongas a Internet sin Cloudflare
Access y sin completar los controles de autorización pendientes.

## Requisitos

- Docker Engine y Docker Compose v2.
- Puertos TCP `80` y `443` libres en el host.
- Para acceso público: zona `routerease.com.ar` en Cloudflare y un Tunnel
  administrado remotamente.

## Primera configuración

Desde `API/`:

```sh
cp .env.example .env
chmod 600 .env
```

Edita `.env` y reemplaza estos valores con secretos distintos y aleatorios:

- `POLICY_DB_PASSWORD`
- `KEYCLOAK_DB_PASSWORD`
- `KEYCLOAK_ADMIN_PASSWORD`
- `TRAEFIK_DASHBOARD_PASSWORD` (distinta de la anterior)

No subas `.env` al repositorio. Las bases usan volúmenes Docker con nombre;
`docker compose down` no borra esos datos.

## Arranque

Sin túnel, para pruebas locales:

```sh
./scripts/start.sh --without-cloudflare
```

Con túnel, añade `CLOUDFLARE_TUNNEL_TOKEN` a `.env` y ejecuta:

```sh
./scripts/start.sh --with-cloudflare
```

Sin argumentos, `./scripts/start.sh` inicia el túnel sólo si detecta un token
no vacío. El script valida el Compose y nunca muestra valores secretos.

Comprueba el estado:

```sh
docker compose ps
docker compose logs --tail=100 reverse_proxie cloudflared
```

## Rutas de prueba

Traefik sólo enruta peticiones con host `dev.routerease.com.ar`. Para comprobar
la ruta local sin DNS público:

```sh
curl -fsS -H 'Host: dev.routerease.com.ar' \
  'http://127.0.0.1/routerease/mac-address/?mac=00:00:5e:00:53:01'
```

Con el Tunnel y el DNS configurados, usa la ruta pública equivalente:

```sh
curl -fsS 'https://dev.routerease.com.ar/routerease/mac-address/?mac=00:00:5e:00:53:01'
```

Las superficies actuales son:

| Ruta | Servicio |
| --- | --- |
| `/routerease/mac-address/?mac=<MAC>` | MAC Lookup |
| `/routerease/dns/users/*`, `/groups/*`, `/domains/*`, `/dns-query` | API Go |
| `/routerease/user/` | Keycloak |
| `/admin/` | Dashboard de Traefik (Basic Auth) |

El dashboard redirige de `/admin` a `/admin/dashboard/`. Su usuario es `admin` por defecto
y la contraseña viene de `TRAEFIK_DASHBOARD_PASSWORD`; si esa variable no se
define en un `.env` antiguo, se usa temporalmente la contraseña bootstrap de
Keycloak. Cámbiala antes de compartir el acceso.

## Cloudflare Tunnel

En Zero Trust crea un Public Hostname para `dev.routerease.com.ar` dentro del
mismo Tunnel cuyo token está en `.env`. La URL de servicio debe ser
`http://localhost:80`: el conector comparte el espacio de red de Traefik y ese
puerto corresponde al entrypoint HTTP de Traefik.

Si Cloudflare devuelve `530` con código `1033`, el registro DNS de `dev` está
asociado a otro Tunnel. Elimina el Public Hostname/registro obsoleto y créalo
otra vez en el Tunnel activo. Los detalles están en
[CLOUDFLARE_TUNNEL.md](CLOUDFLARE_TUNNEL.md).

## Detener

```sh
./scripts/stop.sh
```

Esto detiene y elimina contenedores y redes del proyecto, pero conserva los
volúmenes PostgreSQL. Para borrar datos de desarrollo deliberadamente, lista
primero los volúmenes con `docker volume ls`; no uses ese borrado para una
instalación que deba conservar datos.
