#!/usr/bin/env bash
set -e

# Paths
RPC_PATH="rpc"
MODELS_PATH="pkg/stores/models"
UI_GEN_PATH="liwords-ui/src/gen"

if [[ "$OSTYPE" == "darwin"* ]]; then
    USER_GROUP="$USER:staff"
else
    USER_GROUP="unprivileged:unprivileged"
fi

# Functions
buf_generate() {
  echo "Starting buf generate"
  (
    cd api || exit
    buf generate
  )
  chown -R "$USER_GROUP" "$RPC_PATH"
  chown -R "$USER_GROUP" "$UI_GEN_PATH"
}

sqlc_generate() {
  echo "Starting sqlc generate"
  sqlc generate
  chown -R "$USER_GROUP" "$MODELS_PATH"
}

set_permissions() {
  echo "Setting permissions"
  chmod -R o+w "$UI_GEN_PATH"
  chmod -R o+w "$RPC_PATH"
  chmod -R o+w "$MODELS_PATH"
}

# Check if necessary commands exist
command -v buf >/dev/null 2>&1 || { echo "buf command not found. Please install buf."; exit 1; }
command -v sqlc >/dev/null 2>&1 || { echo "sqlc command not found. Please install sqlc."; exit 1; }

# Main script

buf_generate
sqlc_generate
# set_permissions

echo "Done. Thank you for using our generator."