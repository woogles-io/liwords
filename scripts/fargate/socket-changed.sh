#!/usr/bin/env bash
# Exits 0 if the socket server must be redeployed to go from one image tag to
# another, 1 if it would run the same code.
#
#   scripts/fargate/socket-changed.sh master-gh2337 master-gh2342
#
# It builds cmd/socketsrv from both commits, with the same Go and without
# anything that differs per build (paths, VCS stamp, build ID, the BuildDate
# and BuildHash variables), and compares the binaries. The linker leaves out
# code the socket server never calls, so a change to a package it imports but
# doesn't use (the sqlc models, say) gives the same binary and no deploy. A
# change to its Dockerfile or template is a change too.
#
# The tags are master-gh<run number> of build-and-deploy.yml. A tag it can't
# map to a commit, or a build that fails, counts as changed. Needs gh, Go and
# the full git history.
set -euo pipefail

old="${1:?usage: socket-changed.sh <deployed tag> <new tag>}"
new="${2:?usage: socket-changed.sh <deployed tag> <new tag>}"

[[ "$old" == "$new" ]] && { echo "same tag $new"; exit 1; }

here=$(dirname "$0")
old_sha=$("$here/tag-commit.sh" "$old" || true); new_sha=$("$here/tag-commit.sh" "$new" || true)
if [[ -z "$old_sha" || -z "$new_sha" ]]; then
  echo "can't find the commits of $old and $new; treating the socket server as changed"
  exit 0
fi

other=$(git diff --name-only "$old_sha" "$new_sha" -- deploy/Dockerfile-socketsrv aws/cfn/fargate/liwords-socket.yaml)
if [[ -n "$other" ]]; then
  echo "socket server changed between $old and $new:"
  echo "$other" | sed 's/^/  /'
  exit 0
fi

work=$(mktemp -d)
trap 'git worktree remove --force "$work/src" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

build() { # commit output
  rm -rf "$work/src"
  git worktree add -q --detach "$work/src" "$1"
  (cd "$work/src" && CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "-buildid=" \
    -o "$2" ./cmd/socketsrv)
  git worktree remove --force "$work/src"
}
if ! build "$old_sha" "$work/old" || ! build "$new_sha" "$work/new"; then
  echo "couldn't build the socket server at both commits; treating it as changed"
  exit 0
fi

if cmp -s "$work/old" "$work/new"; then
  echo "the socket server is the same binary at $old and $new"
  exit 1
fi
echo "the socket server's binary differs between $old and $new"
exit 0
