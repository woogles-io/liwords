#!/usr/bin/env bash


echo "Starting buf generate"
buf generate
chown -R unprivileged:unprivileged pkg/gen

echo "Starting sqlc generate"
sqlc generate
chown -R unprivileged:unprivileged pkg/stores/models

# allow anyone to modify these files in the host.
chmod -R o+w liwords-ui/src/gen
chmod -R o+w pkg/gen
chmod -R o+w pkg/stores/models

echo "Done. Thank you for using our generator."
