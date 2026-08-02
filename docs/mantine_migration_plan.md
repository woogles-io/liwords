# Migrate liwords-ui from Ant Design 5 to Mantine 9

> **This file is the source of truth for the migration.** It spans many PRs over
> a long period. Update the Progress table below in the same PR that lands each
> phase, so "where are we?" is answerable by reading the top of this file rather
> than by archaeology on `git log`.
>
> Branches are named `mantine-NN-<slug>` so `git branch --list 'mantine-*'` and
> `git log --oneline --grep=mantine` both give a progress view.

## Progress

| # | Phase | Branch | Status | Notes |
|---|---|---|---|---|
| 0.5 | Cascade hygiene + `cascade-diff` script | `mantine-00-cascade-hygiene` | [#1953](https://github.com/woogles-io/liwords/pull/1953) | 34/38 stylesheets proven cascade-neutral; 4 blocked pending derived tokens. Fixed 4 latent dark-mode bugs. Also declares `postcss` explicitly. |
| 1a | Derived tokens + token bridge | `mantine-01-token-bridge` | **done** | CSS 356.7 → 277.6 kB. `.mode--*` selectors 1710 → 34. 77 tokens on `:root`, 765 `var()` uses. `cascade-diff` repointed at a git ref. |
| 1b | Unwrap `colorModed()` entirely | — | next | Mechanical codemod over ~620 call sites; deletes the passthrough mixin and `m()`/`d()` |
| 0 | Mantine boot: deps, PostCSS, CSS layers, providers | — | todo | Includes `<StyleProvider layer>` for antd |
| 2 | Global component theme | — | todo | From `base.scss`'s reskin mixins |
| 3 | Notifications / modals / confirms façade | — | todo | Deletes `HookAPI` prop threading |
| 4a | `/admin` | — | todo | Proving ground, no fidelity bar |
| 4b | Moderator widget | — | todo | |
| 4c | `/leagues/admin` | — | todo | Hardest form conversion |
| 5a | Settings | — | todo | |
| 5b | Profile + static pages | — | todo | |
| 6a | Auth forms | — | todo | |
| 6b | Lobby | — | todo | |
| 7a | Tournament room | — | todo | |
| 7b | Director tools | — | todo | |
| 7c | Broadcasts + leagues public | — | todo | |
| 7d | Collections + puzzles + boardwizard | — | todo | |
| 8a | Chat | — | todo | |
| 8b | Gameroom shell | — | todo | |
| 8c | Board + analysis | — | todo | Highest risk |
| 9 | Remove antd | — | todo | |

Phases 0.5 and 1 run before Phase 0 deliberately: the token bridge is independent
of antd and worth landing on its own merits.

**Baselines to measure against** (captured 2026-07-27, before any Mantine work):
CSS bundle `350.5 kB` / `44.8 kB` gzipped; 39 SCSS files, 13,356 lines; 177 files
importing `antd`. Target SCSS end state is ~8 files / ~5,000 lines.

**Known pre-existing issues, not caused by this migration:**
- `themed()` is called 31× in `upcoming_tournaments.scss` and `tournaments_page.scss`
  but is **defined nowhere**. `color:themed("text")` ships in production CSS as an
  invalid declaration that browsers drop, so those elements silently inherit their
  colour. Decide the intended colours during phases 6b/7a.
- 7 vitest files fail with `localStorage` undefined, on master too.

## Context

`liwords-ui` is 218 `.tsx` files / ~76k LOC, of which **177 files import `antd`** and 81 import `@ant-design/icons`. Alongside it sits **13,374 lines of hand-written global SCSS**, and **600 of those lines are selectors that reach into antd's generated DOM** (`.ant-card-body`, `.ant-table-tbody > tr > td`, `.ant-modal-root`, …) across 27 files.

That coupling is the problem. Every antd minor release can rename or restructure an internal class, and 600 selectors silently stop matching. The overrides also fight antd's `:where()`-wrapped low-specificity CSS-in-JS, which is why there are 69 `!important`s and why `themes.tsx` contains tokens like `Table: { rowHoverBg: "unset" }` that exist *only* so the SCSS can win. Dark mode is a bolt-on: a `colorModed()` Sass mixin emits every themed rule twice (`.mode--default &` / `.mode--dark &`) at 619 call sites, so nothing is a real runtime variable and JS can't read a color.

**The outcome we want:** the app looks the same, dark mode is Mantine's, and styling lives in typed theme tokens in the codebase rather than in selectors aimed at a vendor's internals.

**Fidelity bar:** user-facing pages should be visually indistinguishable (small drift is fine). `/admin`, `/leagues/admin`, and the moderator modal are internal — no fidelity requirement at all, so they go first as the proving ground.

**Decisions already made (see "Decisions" below):** full rewrite to `@mantine/form`, no compatibility shims, Tabler icons, manual QA per PR.

---

## Why tokens beat the current SCSS-over-antd approach

This is the question you asked, answered against this codebase specifically.

### What we do today

antd v5 generates its CSS at runtime from a token algorithm (`@ant-design/cssinjs`). Those generated values are **invisible to SCSS** — there is no `--ant-color-primary` to read. So restyling anything means writing a selector against the rendered DOM:

- `src/base.scss` has a ~120-line `@mixin button` targeting `button.ant-btn, button.ant-btn-primary, .ant-modal-root button.ant-btn-secondary`, plus `@mixin modal`, `@mixin tabs`, `@mixin action-blocks` (which rewrites `.ant-card-actions` into a 3-button footer bar via `nth-child`).
- `src/chat/chat.scss` repeats a ~30-line `.ant-card.chat → .ant-card-body` height calculation **five times**, once per breakpoint, because layout depends on antd's internal box.
- `src/leagues/leagues.scss` styles `.ant-table-tbody > tr:hover > td` with `!important` to beat antd's own hover rule — and `src/themes.tsx` sets `Table: { rowHoverBg: "unset" }` to disarm that rule from the other side. Two mechanisms fighting over one hover colour.
- `src/App.scss` carries a candid comment: `// XXX: Can't find an AntD theme style thingy for this: .ant-table-filter-dropdown-btns`.

Theme values live in a Sass map (`$modes` in `src/color_modes.scss`), resolved at **build time**. `@include colorModed() { background: m($background) }` compiles to two rules. 619 call sites → ~1,238 emitted rules, none of them overridable at runtime, none inspectable in devtools as a token.

**Two failure modes follow directly:** antd upgrades change internals that 600 selectors depend on, and the specificity war between generated CSS and hand-written CSS has no stable winner.

### What Mantine does instead

1. **Real CSS custom properties.** Mantine emits `--mantine-color-*`, `--mantine-spacing-*`, `--mantine-font-size-*` on `:root`, and swaps light/dark values via `[data-mantine-color-scheme]` on `<html>`. `cssVariablesResolver(theme) => ({ variables, light, dark })` lets us emit our own `--woogles-*` set from `theme.other`. Dark mode becomes *one* rule with a variable, not two rules with baked hex values — and devtools shows you the token name.

2. **Customisation is a public contract.** Instead of guessing at `.ant-card-body`, Mantine documents a Styles API selector per component (`root`, `label`, `section`, `th`, `td`…) and gives you `theme.components.Card.extend({ defaultProps, classNames, styles, vars })`. Those selector names are part of the released API, so upgrades don't silently break them — the exact thing that keeps biting us.

3. **Plain CSS, predictable specificity.** Mantine ships static `.css` files, not runtime CSS-in-JS. Ship `@mantine/core/styles.layer.css` and everything lands in `@layer mantine`; unlayered app CSS always wins, regardless of import order. That single change removes the entire class of specificity fights that produced our 69 `!important`s.

4. **Tokens are typed and readable from JS.** `theme.other.boardTws` is a TypeScript value. Today, `src/base.module.scss` exists purely as a hack (`// a hack. Vite needs .module.scss to load scss variables.`) to smuggle three breakpoint numbers from SCSS into TS.

### What this concretely means here

| Today | After |
|---|---|
| `$modes` Sass map, 70 tokens × 2 modes, build-time | `theme.other` + `cssVariablesResolver` → `--woogles-*`, runtime |
| `@include colorModed() { color: m($x) }`, 619 sites, 2 rules each | `color: var(--woogles-x)`, 1 rule |
| `@mixin button` — 120 lines of `.ant-btn` | `theme.components.Button.extend({ classNames })` + one small CSS module |
| `.ant-card-body` height math × 5 breakpoints in `chat.scss` | Mantine `Card` with our own `classNames`, or plain divs — our DOM, our classes |
| `!important` vs `rowHoverBg: "unset"` standoff | one `Table` classNames override in `@layer`-beating app CSS |
| `base.module.scss` `:export` hack for 3 numbers | `theme.breakpoints` / `theme.other`, imported normally |

**The board and tile theme systems stay in SCSS.** `board_modes.scss` (12 schemes) and `tile_modes.scss` (13 schemes) use the same ancestor-class pattern but are orthogonal to both light/dark and antd, and their `ub()`/`ut()` accessors stay build-time. Nothing about the antd → Mantine move requires touching them. Similarly, the ~4,000 lines of board/tile/scorecard geometry CSS is domain CSS and carries over near-verbatim.

**Correction (verified after the plan was first drafted):** the `blender()`/`lightener()` helpers do only consume `ub()`, but there are **23 additional direct `color.mix()` / `color.scale()` / `color.adjust()` calls that take `m()` as an argument** — 14 in `gameroom.scss`, 4 in `shared/gameLists.scss`, 4 in `leagues/leagues.scss`, 1 in `lobby/lobby.scss`. Every one is a hard `sass` compile error the moment `m()` returns a `var()`. This is the main prerequisite for Phase 1 and is handled in Phase 0.5 below.

**Honest downside:** Mantine's components will not be pixel-identical to antd's out of the box. Matching the current look means writing per-component theme overrides — real work, just work that lives in a supported API instead of in selectors aimed at someone else's DOM.

---

## Decisions

| Question | Choice | Consequence |
|---|---|---|
| Forms (45 files, 325 `<Form.Item>`, 26 `useForm()`) | **Full rewrite to `@mantine/form`** | Largest single chunk of the migration. `rc-field-form`, `rc-input`, `rc-menu` all get dropped. Validation semantics rewritten by hand — the main regression risk. |

**Good news on the form rewrite:** the hard parts of rc-field-form are *not used here*. Measured across the codebase, `dependencies=`, `Form.Provider`, `Form.ErrorList`, `scrollToFirstError`, `onFieldsChange`, and `form.submit()` have **zero** call sites. What is actually used is shallow and maps cleanly: `rules` ×27 files, `labelCol`/`wrapperCol` ×19/17, `setFieldsValue` ×19, `resetFields` ×17, `valuePropName` ×14, `preserve` ×9, custom async `validator:` ×5, `Form.List` ×6, `useWatch` ×8, `noStyle` ×4, `shouldUpdate` ×1. Expect roughly half of `leagues/admin.tsx` to change (~750-900 of 1,718 lines) and about half of a small form like `settings/change_password.tsx` (~55 of 105) — but the resulting code is genuinely shorter and clearer than what it replaces, since each `Form.Item` + child pair collapses into one `getInputProps`-bound input.
| Compat shims | **None** | Every call site becomes idiomatic Mantine. No mini-antd to maintain. More work, cleaner end state. |
| Icons | **Keep `@ant-design/icons`** | Verified standalone — its only deps are `@ant-design/colors`, `@ant-design/icons-svg`, `@rc-component/util`, `clsx`. No `antd`. So it survives antd's removal untouched: zero icon churn, zero visual drift, and the 56 `.anticon*` SCSS selectors keep working. Revisit whenever, or never. |
| Verification | **Manual QA per PR**, light + dark | No screenshot harness. Compensate with small PRs and an explicit per-PR checklist. |

**Consequence of "no shims" worth calling out.** Mantine's `Table` is presentational only — no columns API, no sorting, no pagination — so *something* has to hold that logic across 38 `<Table>` instances in 30 files. The measured prop surface is small and closed: client-side sort (49 `sorter`), client-side pagination, sticky first column (3 `fixed: "left"`, always with `scroll={{x:"max-content"}}`), row click handlers (7 `onRow`), row classes (9). **There are zero grouped headers, no `expandable`, no `summary`, no custom `components`, no `filterDropdown`, and no server-side pagination.**

So: build **one app-owned `DataTable`** (~400 LOC over Mantine's `Table` primitives), but give it a **Mantine-idiomatic API, not antd's**. That respects "no shims" — the 38 call sites all get genuinely rewritten and no antd-shaped API survives — while not hand-rolling the same `useState` sort logic 38 times. It emits our own stable classes (`.wdt-head`, `.wdt-row`, `.wdt-cell`), which lets us delete the 9 `.ant-table-tbody` / 7 `.ant-table-thead` selectors *and* the `Table: { rowHoverBg: "unset" }` token that exists only so the SCSS can win.

Rejected alternatives: `mantine-react-table` (community wrapper, historically trails Mantine majors by 3-6 months — a bad bet mid-migration, ~120 KB gz for ~15% of its features); `mantine-datatable` (fine library, but its API is far enough from ours that we rewrite all 235 column objects *and* take the dependency); `@tanstack/react-table` (most defensible, but it only replaces ~60 of the 400 LOC while adding a translation layer). If `DataTable` ever needs virtualisation or server-side paging, its internals can be swapped for TanStack behind the same signature.

The same logic applies nowhere else — `Row`/`Col`, `Space`, `Typography`, `Popconfirm`, `Affix`, and `Select` all get rewritten at the call site as decided.

---

## Styling policy — how each area gets migrated

The per-area loop, applied identically in every phase from 4 onward:

1. **Translate the components to Mantine.**
2. **Delete that area's `.ant-*` SCSS outright.** It's styling someone else's DOM; none of it survives.
3. **Absorb what's left, in this order of preference:**
   - a theme token (`var(--woogles-*)`, `--mantine-spacing-*`) — for any value
   - a **theme component override** — if it's how a component looks app-wide, it belongs in Phase 2's theme, not here
   - **Mantine layout props** (`p`, `m`, `gap`, `c`, `bg`, `w`, `maw`) — this is where most trivial SCSS genuinely deletes rather than moves
   - a scoped **`*.module.scss` colocated with the component** — for anything genuinely local
   - inline `styles={{}}` — **last resort only**
4. **Delete the area's global SCSS file.**

**The trap to avoid.** Mantine makes `styles={{ root: {...} }}` very easy, and there are already 697 inline `style={{}}` props in this codebase. Scattering styling across TSX at inline specificity is a different flavour of the problem we're leaving. If a rule needs more than a couple of declarations, it goes in a CSS module.

**Scoping is the real win.** Today there are ~2,938 global rule blocks with zero scoping and 69 `!important`. Whatever survives should be a `*.module.scss` next to its component, so it can't collide and can't be fought over.

### What can actually be deleted, honestly

| Category | Lines | Fate |
|---|---|---|
| `.ant-*` overrides (across 27 files) | ~600 | **deleted** |
| Board/tile/gameroom domain CSS (`gameroom.scss` 3,622 of which only 80 are antd, `playerCards.scss` 414, most of `puzzles.scss` 349) | ~4,000 | **stays global SCSS** |
| Token/theme maps (`color_modes` 257, `board_modes` 259, `tile_modes` 239) | 755 | `color_modes` → Mantine theme; board/tile maps **stay** (they need Sass loops) |
| `base.scss` antd reskin mixins (lines 160-532) | ~370 | **deleted** → Phase 2 theme |
| `standalone-embed.scss` | 179 | **untouched** (self-contained, antd-free) |
| App layout + component styling (everything else) | ~7,400 | roughly half deletes into Mantine props/tokens; the rest becomes CSS modules |

**Realistic end state: 39 files → ~8, and 13,356 lines → ~5,000.** Not zero, and anyone promising zero is planning to hide 4,000 lines of board rendering somewhere worse.

The board CSS specifically should **stay one cohesive global sheet**, not become modules: it's generated by Sass loops over the `$boards`/`$tiles` maps and keyed off global body classes (`.board--metallic`, `.tile--whitish`), so CSS modules would mean `:global()` on nearly every selector.

---

## Phase 0 — Foundations (1 PR, zero visual change)

**Files:** `package.json`, new `postcss.config.cjs`, `src/index.tsx`, `src/App.tsx`, new `src/theme/`.

1. Add `@mantine/core`, `@mantine/hooks`, `@mantine/form`, `@mantine/dates`, `@mantine/notifications`, `@mantine/modals`, `@mantine/carousel`, `@tabler/icons-react`; dev-add `postcss`, `postcss-preset-mantine`, `postcss-simple-vars`. Rsbuild picks up `postcss.config.cjs` automatically via `postcss-load-config`, and runs it after `@rsbuild/plugin-sass` — verify both pipelines coexist before anything else.
2. **CSS layers — layer *both* libraries.** In `src/index.tsx`, declare `@layer antd, mantine;` first, then import `@mantine/core/styles.layer.css` (which self-wraps in `@layer mantine`), then `antd/dist/reset.css` via `@import … layer(antd)`. Also wrap antd's *runtime* CSS-in-JS: `@ant-design/cssinjs` exposes `layer?: boolean` on `StyleContext` (verified in `node_modules/@ant-design/cssinjs/lib/StyleContext.d.ts:36-37`), so `<StyleProvider layer>` puts every generated `.ant-*` rule into `@layer antd`.

   This is the most important single line in the migration. Unlayered CSS beats *every* layer regardless of specificity, so all 13,374 lines of app SCSS win automatically — which **neutralises the specificity drop from Phase 1** and ends the `!important` arms race immediately. It also fixes a live bug: antd's `genHoverActiveButtonStyle` compiles to `(0,5,0)` while our `@mixin button` hover rule is `(0,3,1)`, so antd already wins that fight today and ~12 lines of `base.scss` are dead code.
3. **`src/theme/woogles-theme.ts`** — `createTheme({ fontFamily: "Mulish", headings, colors, primaryColor, radius, breakpoints, other })`, where `other` holds every token currently in the `$modes` map, split light/dark; plus a `cssVariablesResolver` emitting them as `--woogles-*` under the `light`/`dark` keys.
4. **Provider tree** in `src/App.tsx`: `MantineProvider` wrapping (for now) the existing `ConfigProvider`/`AntDApp`, with `<Notifications />` and `<ModalsProvider>`. Drive it from the existing Zustand store via `forceColorScheme={themeMode}` so there is exactly one source of truth — do **not** let Mantine manage its own `mantine-color-scheme` localStorage key while `darkMode` still exists.
5. Reconcile z-index: antd's `zIndexPopupBase` is 1000 (and `score_edit_popover.tsx` hardcodes `zIndex={1100}`); Mantine's modal/popover default to 200/300. Raise Mantine's via `theme.components` defaults so Mantine popups sit above antd's during the overlap.

**Verify:** app builds and runs; nothing renders differently; `document.documentElement` has `data-mantine-color-scheme`; a scratch `<Button>` from Mantine renders themed. Confirm the embed bundle (`rsbuild.embed.config.ts`, `injectStyles: true`, `splitChunks: false`) is unaffected — `src/embed/` imports no antd today and must import no Mantine either.

---

## Phase 0.5 — Cascade hygiene (1 PR, prerequisite for Phase 1)

Small, self-contained, and valuable on its own merits. Land it before Phase 1 so the bridge PR's CSS diff is provably behaviour-neutral.

1. **Delete 4 dead hardcoded fallbacks.** The `mixed-decls` deprecation autofix wrapped literal fallbacks in `& { … }`, which moved them *after* the themed rule. They lose today only because `colorModed()` adds a class — so they become live bugs the instant the bridge lands. Verified:
   - `settings/settings.scss:591` (`color: #666`), `:600` (`background-color: #f5f5f5`), `:608` (`color: #999`)
   - `lobby/lobby.scss:95` (`color: inherit`) — the nastiest, because `inherit` is not dead code: the `li` would inherit from its wrapping `<a>`, which `App.scss` paints `$primary-dark`. **Announcement list text would turn blue.**
2. **Add a `cascade-diff` script** (`scripts/cascade-diff.mjs`, ~120 LOC, no new deps — `sass` is already present). Compile every entry SCSS twice, once with the real `colorModed()` and once with a passthrough shim, flatten both to `(selector, property, @media) → winning declaration`, and fail if the winning set differs. This is what found the four flips above; keeping it turns Phase 1 from "trust me" into a CI gate, and it catches future `mixed-decls` autofix regressions.
3. Convert the two legacy `@import "../base"` files (`leagues/leagues.scss`, `collections/collections.scss`) to `@use … as *`, and delete the unused `$shadow-lower`.

---

## Phase 1 — Token bridge (2 PRs, zero visual change)

This is the highest-leverage step and it is **independent of antd**. Do it before touching any component.

**1a — precompute the derived colours, then rewire the mixin.**

First, the blocker: **23 declarations call `color.mix()` / `color.scale()` / `color.adjust()` on `m()` output** (14 in `gameroom.scss`, 4 in `shared/gameLists.scss`, 4 in `leagues/leagues.scss`, 1 in `lobby/lobby.scss`). Once `m()` is a `var()`, each is a compile error.

**Do not reach for CSS `color-mix()`.** There's no `.browserslistrc`, so rsbuild's default targets include browsers without it, and `color.scale($c, $lightness: -10%)` is HSL-space *scaling* (`L → L × 0.9`) which `color-mix()` cannot express at all. Instead keep the maths in Sass and emit the results as additional custom properties: add a `$derived` table to `color_modes.scss` listing each `(op, tokenA, tokenB, amount)`, compute it once per mode at build time, emit `--woogles-d-*`, and add a `d($name)` accessor. The palette stays in one file and the 23 call sites become `d(qws-label)` etc.

Then: redefine `m($key)` to return `var(--woogles-#{$key})` and reduce `colorModed()` to a bare `@content` passthrough. All ~620 include sites / 778 `m()` calls across 30 SCSS files keep compiling unchanged. Delete the `$modes`-driven duplication.

**Declare the variables on `:root` / `<html>`, not `<body>`.** Custom-property substitution resolves at the *declaring* element. Mantine's `cssVariablesResolver` output lands on `:root`; if `--woogles-*` lived on `body`, then a `:root`-level `--mantine-color-text: var(--woogles-color-gray-extreme)` would resolve against `:root`, find nothing, and silently go invalid. So add a `documentElement` mirror to `src/stores/ui-store.ts` alongside the existing body-class write, and keep the body class until Phase 3 deletes the two portal blocks that need it.

Resolved non-issues (checked, no action needed):
- **The `ignore` sentinel is doubly dead.** `ui-store.ts` only adds the class when `mode && mode !== "default"`, so `.board--default` / `.tile--default` never appear in the DOM at all. Separately, in every `userboard()`/`userTile()` block the `colorModed()` include comes *first*, so today the board/tile rule wins a specificity *tie* on source order; after the bridge it wins on specificity outright. Strictly more robust.
- **Non-colour token values** (`shadow`, `board-shadow`, `color-tile-border: 0px solid transparent`) are all used in whole-declaration-value position, which `var()` handles fine. `number-board-space-opacity` and `type-board-space-mix-mode` are read only via `ub()` and stay build-time. There are no `#{m(…)}` interpolations and no `rgba(m(…), …)` calls.
- **No `colorModed()` inside `@keyframes` or `@supports`**; 6 inside `@media`, which is fine.

One thing to add to the review checklist: the `border: 0px solid transparent; border: ut($tile-border);` invalid-value fallback pattern (`gameroom.scss:1552-1553`) **stops working if the second declaration ever becomes a `var()`** — an undefined custom property is invalid at computed-value time and resets to `unset` rather than falling back. `ut()` stays build-time so we don't hit it today.

**1b — unwrap the mixin.** Mechanical codemod: `@include colorModed() { color: m($x); }` → `color: var(--woogles-x);`. Large diff, zero semantic change, trivially reviewable. Deletes `colorModed()` and `m()` entirely. Land it as its own PR so a real change never hides inside it.

**Expected payoff:** the built CSS contains 845 `.mode--default` and 865 `.mode--dark` selectors out of 332 KB minified; expect **35-45% off the CSS bundle**. **Honest cost:** dark mode stops being greppable in the compiled stylesheet — you'll read `var(--woogles-color-x)` and have to use devtools' computed panel instead of seeing literal hex.

---

## Phase 2 — Global component theme (1 PR)

**Do this once, before any area migrates.** Otherwise the first area hand-fixes Button/Card/Input styling locally and the next eight each re-fix it slightly differently — rebuilding the current mess inside a new library.

The source material already exists. `base.scss` lines 160-532 (~370 of its 562 lines) are `@mixin button`, `@mixin modal`, `@mixin tabs`, `@mixin notification`, `@mixin action-blocks` — they already encode what a Woogles button, modal, and tab bar *are*. Translate them into `theme.components.X.extend({ defaultProps, classNames, vars })` in `src/theme/`, backed by a small `src/theme/components/*.module.scss` per component where a class is genuinely needed.

Cover the components that appear everywhere and therefore must be consistent: `Button`, `Card`/`Paper`, `Modal`, `TextInput`/`NumberInput`/`PasswordInput`/`Textarea`, `Select`, `Tabs`, `Tooltip`, `Badge` (for `Tag`), `Alert`, `Menu` (for `Dropdown`), `Table`. Also fold in the four nested `<ConfigProvider>` overrides (`lobby.tsx:97`, `room.tsx:225`, `bot_selector.tsx:134`, `badge.tsx:59`) — three of them are the same Dropdown-padding tweak and collapse into one theme default.

Also port the custom 16-colour `Tag` palette from `App.scss` (`.ant-tag-gold`, `.ant-tag-win`, `.ant-tag-bye`, `.ant-tag-forfeit`, …) into a `Badge` variant set on the theme, since ~33 files depend on it.

Nothing renders differently at the end of this PR — no component has migrated yet. It just means every later PR gets the right look for free.

**Icons: no work.** `@ant-design/icons` stays (see Decisions), so the 79 icons and 56 `.anticon*` selectors are untouched by this migration.

**Verify:** build size doesn't balloon; walk each area once for icon size/weight/alignment. Expect to nudge sizes — Tabler defaults to 24px stroke-1.5 vs antd's 1em glyphs.

---

## Phase 3 — Notifications, messages, confirms (1 PR)

Replace `App.useApp()` (29 files: notification 21, modal 6, message 5) and the static `message.*` API (~35 files, ~60 call sites) with `@mantine/notifications` and `@mantine/modals`. Both are directly callable from non-React modules, which is what `src/store/socket_handlers.ts`, `src/socket/socket.ts`, `src/tournament/ready.ts` need.

This also **deletes the instance-threading**, which is the real win:
- `HookAPI` from `antd/lib/modal/useModal` currently crosses module boundaries as a prop — `src/mod/moderate.tsx` exports `moderateUser(modal: HookAPI, …)`, threaded through `src/shared/usernameWithContext.tsx` and `src/chat/chat_entity.tsx`. All three signatures lose the parameter.
- `MessageInstance` / `NotificationInstance` params in `src/store/meta_game_events.ts`, `src/tournament/tournament_error.tsx`, `src/gameroom/show_notif.tsx` likewise.

Keyed dismissal (`message.destroy("board-messages")` in `gameroom/table.tsx`, `store/socket_handlers.ts`) maps to `notifications.hide(id)`.

Match the current look via `theme.components.Notification` — note the existing design deliberately **inverts** notification colours against the app mode (light app → dark blue toast). Then delete `@mixin notification` from `base.scss` and the `body.mode--dark .ant-notification-*` / `.ant-message-notice-content` block.

**Verify:** trigger a game notification, a chat error, a tournament error, and a "leave game?" confirm, in both modes.

---

## Phase 4 — Internal surfaces / proving ground (3 PRs)

No fidelity requirement. This is where the `@mantine/form` patterns get established before they're applied to user-facing pages.

- **4a — `/admin`** (`src/admin/*`, 10 files, 3,091 LOC): `Layout.Sider`+`Menu` → Mantine `AppShell`/`NavLink`; `Table` ×6 including a server-controlled pagination case in `analysis_stats.tsx`; `Form` ×6 including a `Form.List` in `announcement_editor.tsx` and `useWatch` ×6 in `tourney_editor.tsx`; `Statistic`, `Image` preview, `AutoComplete`, `Popconfirm`. Clears all 26 deep `antd/lib/*` imports concentrated here, and `rc-menu`. Rewrite `src/admin/admin.scss`. **L**
- **4b — moderator widget**: `src/mod/moderate.tsx` (205 LOC) + `src/settings/roles_permissions.tsx`. Small, and it finishes the `HookAPI` removal started in Phase 3. **S**
- **4c — `/leagues/admin`**: `src/leagues/admin.tsx` — 1,718 LOC, the largest file in the repo, 48 `Form.Item`, 3 `useForm`, 24 `rules` blocks, 4 `DatePicker`, `Select showSearch`. Plus `leagues/zero_move_games_dashboard.tsx`. This is the single hardest form conversion; doing it here, unobserved, is deliberate. **XL**

**Verify:** exercise every admin action end-to-end against a local backend — create/edit a tournament, edit an announcement, generate puzzles, assign roles, run a league season action. Validation behaviour is what to watch, not pixels.

---

## Phase 5 — Settings, profile, static pages (2 PRs)

- **5a — Settings** (15 files): `personal_info` (Upload avatar + `react-easy-crop` modal + `DatePicker`), `preferences` (the dark-mode `Switch` itself — move it to `useMantineColorScheme` semantics while keeping the Zustand store authoritative), `change_password`, `integrations` (3 forms, Table, Upload with `LIST_IGNORE`, `Flex`), `api`, `blocked_players`, `close_account`, `markdown_tips`. Rewrite `settings/settings.scss`, `settings/markdown_tips.scss`. **L**
- **5b — Profile + static**: `profile/profile.tsx` (incl. `Carousel rows={2}` → `@mantine/carousel`; Embla has no multi-row, so the two rows become a CSS grid inside each slide), `profile/badge.tsx` (drop its nested `ConfigProvider`), `games_history`, `annotated_games_history`, `about/team.tsx`, `termsOfService`, `donate`, `donate_success`, `clubs`. **M**

---

## Phase 6 — Lobby and auth (2 PRs)

- **6a — Auth forms**: `register` (AutoComplete birthdate with a borderless `Input` child, 6 rules blocks, ToS `Checkbox`, `Modal.success`), `login`, `password_reset`, `new_password`, `verify_email` (`Result`). Rewrite `lobby/accountForms.scss`. **M**
- **6b — Lobby proper**: `seek_form.tsx` (2 always-open `Slider` tooltips, `AutoComplete`, `Radio.Group`/`Radio.Button` segmented → Mantine `SegmentedControl`), `bot_selector.tsx` (rich JSX option labels + its zero-padding `ConfigProvider`), `seek_confirm_modal.tsx` (the only `theme.useToken()` call), `gameLists`, `active_games`, `sought_games`, `correspondence_games`, `announcements` (Tabs). Rewrite `lobby/lobby.scss`, `seek_form.scss`, `upcoming_tournaments.scss`, `shared/gameLists.scss`. Drop the nested `ConfigProvider` in `lobby.tsx`. **L**

---

## Phase 7 — Tournaments, broadcasts, leagues, collections, puzzles (4 PRs)

- **7a — Tournament room** (17 files): `room.tsx` (nested `ConfigProvider`), `actions_panel`, `pairings` (`Affix offsetTop={10}` → `position: sticky`), `standings`, `recent_games`, `competitor_status`, `score_edit_popover` (a whole Form inside a `Popover`), `manage_directors_modal`, `player_scorecard_modal`, `monitoring/*` (incl. the only `rowSelection` Table). **L**
- **7b — Director tools** (16 files incl. `ghetto_tools/*`): form-dense, low traffic, moderate fidelity bar. Includes the `DatePicker.RangePicker` and `Form.List` in `ghetto_tools/*`, and `tournament_wizard.tsx` (`Steps`, `Radio.Button` cards, `showTime` DatePicker). **L**
- **7c — Broadcasts + leagues public** (12 + 11 files): `BroadcastRoom` (the only `Grid.useBreakpoint`, `TableColumnsType`, Tabs↔Select swap on mobile), `CreateBroadcast`/`EditBroadcast`, `tabs/*`; `league_page`, `standings`, `league_roster` (5 sorters + `SortOrder`), `leagues_list` (legacy `Collapse.Panel`), `player_game_history_modal`, `invite_user_widget`. Rewrite `leagues/leagues.scss` (1,024 lines, the worst `!important` offender) and `broadcasts/broadcasts.scss`. **XL**
- **7d — Collections + puzzles + boardwizard** (6 + 7 + 5 files): `CollectionViewer`, `CollectionNavigation` (`Paragraph ellipsis expandable`), `AddToCollectionModal`, `puzzle.tsx`, `puzzle_preview`, `editor_control` (Collapse `items`), `gcg_process_form` (Upload with client-side read), `new_game`. Rewrite `collections/collections.scss` (33 `!important`), `puzzles/*.scss`. **L**

---

## Phase 8 — Chat and gameroom (3 PRs, highest risk)

Do this last. `gameroom.scss` is 3,622 lines and `chat.scss` repeats `.ant-card-body` height math five times.

- **8a — Chat** (4 files): `chat.tsx` (controlled Tabs with `destroyOnHidden={false}` so panes stay mounted — Mantine `Tabs` needs `keepMounted`), `chat_entity`, `chat_channels` (AutoComplete + `filterOption`), `players`, `presences`. Rewrite `chat/chat.scss` and `playerList.scss`; the per-breakpoint card-height block is the single nastiest piece of CSS in the repo and should become our own container with our own class. **L**
- **8b — Gameroom shell** (`table.tsx`, `game_controls.tsx` with 4 Dropdowns + `Affix offsetTop={210}`, `player_cards`, `scorecard`, `pool`, `notepad`, `CommentsDrawer`, `comments`, `meta_event_control`, `show_notif`). **L**
- **8c — Board + analysis** (`board_panel.tsx` with `Affix offsetTop={126}`, `tile.tsx` controlled `Popover`, `rack_editor.tsx` (`InputRef` from `rc-input`), `exchange_tiles`, `challenge_words_modal` (indeterminate Checkbox), `analyzer`, `computer_analysis`, `blank_selector`, `definitions`). Rewrite `gameroom/scss/gameroom.scss` and `playerCards.scss` — but only the ~80 `.ant-*` lines; the ~3,500 lines of board geometry stay. **XL**

**Verify:** play a full game in both modes; check all 12 board themes and 13 tile themes; check the sticky rack and controls at every breakpoint; exercise examine mode, challenge, exchange, and the analyzer.

---

## Phase 9 — Teardown (1 PR)

Remove `antd`, `@ant-design/v5-patch-for-react-19`, `rc-field-form`, `rc-input`, `rc-menu`. Delete `src/themes.tsx`, `src/utils/focus_modal.tsx` (Mantine `Modal` traps focus natively), the `antd/dist/reset.css` import + `theme/vendor.css`, the `<StyleProvider layer>` wrapper, and `ConfigProvider`/`AntDApp` from `App.tsx`. `base.scss`'s antd reskin mixins are already gone as of Phase 2; delete the now-dangling `@include`s at `App.scss:5-8` and `lobby/lobby.scss:19`.

**Keep** `@ant-design/icons` — it has no `antd` dependency and the 56 `.anticon*` selectors depend on it.

`src/utils/focus_modal.tsx` deserves a look rather than a mechanical port: it still reads `props.visible`, which antd 5 renamed to `open`, so its autofocus behaviour is **probably already dead** across all 16 consumers. Verify before reimplementing it.

Gate: `grep -rn "from \"antd\|rc-field-form\|rc-input\|rc-menu" src` returns nothing, and `grep -rn "\.ant-" src --include=*.scss` returns only `.anticon*`. Add `antd` to the existing `no-restricted-imports` block in `eslint.config.mjs` permanently (the repo already uses that rule for `dayjs`).

**Do not miss this:** `src/setupTests.ts:2` imports `resize-observer-polyfill`, which is present only *transitively* via `rc-resize-observer` / `@ant-design/react-slick` — i.e. via antd. Add it as an explicit `devDependency` in this PR or the entire test suite breaks the moment antd is removed.

**Enforcement during the migration:** after each area PR, add a scoped rule so the area can't backslide:
```js
{ files: ["src/admin/**", "src/mod/**"],
  rules: { "no-restricted-imports": ["error", { paths: [{ name: "antd" }] }] } }
```

Opportunistic cleanups in the same PR: dead `src/learn/learn.scss`; the unreferenced Source Code Pro webfont; the `install` junk dependency; `moment` (keep `dayjs` — `@mantine/dates` needs it); the empty `liwords-ui/liwords-ui/src/{providers,theme,stores}` scaffolding dirs; the two legacy `@import "../base"` files.

---

## Tests

`setupTests.ts` stubs `window.matchMedia` for antd — **keep it**, Mantine uses it too. `src/__mocks__/styleMock.js` stays.

Three tests assert antd internals and must be handled as their areas migrate:
- `src/store/analysis_toast.test.tsx` — asserts toast colours derived from `theme.components.Notification.colorBgElevated`. Rewrite against Mantine's notification DOM in **Phase 3**, or delete it and rely on manual QA; it was written for issue #1917 and its value is regression-catching the inverted colour scheme.
- `src/leagues/player_game_history_modal.test.tsx` — wraps in antd `ConfigProvider`, `describe.each(["default","dark"])`, asserts `getAllByRole("columnheader")`. Swap the wrapper for `MantineProvider` in **Phase 7c**; the role assertions survive since Mantine `Table` renders real `<th>`.
- `src/gameroom/board_panel.test.tsx.snap` — regenerate in **Phase 8c**.

Add a shared `renderWithProviders()` test helper in Phase 0 so the provider swap is a one-line change later.

---

## Verification (per PR)

No screenshot harness, so each PR carries an explicit checklist:

1. `npm run build` (runs `tsc` + eslint via `@rsbuild/plugin-eslint`) and `npm run test` clean.
2. `cd liwords-ui && npm start` (or `docker compose up frontend`, which runs the same), walk every route the PR touches in **light and dark**, at mobile / laptop / desktop widths (768 / 1280 / 1440 — the breakpoints in `base.scss`).
3. `grep -rn "\.ant-" src/<area>` — the PR should reduce this to zero for its own area.
4. Exercise the interactive paths, not just the render: submit each form (including the invalid case — validation is the top regression risk given the `@mantine/form` rewrite), open each modal, sort/paginate each table, trigger each notification.
5. For Phases 1a/1b specifically: diff the built CSS size, and spot-check board and tile themes since they share the cascade with `colorModed()` output.

Keep PRs small enough to eyeball. If any single one gets too big to review visually, split it by file group before merging.

---

## Critical files

| File | Role |
|---|---|
| `liwords-ui/src/color_modes.scss` | 70 tokens + `$modes` map + `colorModed()`/`m()` — rewritten in Phase 1 |
| `liwords-ui/src/base.scss` | spacing/font/geometry vars, `:export` shim, typography mixins, **and** the antd reskin mixins |
| `liwords-ui/src/themes.tsx` | the ~30 antd tokens; deleted in Phase 9, superseded by `src/theme/woogles-theme.ts` |
| `liwords-ui/src/App.tsx` | provider tree + the whole route table |
| `liwords-ui/src/stores/ui-store.ts` | dark/board/tile mode source of truth; stays, drives Mantine |
| `liwords-ui/src/index.tsx` | style import order + CSS layer declaration |
| `liwords-ui/src/board_modes.scss`, `tile_modes.scss` | 12 + 13 themes; **untouched** by this migration |
| `liwords-ui/src/gameroom/scss/gameroom.scss` | 3,622 lines; only its ~80 `.ant-*` lines change |
| `liwords-ui/src/leagues/admin.tsx` | 1,718 LOC, hardest form conversion |
| `liwords-ui/rsbuild.config.ts` | needs no change; `postcss.config.cjs` is auto-discovered |
| `liwords-ui/rsbuild.embed.config.ts` | embed bundle is antd-free and must stay Mantine-free |
