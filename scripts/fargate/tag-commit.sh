#!/usr/bin/env bash
# Prints the commit an image tag was built from: master-gh<N> is run N of
# build-and-deploy.yml. Exits 1 if it can't tell.
#
#   scripts/fargate/tag-commit.sh master-gh2342
set -euo pipefail

tag="${1:?usage: tag-commit.sh <tag>}"
REPO="${GITHUB_REPOSITORY:-woogles-io/liwords}"

n="${tag#master-gh}"
[[ "$tag" == master-gh* && "$n" =~ ^[0-9]+$ ]] || exit 1
sha=$(gh run list --repo "$REPO" --workflow build-and-deploy.yml --branch master --event push \
  --limit 300 --json number,headSha -q ".[] | select(.number == $n) | .headSha")
[[ -n "$sha" ]] || exit 1
echo "$sha"
