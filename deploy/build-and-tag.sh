#!/usr/bin/env bash
set -e

echo $GHCR_TOKEN | docker login ghcr.io -u domino14 --password-stdin
docker build -t liwords-builder  -f Dockerfile-builder ..
docker build --build-arg BUILD_HASH=${CIRCLE_SHA1} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-api:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM} \
          -f Dockerfile-apiserver .
docker push ghcr.io/woogles-io/liwords-api:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM}

docker build --build-arg BUILD_HASH=${CIRCLE_SHA1} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-puzzlegen:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM} \
          -f Dockerfile-puzzlegen .
docker push ghcr.io/woogles-io/liwords-puzzlegen:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM}

docker build --build-arg BUILD_HASH=${CIRCLE_SHA1} \
          --build-arg BUILD_DATE=$(date -Iseconds -u) \
          -t ghcr.io/woogles-io/liwords-maintenance:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM} \
          -f Dockerfile-maintenance .
docker push ghcr.io/woogles-io/liwords-maintenance:${CIRCLE_BRANCH}-${CIRCLE_BUILD_NUM}
