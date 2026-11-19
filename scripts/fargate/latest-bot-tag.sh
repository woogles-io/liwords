#!/usr/bin/env bash
# Prints the newest macondo-bot image built from macondo's master, e.g.
# ghcr.io/domino14/macondo-bot:master-gh574.
#
#   scripts/fargate/latest-bot-tag.sh
#
# The image is pushed by the push_docker job of macondo's "Build and Deploy"
# workflow; runs where that job didn't succeed are skipped. macondo is
# public, so any logged-in gh (GH_TOKEN in Actions) can read its runs.
set -euo pipefail

REPO=domino14/macondo

runs=$(gh run list --repo "$REPO" --workflow build-and-deploy-bot.yml --branch master \
  --event push --limit 30 --json databaseId,number -q '.[] | "\(.databaseId) \(.number)"')

while read -r id number; do
  [[ -n "$id" ]] || continue
  ok=$(gh run view "$id" --repo "$REPO" --json jobs -q '
    [.jobs[] | select(.name == "push_docker" and .conclusion == "success")] | length')
  if [[ "$ok" == 1 ]]; then
    echo "ghcr.io/domino14/macondo-bot:master-gh${number}"
    exit 0
  fi
done <<< "$runs"

echo "no macondo master build in the last 30 pushed a bot image" >&2
exit 1
