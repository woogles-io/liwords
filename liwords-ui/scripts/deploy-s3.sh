#!/usr/bin/env bash
# Deploys a frontend build (liwords-ui/dist) to the woogles.io bucket, which
# CloudFront serves, and deletes the files no deploy has served for a while.
#
#   liwords-ui/scripts/deploy-s3.sh <dist dir> <ui hash>
#   DRY_RUN=1 liwords-ui/scripts/deploy-s3.sh dist test   # writes nothing
#
# The order is what keeps a deploy safe for people loading the site while it
# runs: index.html names the files it needs, so everything else is uploaded
# before it, and it goes last.
#
# Files and how long browsers and CloudFront may keep them:
# - static/: the bundler's output. Each name carries a hash of its content,
#   so a name never changes content: cached for a year, and a file the bucket
#   already has is not sent again.
# - wasm/: the analyzer's word lists, which are never changed under the same
#   name either (see upload-wasm.mjs): 35 days, as upload-wasm.mjs sets, and
#   not sent again.
# - index.html and the other .html, .js, .json and .txt files at the top
#   (config.js, sw.js, manifest.json, robots.txt): revalidated on every load,
#   so a deploy takes effect at once.
# - Everything else (images, fonts, favicon): a day.
#
# Expiry: each deploy records the files it serves in _deploys/<time>.txt. A
# file stays while any deploy that served it was live within RETAIN_DAYS (30
# days by default) - an open tab from that deploy can still ask for the
# chunks it hasn't loaded yet - and is deleted after. Files no record names
# (from before these records, or uploaded by hand) are deleted once they are
# RETAIN_DAYS old and the current deploy doesn't serve them. Nothing uploaded
# within RETAIN_DAYS is ever deleted, and neither is the analyzer's current
# dir (wasm/<dir>/ of wasm-files.json), which upload-wasm.mjs fills by hand.
#
# Needs the aws CLI, with credentials that may read, write and delete in the
# bucket, and jq.
set -euo pipefail

dist="${1:?usage: deploy-s3.sh <dist dir> <ui hash>}"
hash="${2:?usage: deploy-s3.sh <dist dir> <ui hash>}"
BUCKET="${BUCKET:-woogles.io}"
RETAIN_DAYS="${RETAIN_DAYS:-30}"
DRY_RUN="${DRY_RUN:-0}"

ui=$(cd "$(dirname "$0")/.." && pwd)
analyzer="wasm/$(jq -r .dir "$ui/wasm-files.json")/"
now=$(date -u +%Y%m%dT%H%M%SZ)
cutoff=$(date -u -d "$RETAIN_DAYS days ago" +%Y%m%dT%H%M%SZ)
cutoff_iso=$(date -u -d "$RETAIN_DAYS days ago" +%Y-%m-%dT%H:%M:%S)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

YEAR='public, max-age=31536000, immutable'
WASM='max-age=3024000'
DAY='public, max-age=86400'
ALWAYS='no-cache'

run() {
  if [[ "$DRY_RUN" == 1 ]]; then
    echo "  would run: $*"
  else
    "$@"
  fi
}

cd "$dist"
[[ -f index.html ]] || { echo "no index.html in $dist" >&2; exit 1; }
find . -type f | sed 's|^\./||' | LC_ALL=C sort > "$work/deployed.txt"
echo "Deploying $(wc -l < "$work/deployed.txt") files to s3://$BUCKET (ui hash $hash)"

meta=(--metadata "FEHash=$hash")

echo "== static/ (new files only)"
if [[ -d static ]]; then
  run aws s3 sync static "s3://$BUCKET/static" --size-only --no-progress \
    --cache-control "$YEAR" "${meta[@]}" --exclude '*.wasm'
  run aws s3 sync static "s3://$BUCKET/static" --size-only --no-progress \
    --cache-control "$YEAR" "${meta[@]}" --exclude '*' --include '*.wasm' \
    --content-type application/wasm
fi

echo "== wasm/ (new files only)"
if [[ -d wasm ]]; then
  run aws s3 sync wasm "s3://$BUCKET/wasm" --size-only --no-progress \
    --cache-control "$WASM" "${meta[@]}"
fi

echo "== other directories"
for d in */; do
  d="${d%/}"
  [[ "$d" == static || "$d" == wasm ]] && continue
  run aws s3 sync "$d" "s3://$BUCKET/$d" --no-progress \
    --cache-control "$DAY" "${meta[@]}"
done

echo "== top-level files, index.html last"
for f in *; do
  [[ -f "$f" && "$f" != index.html ]] || continue
  case "$f" in
    *.html | *.js | *.json | *.txt) cc="$ALWAYS" ;;
    *) cc="$DAY" ;;
  esac
  run aws s3 cp "$f" "s3://$BUCKET/$f" --no-progress --cache-control "$cc" "${meta[@]}"
done
run aws s3 cp index.html "s3://$BUCKET/index.html" --no-progress \
  --cache-control "$ALWAYS" "${meta[@]}"

echo "== recording this deploy as _deploys/$now.txt"
run aws s3 cp "$work/deployed.txt" "s3://$BUCKET/_deploys/$now.txt" --no-progress \
  --content-type text/plain --cache-control no-store

echo "== expiring files no deploy has served for $RETAIN_DAYS days"
aws s3api list-objects-v2 --bucket "$BUCKET" --output json \
  --query 'Contents[].{k: Key, t: LastModified}' > "$work/objects.json"
jq -r '.[].k | select(startswith("_deploys/"))' "$work/objects.json" \
  | LC_ALL=C sort > "$work/records.txt"

# A deploy was live from its own time until the next one's. Keep what the
# deploys live since the cutoff served; drop the records of older ones.
cp "$work/deployed.txt" "$work/keep.txt"
: > "$work/expired-records.txt"
mapfile -t records < "$work/records.txt"
for i in "${!records[@]}"; do
  rec="${records[$i]}"
  [[ "$rec" == "_deploys/$now.txt" ]] && continue
  next="${records[$((i + 1))]:-_deploys/$now.txt}"
  until_t=$(basename "$next" .txt)
  if [[ "$until_t" > "$cutoff" ]]; then
    aws s3 cp "s3://$BUCKET/$rec" - >> "$work/keep.txt"
  else
    echo "$rec" >> "$work/expired-records.txt"
  fi
done

jq -r --rawfile keep "$work/keep.txt" --arg cutoff "$cutoff_iso" --arg analyzer "$analyzer" '
  ($keep | split("\n") | map(select(. != "") | {key: ., value: true}) | from_entries) as $k
  | .[]
  | select(.k | startswith("_deploys/") | not)
  | select(.k | startswith($analyzer) | not)
  | select($k[.k] | not)
  | select(.t[0:19] < $cutoff)
  | .k' "$work/objects.json" > "$work/delete.txt"
cat "$work/expired-records.txt" >> "$work/delete.txt"

total=$(jq length "$work/objects.json")
count=$(grep -c . "$work/delete.txt" || true)
echo "Bucket has $total files; deleting $count (records kept: $(( $(grep -c . "$work/records.txt" || true) - $(grep -c . "$work/expired-records.txt" || true) )))"
sed 's/^/  /' "$work/delete.txt" | head -50
(( count > 50 )) && echo "  ... and $((count - 50)) more"

if (( count > 0 )); then
  split -l 1000 "$work/delete.txt" "$work/batch."
  for b in "$work"/batch.*; do
    jq -R -s 'split("\n") | map(select(. != "") | {Key: .}) | {Objects: ., Quiet: true}' "$b" > "$b.json"
    run aws s3api delete-objects --bucket "$BUCKET" --delete "file://$b.json" --output json
  done
fi
echo "Done."
