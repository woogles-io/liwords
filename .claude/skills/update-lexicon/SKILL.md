---
name: update-lexicon
description: Add a new edition of an existing woogles lexicon (e.g. NSF25 -> NSF26, CSW21 -> CSW24) from a raw word list. It builds and verifies the .kwg/.kad with kwgc, reuses or accepts a .klv2, wires the lexicon into liwords, retires the old edition, opens the PR and walks through the production rollout. Use when the user says a new dictionary, word list or lexicon version is out, or asks to update, add or obsolete a lexicon.
---

# Updating a lexicon on woogles

This covers a **new edition of a lexicon whose alphabet woogles already supports**. A brand-new alphabet or letter distribution (for example Slovene in PR #1952 and #1955) is a different, multi-repo job. See `reference/new-alphabet.md` and stop to plan it with the user before touching anything.

Work carefully. The repo usually has unrelated untracked files, and the old edition's files are still needed by existing games. Several steps affect production and must never be done without explicit approval.

Throughout, `OLD` is the edition being replaced (e.g. `NSF25`), `NEW` is the new one (e.g. `NSF26`), and `WASM=liwords-ui/public/wasm/2024`. `data/lexica/gaddag` is a symlink to `WASM`, so local dev and CI read the same files.

## 0. Pin down the inputs

Work these out before building, and confirm them with the user in one message:

| Input | How to decide |
|---|---|
| Word list path | Given by the user. Look at it first: `head`, line count, encoding, case, and whether there's a second column (definitions). |
| `NEW` code | `<FAMILY><2-digit year>`, following the existing ones (`ls $WASM`). A file named `nsf2026.txt` becomes `NSF26`. |
| `OLD` code | The family's current entry in `AllowedNewGameLexica` (`pkg/entity/lexica.go`). |
| Alphabet | `tilemapping.ProbableLetterDistributionName` prefix map in word-golib, e.g. `NSF` → `norwegian`. It must exist in `pkg/memento/letterdistributions/` **and** as a kwgc tileset. If either is missing, this isn't a version update. |
| Leaves (`.klv2`) | Reuse `OLD.klv2` (copied as `NEW.klv2`) unless the user supplies new leaves. Same for `super-OLD.klv2` if it exists. |
| Length window | min 2 (no 1-tile plays exist). Max 15, **or 21 if `super-OLD.klv2` exists** (that lexicon is also played on the 21x21 SuperCrosswordGame board). |
| Obsolete OLD? | Default yes: OLD leaves new-game lists but stays loadable (see step 3). |

Earlier editions were not always filtered. `NSF25.kwg` contains 1-letter words and words up to 33 letters, so a big "fewer words" number compared with OLD's raw count is expected and not a problem. The diff below only compares words inside the length window.

## 1. Build and verify the word graphs

```bash
cd ~/code/kwgc && make   # only if ./kwgc is missing or stale
python3 .claude/skills/update-lexicon/scripts/build_lexicon.py \
  --input <wordlist> --name NEW --alphabet <alphabet> \
  --prev $WASM/OLD.kwg --max-len <15|21> \
  --out <scratchpad>/NEW
```

The script normalizes the list: NFC, uppercase, first column only, duplicates removed. It stops if any character is outside the alphabet, drops words outside the length window, and builds `NEW.kwg` and `NEW.kad` with kwgc. It then **reads them back and checks them exactly**:

- DAWG half of the kwg = the kept word set
- `.kad` = the set of per-word sorted-tile anagram keys
- GADDAG half of the kwg = every `reverse(prefix)?suffix` entry for every word

It also writes `report.json`, `dropped.txt`, `added.txt` and `removed.txt` (diff against OLD inside the length window). kwgc exits 0 even when it fails, so rely on the script's checks, never on kwgc's exit code.

**Show the user a summary and wait for an OK before going on.** Include: raw / kept / dropped (short, long, invalid) counts, the 1-letter words dropped, the blank-only word count (words using zero-count tiles such as Q/X/Z/Ä/Ö/Ü in Norwegian are valid and stay in), added and removed counts with examples, and the file sizes next to OLD's. Stop and ask if:
- any check failed
- `--drop-invalid` was needed
- removed > ~2% of OLD, or added is far outside normal edition-to-edition churn
- the kwg is much bigger or smaller than OLD's

## 2. Branch

Start from an up-to-date `master` on a new branch: `git fetch && git checkout -b NEW-lexicon origin/master && git branch --unset-upstream`. Without the unset, the branch tracks master. The working tree usually has unrelated untracked files (plans, binaries, test JSON). **Never `git add -A` or `git add .`.** Stage paths explicitly.

## 3. Put the files in place

Copy into `$WASM`: `NEW.kwg`, `NEW.kad`, `NEW.klv2` (from `OLD.klv2` or the supplied file), and `super-NEW.klv2` if OLD has a super table. Check with `md5` that the copies match their sources.

**Never delete, rename or overwrite OLD's files.** Existing games, annotated games, the analyzer, and macondo's download-on-demand (it fetches from `$WASM` on GitHub master) all load lexica by name.

## 4. Wire NEW into liwords

Grep for every mention first: `git grep -n 'OLD' -- ':!*.kwg' ':!*.kad' ':!*.klv2' ':!*_pb*'`. The usual edits (compare with commit `ddb1199d` for NSF25 and PR #1381 for CSW24):

- `pkg/entity/lexica.go`: in `AllowedNewGameLexica`, **replace** OLD with NEW. This list is the only thing stopping new games on OLD.
- `liwords-ui/src/shared/lexica.ts`: rename the `AllLexica` entry's key, `code`, and `matchName` if it's versioned. Update `shortDescription`, and the `longDescription` copyright year if it has one. **Do not touch** `lexiconCodeToInternalRatingName` or `InternalRatingNameToProfileRatingName`: ratings are keyed by family (`NSF*` → `NSF21`), so players keep their rating across editions. `pkg/entity/ratings.go` `transformLexiconName` is the same, so leave it alone.
- `liwords-ui/src/shared/lexicon_display.tsx`:
  - `lexiconOrder`: replace OLD with NEW.
  - `puzzleLexica`: replace OLD if it's there.
  - `historicalLexica`: **add OLD**, so annotated games and GCG imports can still use it.
- `liwords-ui/src/wasm/loader.ts`: add `NEW.kad`, `NEW.klv2`, `NEW.kwg` (and `super-NEW.klv2`) in sorted position. Keep OLD's entries. The browser loader needs a klv2 next to every kwg.
- `liwords-ui/src/lobby/seek_form.tsx`: the stored-form migration switch should send OLD **and** every older edition to NEW. Merge them into one case group; don't just add a new case, or older editions stay pointed at OLD.
- `liwords-ui/src/App.tsx` `puzzleLexicon` migration: same, if the family is a puzzle lexicon.
- English only: the definition fallbacks in `liwords-ui/src/utils/hooks/definitions.tsx`, and any defaults like `pkg/league/season_start.go`, `liwords-ui/src/leagues/admin.tsx`, `pkg/broadcasts/obs.go`, `db/migrations/*broadcasts*` defaults, and `sought_game_interactions.test.ts`.
- Anything else the grep turns up. For each hit, decide whether it means "the current edition" (update it) or "that exact edition" (leave it). Test fixtures, GCGs and CGPs almost always mean the exact edition.

Do not regenerate protos, bump go.mod, or touch letter distributions for a version update.

## 5. Tests that pin the new files

Add a test in `pkg/cwgame/lexicon_test.go` next to `TestCSW15IsNotALaterEdition` / `TestNSF26IsNotAnEarlierEdition`, using the `lexiconHasWord` helper there. It asserts that a few words from `added.txt` are present and a few from `removed.txt` are absent. Pick 2–3 of each, 2–15 tiles long, using only non-zero-count tiles. Also assert a dropped 1-letter and a >max-len word are absent, and check the reverse against OLD. This catches a wrong or mislabeled file being copied in.

In `pkg/entity/lexica_test.go`, add OLD to the historical list in `TestHistoricalLexicaCannotStartGames`.

## 6. Check it locally

```bash
go build ./pkg/... && source local.env && go test ./pkg/cwgame/ ./pkg/entity/
cd liwords-ui && npx prettier --write <changed files> && npx tsc && npx vitest run src/lobby
```

If tsc or vitest complain about missing modules or native bindings, the local `node_modules` is stale: run `npm ci` (it doesn't change the lockfile) and retry. Don't blame the lexicon change.

Optional but worth it: start the app, open the browser analyzer or WordSmog in NEW, and check that a word from `added.txt` is accepted.

## 7. Commit and PR

Stage only the files you touched, plus the new binaries. Commit, then **ask before pushing**. PR body:
- source list (file name and date), NEW code, alphabet
- the build summary: kept count, dropped counts and why, the three check results, added and removed counts with a few examples
- klv2 provenance (reused from OLD, or new)
- "Before merging" and "After merging" rollout checklists (step 8)

## 8. Production rollout (the user runs this, or you run it only with explicit approval)

Order matters, because the API's word service only picks up kwgs that exist when it starts.

**Before merging:**
1. Copy the files to the EFS data volume with `scripts/deploy_to_efs.sh NEW OLD`. It does a dry run by default: it lists the local md5s, shows OLD's files on the volume, and flags any NEW target that already exists. Show the user that output, then run it with `--apply` only once they approve. The script never overwrites anything. The volume is mounted at `/mnt/efs` on `woogles-wg` (`ssh woogles-wg`, as ec2-user, which owns the files), with data under `/mnt/efs/data`:
   - `lexica/gaddag/2024/`: `NEW.kwg`, `NEW.kad`, `NEW.klv2`, `super-NEW.klv2`. This is what the API, the bot and the analysis workers read. The `2024` is the KWG path prefix; it's not `lexica/gaddag/`.
   - `strategy/NEW/leaves.klv2` (+ `super-leaves.klv2`): the legacy leaves folder. Every current lexicon has one, identical to its gaddag klv2.
   - `lexica/NEW.wmp`: the word map (step 2). Not handled by the script.
   - `lexica/words/NEW.txt`: optional definitions (`WORD\tdefinition`, `LC_ALL=C` sorted). There isn't one for NSF. Not handled by the script.
   - The klv2 matters. Without `NEW.klv2` on disk, macondo **doesn't download it**: it silently borrows the family default's leaves from `defaultForLexicon` in macondo `dataloaders/strategy.go` (NSF → `NSF23`), which is an older table.
2. If the alphabet has ≤32 letters, build a word map for macondo with `go run ./cmd/make_all_wmps -lexica NEW` in macondo and put it on the volume. Norwegian and Polish have 33 letters, so skip it for them.

**After the deploy:**
3. Restart the macondo bot. It also serves WordSmog natively, from `NEW.kad`; there is no `wolges_awsm` service any more.
4. Clubs whose default is an old edition: **before writing the PR, check whether there are any.** Run the read-only query in `reference/prod-sql.md` with `bin/proddb -c "<SELECT>"` (it opens an SSH tunnel to the prod DB; SELECT only). If there are none (NSF26 had none), skip this step: no migration, no SQL. If there are some, add a migration to the PR (`./gen_migration.sh <name>`, which needs docker, or create the two files by hand with a UTC `YYYYMMDDHHmm` prefix; see `202402150312_default_club_settings_nwl23` for the pattern). **Match with an explicit list of editions, never `LIKE 'CSW%'`**: that would also catch `CSW24X`.
5. List, but don't change, leagues (`settings->>'lexicon'`) and unfinished tournaments still on OLD, and ask the user what to do with them. Running events keep their lexicon. Leagues use their settings at the next season start.
6. Smoke test: a rated game against a bot in NEW, playing a word from `added.txt`; WordSmog; the browser analyzer; a word lookup.

## 9. Cross-repo follow-ups (usually none needed; check each)

- **macondo**: `dataloaders/strategy.go` `defaultForLexicon` (the family's fallback leaves), and `cmd/make_all_wmps/main.go` `defaultLexica`, which must mirror `AllowedNewGameLexica`. Suggest a small PR if either still names an older edition.
- **kwgc / word-golib / MinMacondoVersion** (`pkg/analysis/service.go`): only for a new alphabet.
- **Wiki**: if anything in this process changed, update `Adding-a-new-lexicon-(2026)` in the liwords wiki (a separate git repo, `liwords.wiki.git`), and this skill.
