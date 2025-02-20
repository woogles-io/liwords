#!/usr/bin/env bash
set -e

DB_PATH="db/migrations"

if [[ "$OSTYPE" == "darwin"* ]]; then
    USER_GROUP="$USER:staff"
else
    USER_GROUP="unprivileged:unprivileged"
fi

docker compose run --rm goutils sh -c "
    migrate -database 'postgres://postgres:pass@db:5432/liwords?sslmode=disable' \
    -verbose create -dir db/migrations -format 200601021504 -ext sql $1 && \
    chown -R \"$USER_GROUP\" \"$DB_PATH\""