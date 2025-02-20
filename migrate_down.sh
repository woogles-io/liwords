#!/usr/bin/env bash
set -e

docker compose run --rm goutils sh -c "
    migrate -database "postgres://postgres:pass@db:5432/liwords?sslmode=disable" \
    -source file://./db/migrations down $1"