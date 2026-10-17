#!/usr/bin/env sh
# Start RouterEase's development stack without printing environment secrets.
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage: ./scripts/start.sh [--with-cloudflare|--without-cloudflare]

Without an option, Cloudflare Tunnel starts only when CLOUDFLARE_TUNNEL_TOKEN
is set in .env. The script never prints secret values.
EOF
}

fail() {
  printf '%s\n' "Error: $*" >&2
  exit 1
}

mode=auto
case "${1:-}" in
  '') ;;
  --with-cloudflare) mode=with-cloudflare ;;
  --without-cloudflare) mode=without-cloudflare ;;
  --help|-h) usage; exit 0 ;;
  *) usage; exit 2 ;;
esac

command -v docker >/dev/null 2>&1 || fail "Docker is not installed."
docker compose version >/dev/null 2>&1 || fail "Docker Compose v2 is required."

if [ ! -f .env ]; then
  cp .env.example .env
  chmod 600 .env
  fail ".env was created from .env.example. Set the passwords and run again."
fi

env_value() {
  awk -v key="$1" 'index($0, key "=") == 1 { sub("^[^=]*=", ""); sub("\r$", ""); print; exit }' .env
}

require_value() {
  value=$(env_value "$1")
  case "$value" in
    ''|replace-with-*) fail "Set $1 in .env." ;;
  esac
}

require_value POLICY_DB_PASSWORD
require_value KEYCLOAK_DB_PASSWORD
require_value KEYCLOAK_ADMIN_PASSWORD
require_value OIDC_ISSUER
require_value OIDC_AUDIENCE

tunnel_token=$(env_value CLOUDFLARE_TUNNEL_TOKEN)
use_cloudflare=false
case "$mode" in
  with-cloudflare)
    [ -n "$tunnel_token" ] || fail "Set CLOUDFLARE_TUNNEL_TOKEN in .env."
    use_cloudflare=true
    ;;
  auto)
    [ -n "$tunnel_token" ] && use_cloudflare=true
    ;;
esac

docker compose config --quiet

if [ "$use_cloudflare" = true ]; then
  docker compose --profile cloudflare up -d --build
else
  # A profile service already running is not removed by `compose up` without
  # that profile, so stop it explicitly when the caller opted out.
  if [ "$mode" = without-cloudflare ]; then
    docker compose --profile cloudflare stop cloudflared >/dev/null 2>&1 || true
  fi
  docker compose up -d --build
fi

docker compose ps
printf '%s\n' ''
printf '%s\n' 'Local smoke test:'
printf '%s\n' "  curl -H 'Host: dev.routerease.com.ar' 'http://127.0.0.1/routerease/mac-address/?mac=00:00:5e:00:53:01'"
if [ "$use_cloudflare" = true ]; then
  printf '%s\n' 'Cloudflare Tunnel was started. Public hostname troubleshooting: docs/CLOUDFLARE_TUNNEL.md'
else
  printf '%s\n' 'Cloudflare Tunnel was not started (no token or --without-cloudflare).'
fi
