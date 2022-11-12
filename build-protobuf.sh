#!/usr/bin/env bash

export PATH="/opt/node_modules/.bin":$PATH

cd api/proto
buf generate
buf generate buf.build/domino14/macondo --template buf.gen.macondo.yaml
cd -

chown -R unprivileged:unprivileged liwords-ui/src/gen
chown -R unprivileged:unprivileged rpc
# allow anyone to modify these files in the host.
chmod -R o+w liwords-ui/src/gen
chmod -R o+w rpc