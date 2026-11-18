#!/usr/bin/env bash
# Builds the backend images and pushes them to ghcr.io.
#
# The Go binaries are compiled once, here on the host, in a single `go build`
# so the packages they share are compiled once and the runner's Go build
# cache (restored by actions/setup-go) is used. The Dockerfiles only copy the
# finished binaries into debian. Compiling inside Docker instead meant a
# cold build of the whole module per image, about five minutes in all.
set -euo pipefail

cd "$(dirname "$0")/.."

TAG="${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER}"
BUILD_DATE=$(date -Iseconds -u)

# The build context: just the binaries and, for the API, db/ (migrations).
CTX=$(mktemp -d)
trap 'rm -rf "$CTX"' EXIT

# Static binaries: only the standard library's net and os/user use cgo, and
# both have pure-Go fallbacks. This keeps the binaries independent of the
# runner's glibc.
CGO_ENABLED=0 go build \
  -ldflags "-X=main.BuildDate=${BUILD_DATE} -X=main.BuildHash=${GITHUB_SHA}" \
  -o "$CTX/" \
  ./cmd/liwords-api ./cmd/puzzlegen ./cmd/socketsrv
cp -r db "$CTX/db"

build() { # image dockerfile
  docker build -t "ghcr.io/woogles-io/$1:${TAG}" -f "deploy/$2" "$CTX"
}
build liwords-api Dockerfile-apiserver
build liwords-puzzlegen Dockerfile-puzzlegen
build liwords-socket Dockerfile-socketsrv

pids=()
for image in liwords-api liwords-puzzlegen liwords-socket; do
  docker push -q "ghcr.io/woogles-io/${image}:${TAG}" &
  pids+=($!)
done
for pid in "${pids[@]}"; do
  wait "$pid"
done
