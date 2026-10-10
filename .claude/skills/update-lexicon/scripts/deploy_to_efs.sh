#!/usr/bin/env bash
# Copy a new lexicon's files onto the production EFS data volume.
#
#   deploy_to_efs.sh NEW OLD            # dry run: show what would be copied
#   deploy_to_efs.sh NEW OLD --apply    # do it
#
# Layout on woogles-wg (EFS mounted at /mnt/efs, data at /mnt/efs/data):
#   lexica/gaddag/2024/NEW.{kwg,kad,klv2}, super-NEW.klv2   word graphs + leaves
#   strategy/NEW/leaves.klv2, super-leaves.klv2             legacy leaves folder
#   lexica/NEW.wmp                                           macondo word map (not handled here)
#   lexica/words/NEW.txt                                     definitions (not handled here)
#
# Never overwrites or deletes anything: it refuses if a target already exists.
set -euo pipefail

NEW=${1:?usage: deploy_to_efs.sh NEW OLD [--apply]}
OLD=${2:?usage: deploy_to_efs.sh NEW OLD [--apply]}
APPLY=${3:-}
HOST=woogles-wg
DATA=/mnt/efs/data
WASM="$(git rev-parse --show-toplevel)/liwords-ui/public/wasm/2024"

files=("$NEW.kwg" "$NEW.kad" "$NEW.klv2")
[[ -f "$WASM/super-$NEW.klv2" ]] && files+=("super-$NEW.klv2")
for f in "${files[@]}"; do
  [[ -f "$WASM/$f" ]] || { echo "missing local $WASM/$f" >&2; exit 1; }
done

echo "Local files (md5):"
(cd "$WASM" && md5 -r "${files[@]}")

echo; echo "Remote state:"
ssh "$HOST" "cd $DATA && ls -la lexica/gaddag/2024/$OLD.* strategy/$OLD/ && \
  for f in ${files[*]}; do test -e lexica/gaddag/2024/\$f && echo \"EXISTS: lexica/gaddag/2024/\$f\"; done; \
  test -e strategy/$NEW && echo \"EXISTS: strategy/$NEW\"; true"

if [[ "$APPLY" != "--apply" ]]; then
  echo; echo "Dry run. Would copy ${files[*]} to $HOST:$DATA/lexica/gaddag/2024/"
  echo "and $NEW.klv2 -> strategy/$NEW/leaves.klv2 (super-$NEW.klv2 -> super-leaves.klv2)."
  echo "Re-run with --apply to do it."
  exit 0
fi

STAGE=/tmp/lexicon-$NEW
ssh "$HOST" "mkdir -p $STAGE"
(cd "$WASM" && scp "${files[@]}" "$HOST:$STAGE/")
ssh "$HOST" bash -s <<EOF
set -euo pipefail
cd $DATA
for f in ${files[*]}; do
  if [ -e lexica/gaddag/2024/\$f ]; then echo "refusing: lexica/gaddag/2024/\$f exists" >&2; exit 1; fi
done
if [ -e strategy/$NEW ]; then echo "refusing: strategy/$NEW exists" >&2; exit 1; fi
for f in ${files[*]}; do cp -n $STAGE/\$f lexica/gaddag/2024/\$f; done
mkdir strategy/$NEW
cp -n $STAGE/$NEW.klv2 strategy/$NEW/leaves.klv2
if [ -e $STAGE/super-$NEW.klv2 ]; then cp -n $STAGE/super-$NEW.klv2 strategy/$NEW/super-leaves.klv2; fi
rm -rf $STAGE
echo "Remote md5:"
md5sum lexica/gaddag/2024/$NEW.* lexica/gaddag/2024/super-$NEW.klv2 strategy/$NEW/* 2>/dev/null || true
EOF
echo; echo "Compare the remote md5s above with the local ones."
