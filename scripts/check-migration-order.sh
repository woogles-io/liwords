#!/bin/bash
# Checks that every migration a branch adds is newer than the newest one on
# its base.
#
# golang-migrate only applies versions above the one the database is already
# at. A migration numbered below master's newest is skipped in production
# without an error, so a branch that sat open while master moved on can merge
# a migration that never runs. Two files with the same version make the API
# panic at startup instead ("duplicate migration file").
#
# Usage: scripts/check-migration-order.sh <base> [<head>]
#   base  the ref the branch merges into (origin/master), or for a push to
#         master, the commit before the push
#   head  the branch tip; HEAD by default
#
# Only files the branch adds (since its merge base with <base>) are checked.
# Migrations already on master were applied in order, so their numbers no
# longer matter. A renumbered migration shows up as an added file, so a
# branch that fixes its numbering passes.

set -euo pipefail

base=${1:?usage: $0 <base> [<head>]}
head=${2:-HEAD}
dir=db/migrations

version_of() { basename "$1" | sed -n 's/^\([0-9][0-9]*\)_.*$/\1/p'; }

latest=$(git ls-tree --name-only "$base" "$dir/" |
    while read -r f; do version_of "$f"; done | sort -n | tail -1)

added=$(git diff --name-only --diff-filter=A "$base...$head" -- "$dir/*.sql")

if [ -z "$added" ]; then
    echo "No migrations added."
    exit 0
fi

# One entry per migration, not per .up/.down file.
names=$(for f in $added; do basename "$f" | sed 's/\.\(up\|down\)\.sql$//'; done | sort -u)

fail=0
declare -A seen

for name in $names; do
    v=$(version_of "$name")
    if [ -z "$v" ]; then
        echo "✗ $name: name does not start with a numeric version."
        fail=1
        continue
    fi
    # golang-migrate compares versions as numbers, so a longer one (say
    # YYYYMMDDHHmmss) sorts above every YYYYMMDDHHmm version and, once
    # applied, makes every later migration look old.
    if [ "${#v}" -ne 12 ]; then
        echo "✗ $name: version $v is not 12 digits (YYYYMMDDHHmm); a longer one would sort above every later migration."
        fail=1
        continue
    fi
    if [ -n "${seen[$v]:-}" ]; then
        echo "✗ $name and ${seen[$v]} share version $v; golang-migrate panics at startup on a duplicate version."
        fail=1
    fi
    seen[$v]=$name
    if [ -n "$latest" ] && [ "$v" -le "$latest" ]; then
        echo "✗ $name is not newer than the newest migration on the base ($latest); golang-migrate would skip it in production. Renumber it above $latest, e.g. $((latest + 1))."
        fail=1
    fi
done

if [ "$fail" -eq 0 ]; then
    echo "✓ All added migrations are newer than the base's newest ($latest):"
    printf '  %s\n' $names
fi
exit "$fail"
