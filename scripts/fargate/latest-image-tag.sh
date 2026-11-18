#!/usr/bin/env bash
# Prints the image tag of the newest master build that pushed its images and
# ran its migration, e.g. master-gh2340: what a deploy should normally ship.
#
#   scripts/fargate/latest-image-tag.sh
#
# Builds that changed only the frontend push no images, and builds still
# running, or whose images or migration failed, are skipped. Needs gh, logged
# in (GH_TOKEN in Actions) with read access to the repository's Actions.
set -euo pipefail

REPO="${GITHUB_REPOSITORY:-woogles-io/liwords}"

runs=$(gh run list --repo "$REPO" --workflow build-and-deploy.yml --branch master \
  --event push --limit 30 --json databaseId,number -q '.[] | "\(.databaseId) \(.number)"')

while read -r id number; do
  [[ -n "$id" ]] || continue
  ok=$(gh run view "$id" --repo "$REPO" --json jobs -q '
    [.jobs[] | select(.name == "deploy_api_docker" or .name == "run_db_migration")
             | select(.conclusion == "success")] | length')
  if [[ "$ok" == 2 ]]; then
    echo "master-gh${number}"
    exit 0
  fi
done <<< "$runs"

echo "no master build in the last 30 pushed images and migrated" >&2
exit 1
