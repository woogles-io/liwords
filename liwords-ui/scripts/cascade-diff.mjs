#!/usr/bin/env node
/**
 * cascade-diff — compares the compiled CSS cascade against another git ref and
 * reports every declaration that changes which value wins.
 *
 *   node scripts/cascade-diff.mjs                 # vs origin/master
 *   node scripts/cascade-diff.mjs master          # vs a specific ref
 *   node scripts/cascade-diff.mjs --verbose       # also list unchanged files
 *
 * Exits non-zero if anything changed, so it works as a CI gate or a pre-merge
 * check. This is the safety net for the antd -> Mantine migration: each phase
 * rewrites stylesheets, and "did any element end up a different colour?" is
 * exactly the question that manual QA is worst at answering.
 *
 * How it compares:
 *  - Both trees are compiled per entry point, the same way rsbuild does it.
 *  - Selectors are projected into a single colour mode, stripping any
 *    `.mode--default ` / `.mode--dark ` prefix, so a tree that predates the
 *    CSS-custom-property bridge compares directly against one that follows it.
 *  - var(--woogles-*) is resolved back to a literal using each tree's own token
 *    values, so `#707070` on one side and `var(--woogles-color-gray-medium)` on
 *    the other are recognised as the same colour.
 *
 * Scope: it compares declarations competing on the same normalised selector.
 * It will not catch a flip between two different selectors that happen to match
 * the same element; an earlier rightmost-compound heuristic did, but produced
 * far more noise than signal (every `.anticon` rule against every other).
 */
import * as sass from "sass";
import postcss from "postcss";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, relative, resolve } from "node:path";

const UI = resolve(import.meta.dirname, "..");
const SRC = join(UI, "src");
const MODES = ["default", "dark"];

const args = process.argv.slice(2).filter((a) => a !== "--verbose");
const VERBOSE = process.argv.includes("--verbose");
const REF = args[0] ?? "origin/master";

/** Definition-only; no CSS output of its own worth diffing. */
const SKIP = new Set(["base.module.scss"]);

function findScss(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) findScss(p, out);
    else if (name.endsWith(".scss") && !SKIP.has(name)) out.push(p);
  }
  return out;
}

// ---------------------------------------------------------------------------
// Materialise the base tree
// ---------------------------------------------------------------------------

/**
 * Extracts liwords-ui/src at `ref` into a temp dir. git archive rather than a
 * worktree: no index or bookkeeping to clean up, and it cannot disturb the
 * working tree if the process dies.
 */
function checkoutBase(ref) {
  const dir = mkdtempSync(join(tmpdir(), "cascade-diff-"));
  try {
    const tar = execFileSync("git", ["archive", ref, "--", "liwords-ui/src"], {
      cwd: resolve(UI, ".."),
      maxBuffer: 512 * 1024 * 1024,
    });
    execFileSync("tar", ["-x", "-C", dir], { input: tar });
  } catch (e) {
    rmSync(dir, { recursive: true, force: true });
    throw new Error(`could not read ${ref}: ${e.message.split("\n")[0]}`);
  }
  return { dir, root: join(dir, "liwords-ui", "src") };
}

function compile(file, root) {
  return sass.compile(file, {
    loadPaths: [root],
    silenceDeprecations: ["import", "global-builtin"],
    quietDeps: true,
  }).css;
}

/**
 * --woogles-<key> -> literal, per mode, from a tree's own colour definitions.
 * Prefers the emitTokens() mixin; falls back to walking $modes directly so that
 * refs predating the token bridge still resolve.
 */
function tokenTable(root) {
  const table = {};
  for (const mode of MODES) {
    const opts = {
      loadPaths: [root],
      silenceDeprecations: ["import", "global-builtin"],
      quietDeps: true,
    };
    let css = "";
    try {
      css = sass.compileString(
        `@use "color_modes" as cm; :root { @include cm.emitTokens(${mode}); }`,
        opts,
      ).css;
    } catch {
      try {
        css = sass.compileString(
          `@use "sass:map";
           @use "color_modes" as cm;
           :root { @each $k, $v in map.get(cm.$modes, ${mode}) { --woogles-#{$k}: #{$v}; } }`,
          opts,
        ).css;
      } catch {
        css = "";
      }
    }
    const m = {};
    postcss.parse(css).walkDecls((d) => {
      if (d.prop.startsWith("--woogles-")) m[d.prop] = d.value.trim();
    });
    table[mode] = m;
  }
  return table;
}

// ---------------------------------------------------------------------------
// Selector analysis
// ---------------------------------------------------------------------------

/** Approximate specificity. Adequate here — no ids, no :is()/:where(). */
function specificity(sel) {
  const pseudoElements = (sel.match(/::[\w-]+/g) || []).length;
  const s = sel.replace(/::[\w-]+/g, "");
  const ids = (s.match(/#[\w-]+/g) || []).length;
  const classes =
    (s.match(/\.[\w-]+/g) || []).length +
    (s.match(/\[[^\]]*\]/g) || []).length +
    (s.match(/(?<!:):[\w-]+(\([^)]*\))?/g) || []).length;
  const tags = (s.match(/(^|[\s>+~])[a-zA-Z][\w-]*/g) || []).length;
  // Combinators contribute nothing to specificity.
  return ids * 10000 + classes * 100 + (tags + pseudoElements);
}

const MODE_PREFIX = /^\.mode--(default|dark)\s+/;

/**
 * Projects a rule set into one colour mode: keeps unprefixed rules and rules for
 * this mode, stripping the prefix so both trees are directly comparable.
 */
function projectMode(rules, mode) {
  const out = [];
  for (const r of rules) {
    const m = r.selector.match(MODE_PREFIX);
    if (m && m[1] !== mode) continue;
    out.push({
      ...r,
      selector: m ? r.selector.replace(MODE_PREFIX, "") : r.selector,
    });
  }
  return out;
}

function flatten(css) {
  const rules = [];
  let order = 0;
  const walk = (container, media) => {
    container.each((node) => {
      if (node.type === "atrule") {
        if (node.name === "media")
          walk(node, [...media, node.params].join(" and "));
        else if (node.nodes) walk(node, media);
      } else if (node.type === "rule") {
        const decls = [];
        node.walkDecls((d) =>
          decls.push({
            prop: d.prop,
            value: d.value.trim(),
            important: d.important,
          }),
        );
        if (!decls.length) return;
        for (const selector of node.selectors) {
          rules.push({
            selector,
            media,
            decls,
            order: order++,
            spec: specificity(selector),
          });
        }
      }
    });
  };
  walk(postcss.parse(css), []);
  return rules;
}

/** Resolve the cascade: key -> winning declaration. */
function resolveCascade(rules, tokens) {
  const winners = new Map();
  for (const r of rules) {
    r.decls.forEach((d, i) => {
      const key = `${r.media}|${r.selector}|${d.prop}`;
      const rank = (d.important ? 1e9 : 0) + r.spec;
      // Declarations within one rule must break ties among themselves, so the
      // document position is (rule order, declaration index), not rule order.
      const pos = r.order * 1000 + i;
      const prev = winners.get(key);
      if (!prev || rank > prev.rank || (rank === prev.rank && pos > prev.pos)) {
        const value = d.value.replace(
          /var\((--woogles-[\w-]+)\)/g,
          (whole, name) => tokens[name] ?? whole,
        );
        winners.set(key, { value, rank, pos, selector: r.selector });
      }
    });
  }
  return winners;
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

const base = checkoutBase(REF);
process.on("exit", () => rmSync(base.dir, { recursive: true, force: true }));

const headTokens = tokenTable(SRC);
const baseTokens = tokenTable(base.root);

const headFiles = new Set(findScss(SRC).map((f) => relative(SRC, f)));
const baseFiles = new Set(
  findScss(base.root).map((f) => relative(base.root, f)),
);

const added = [...headFiles].filter((f) => !baseFiles.has(f)).sort();
const removed = [...baseFiles].filter((f) => !headFiles.has(f)).sort();
const shared = [...headFiles].filter((f) => baseFiles.has(f)).sort();

const errors = [];

/**
 * Resolve every stylesheet in a tree. Returns per-file winners plus a tree-wide
 * index of (mode, selector, property) -> every value that wins somewhere.
 *
 * The tree-wide index is what makes moving a rule between files a non-event.
 * Each .scss is its own Sass compilation, so a rule living in a shared partial
 * is emitted once per entry point; relocating it to a single file reads as
 * "declaration gone" in dozens of files even though the bundle is unchanged.
 */
function resolveTree(root, files, sideLabel, tokens) {
  const perFile = new Map();
  const index = new Map();
  for (const rel of files) {
    let css;
    try {
      css = compile(join(root, rel), root);
    } catch (e) {
      errors.push({ rel, side: sideLabel, message: e.message.split("\n")[0] });
      continue;
    }
    const rules = flatten(css);
    const byMode = {};
    for (const mode of MODES) {
      const winners = resolveCascade(projectMode(rules, mode), tokens[mode]);
      byMode[mode] = winners;
      for (const [key, w] of winners) {
        const gk = `${mode}|${key}`;
        if (!index.has(gk)) index.set(gk, new Set());
        index.get(gk).add(w.value);
      }
    }
    perFile.set(rel, byMode);
  }
  return { perFile, index };
}

// Added/removed files are compiled too: they belong in the tree-wide index even
// though there is no counterpart to diff them against file-by-file.
const headTree = resolveTree(SRC, [...shared, ...added], "HEAD", headTokens);
const baseTree = resolveTree(
  base.root,
  [...shared, ...removed],
  REF,
  baseTokens,
);

const changes = [];
let compared = 0;

for (const rel of shared) {
  const h = headTree.perFile.get(rel);
  const b = baseTree.perFile.get(rel);
  if (!h || !b) continue;
  compared++;

  for (const mode of MODES) {
    const gk = (key) => `${mode}|${key}`;
    for (const [key, bw] of b[mode]) {
      const hw = h[mode].get(key);
      if (!hw) {
        // Only a real loss if nothing anywhere in HEAD still wins this value.
        if (!headTree.index.get(gk(key))?.has(bw.value)) {
          changes.push({
            rel,
            mode,
            key,
            from: bw.value,
            to: "(declaration gone)",
          });
        }
      } else if (hw.value !== bw.value) {
        changes.push({ rel, mode, key, from: bw.value, to: hw.value });
      }
    }
    for (const [key, hw] of h[mode]) {
      if (!b[mode].has(key) && !baseTree.index.get(gk(key))?.has(hw.value)) {
        changes.push({
          rel,
          mode,
          key,
          from: "(new declaration)",
          to: hw.value,
        });
      }
    }
  }
}

console.log(`\ncascade-diff — HEAD vs ${REF}`);
console.log(`${compared} stylesheets compared, modes: ${MODES.join(", ")}\n`);

if (added.length) console.log(`Added (not compared):   ${added.join(", ")}`);
if (removed.length)
  console.log(`Removed (not compared): ${removed.join(", ")}`);
if (added.length || removed.length) console.log("");

if (errors.length) {
  console.log(`COMPILE ERRORS (${errors.length}):\n`);
  for (const e of errors)
    console.log(`  [${e.side}] ${e.rel}\n      ${e.message}`);
  console.log("");
  process.exitCode = 1;
}

if (changes.length) {
  console.log(
    `CASCADE CHANGES (${changes.length}) — a different value wins:\n`,
  );
  for (const c of changes) {
    const [media, selector, prop] = c.key.split("|");
    console.log(
      `  ${c.rel} [${c.mode}] ${selector} { ${prop} }${media ? `  @media ${media}` : ""}`,
    );
    console.log(`      ${REF}: ${c.from}`);
    console.log(`      HEAD:   ${c.to}`);
  }
  console.log("");
  process.exitCode = 1;
} else if (!errors.length) {
  console.log(
    "No cascade changes. Every declaration resolves to the same value.\n",
  );
}

if (VERBOSE && !changes.length) {
  console.log(`Unchanged: ${shared.join(", ")}\n`);
}
