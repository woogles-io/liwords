#!/usr/bin/env bash
# Runs the entire hybrid dev stack in a single terminal:
#   - docker services (postgres, redis, NATS) via dc-local-services.yml
#   - the liwords API server (go)
#   - the socket server (go)
#   - the frontend dev server (rsbuild, port 3000)
#   - optionally, the macondo bot (pass --bot)
#
# Ctrl-C stops the local processes; the docker services are left running
# (stop them with `docker compose -f dc-local-services.yml stop`).
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ ! -f local.env ]]; then
  echo "local.env not found." >&2
  echo "Copy local_skeleton.env to local.env and adjust the _PATH variables first." >&2
  exit 1
fi
source local.env

WITH_BOT=0
for arg in "$@"; do
  case "$arg" in
    --bot) WITH_BOT=1 ;;
    *) echo "unknown argument: $arg (supported: --bot)" >&2; exit 1 ;;
  esac
done

if [[ "$WITH_BOT" == 1 && ! -d ../macondo ]]; then
  echo "--bot requires the macondo repo checked out next to liwords." >&2
  exit 1
fi

echo "==> Starting docker services (postgres, redis, NATS)..."
docker compose -f dc-local-services.yml up -d nats redis db

printf "==> Waiting for postgres to be ready"
until docker compose -f dc-local-services.yml exec -T db pg_isready -U postgres -q 2>/dev/null; do
  printf "."
  sleep 1
done
echo " ready."

if [[ ! -d liwords-ui/node_modules ]]; then
  echo "==> Installing frontend dependencies (first run)..."
  (cd liwords-ui && npm install)
fi

CYAN=$'\033[36m'; MAGENTA=$'\033[35m'; YELLOW=$'\033[33m'; GREEN=$'\033[32m'; RESET=$'\033[0m'
label() { sed -e "s/^/$1 /"; }

echo "==> Starting api, socket, and frontend. Ctrl-C stops everything."

go run cmd/liwords-api/*.go 2>&1 | label "${CYAN}[api]${RESET}   " &
go run cmd/socketsrv/main.go 2>&1 | label "${MAGENTA}[socket]${RESET}" &
(cd liwords-ui && npm start) 2>&1 | label "${YELLOW}[fe]${RESET}    " &

if [[ "$WITH_BOT" == 1 ]]; then
  (cd ../macondo && MACONDO_NATS_URL=nats://localhost:4222 go run cmd/bot/*.go) \
    2>&1 | label "${GREEN}[bot]${RESET}   " &
fi

cleanup() {
  trap - INT TERM
  echo
  echo "==> Shutting down local processes..."
  # Kill our whole process group (api, socket, frontend, bot, and the sed
  # labelers). Docker services stay up.
  kill 0
}
trap cleanup INT TERM

echo "==> Woogles will be at http://localhost:3000 once the frontend compiles."
wait
