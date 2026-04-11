#!/usr/bin/env bash
set -euo pipefail

if [ $# -eq 0 ]; then
  docker compose exec -it db psql -U postgres liwords
else
  docker compose exec -T db psql -U postgres liwords -c "$1"
fi
