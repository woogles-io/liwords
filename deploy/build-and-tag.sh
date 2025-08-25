#!/usr/bin/env bash
set -e

docker build -t liwords-builder  -f Dockerfile-builder ..
docker build --build-arg BUILD_HASH=${GITHUB_SHA} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-api:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER} \
          -f Dockerfile-apiserver .
docker push ghcr.io/woogles-io/liwords-api:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER}

docker build --build-arg BUILD_HASH=${GITHUB_SHA} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-puzzlegen:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER} \
          -f Dockerfile-puzzlegen .
docker push ghcr.io/woogles-io/liwords-puzzlegen:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER}

docker build --build-arg BUILD_HASH=${GITHUB_SHA} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-maintenance:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER} \
          -f Dockerfile-maintenance .
docker push ghcr.io/woogles-io/liwords-maintenance:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER}

docker build --build-arg BUILD_HASH=${GITHUB_SHA} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-socket:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER} \
          -f Dockerfile-socketsrv ../services/socketsrv
docker push ghcr.io/woogles-io/liwords-socket:${GITHUB_REF_NAME}-gh${GITHUB_RUN_NUMBER}
