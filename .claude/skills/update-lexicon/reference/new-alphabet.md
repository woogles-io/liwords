# Adding a lexicon with a new alphabet or letter distribution

This is much bigger than a version update and spans several repos. Plan it with the user first. Reference PRs: liwords #1952 (add Slovene) and #1955 (browser analyzer and MinMacondoVersion bump).

Rough order, since each repo depends on the one before:

1. **kwgc / wolges** (Andy Kurnia's repos): add a tileset so the kwg, kad and klv2 can be built at all. `build_lexicon.py` refuses alphabets kwgc doesn't know.
2. **word-golib**: add the letter distribution and a prefix in `tilemapping.ProbableLetterDistributionName`. Tag a release.
3. **macondo**: bump word-golib, handle anything alphabet-specific (for example, alphabets over 32 letters can't use WMP), and tag a release. The remote analyzer workers need this version.
4. **liwords** (PR #1952 is the template):
   - `go.mod`: bump word-golib (and macondo).
   - `pkg/memento/letterdistributions/<name>` and `pkg/memento/tiles-<name>.png`, plus the `tiles.go` entries (for GIF/PNG rendering).
   - `liwords-ui/src/constants/alphabets.ts`: the new alphabet (scores, counts, vowels, categories, `bnjyable`).
   - `pkg/entity/lexica.go`, `pkg/entity/ratings.go` (a new rating family key), `pkg/entity/sought_game.go`, `pkg/omgwords/service.go`.
   - `liwords-ui/src/shared/lexica.ts`: a new entry plus both rating-name maps. Also `lexicon_display.tsx`, `sought_game_interactions.ts` and `loader.ts`.
   - The proto comment listing supported distributions (`api/proto/ipc/omgwords.proto`). Regenerate with `go generate`, never `make proto`.
   - `pkg/analysis/service.go` `MinMacondoVersion` → the macondo version that supports the alphabet (PR #1955).
5. Then the same file build, verification and rollout as a version update (SKILL.md steps 1, 3, 5–8), with a klv2 that has to come from somewhere real (magpie/wolges autoplay), not a copy.
