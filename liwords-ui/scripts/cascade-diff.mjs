#!/usr/bin/env node
/**
 * cascade-diff — proves that turning `colorModed()` into a passthrough does not
 * change which declaration wins anywhere in the stylesheet.
 *
 * Background: `src/color_modes.scss` currently emits every themed rule twice,
 * under `.mode--default &` and `.mode--dark &`. The Mantine migration replaces
 * that with runtime CSS custom properties, which means every themed rule loses
 * one class of specificity. Rules that only win today because of that extra
 * class would silently flip.
 *
 * This script compiles every stylesheet twice — once as-is ("before"), once with
 * `colorModed()` reduced to `@content` and `m()` returning `var(--woogles-*)`
 * ("after") — projects both into a single color mode, resolves the cascade, and
 * reports any (selector, property) whose winning value changes.
 *
 * Scope: it compares declarations competing on the *same* normalized selector,
 * which is where the mixin change actually bites. It will not catch a flip
 * between two different selectors that happen to match the same element; an
 * earlier rightmost-compound heuristic did, but produced far more noise than
 * signal (every `.anticon` rule in the codebase against every other).
 *
 *   node scripts/cascade-diff.mjs            # diff, exit 1 on any change
 *   node scripts/cascade-diff.mjs --verbose  # also list blocked files in full
 *
 * Files that cannot compile in "after" mode are reported separately: those are
 * the ones calling Sass color functions on `m()` output, which must be converted
 * to precomputed derived tokens before the bridge can land.
 */
import * as sass from "sass";
import postcss from "postcss";
import {
  cpSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, relative, resolve } from "node:path";

const SRC = resolve(import.meta.dirname, "..", "src");
const MODES = ["default", "dark"];
const VERBOSE = process.argv.includes("--verbose");

/** Definition-only files: no CSS output of their own worth diffing. */
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
// Compilation
// ---------------------------------------------------------------------------

/**
 * Builds a shadow copy of the source tree with color_modes.scss patched to the
 * "after" world. A custom Sass importer will not do: Sass gives the containing
 * file's importer first crack at a load, so filesystem-relative `@use` never
 * reaches a custom one. Shadowing the whole tree sidesteps that entirely.
 */
function makeShadowTree() {
  const dir = mkdtempSync(join(tmpdir(), "cascade-diff-"));
  const root = join(dir, "src");
  cpSync(SRC, root, {
    recursive: true,
    filter: (s) => statSync(s).isDirectory() || s.endsWith(".scss"),
  });

  const patched =
    readFileSync(join(SRC, "color_modes.scss"), "utf8").replace(
      /@mixin colorModed\(\)[\s\S]*$/,
      "",
    ) +
    `
@mixin colorModed() { @content; }
@function m($key) { @return var(--woogles-#{$key}); }
`;
  writeFileSync(join(root, "color_modes.scss"), patched);
  return { dir, root };
}

const shadow = makeShadowTree();
process.on("exit", () => rmSync(shadow.dir, { recursive: true, force: true }));

function compile(file, passthrough) {
  const root = passthrough ? shadow.root : SRC;
  const target = passthrough ? join(root, relative(SRC, file)) : file;
  return sass.compile(target, {
    loadPaths: [root],
    silenceDeprecations: ["import", "global-builtin"],
    quietDeps: true,
  }).css;
}

/** Builds `--woogles-<key>` → literal value, per mode, straight from $modes. */
function tokenTable() {
  const table = {};
  for (const mode of MODES) {
    const css = sass.compileString(
      `@use "sass:map";
       @use "color_modes" as cm;
       :root { @each $k, $v in map.get(cm.$modes, ${mode}) { --woogles-#{$k}: #{$v}; } }`,
      {
        loadPaths: [SRC],
        silenceDeprecations: ["import", "global-builtin"],
        quietDeps: true,
      },
    ).css;
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

/** Approximate specificity. Adequate for this codebase — no ids, no :is()/:where(). */
function specificity(sel) {
  // Pseudo-elements carry element weight; strip them first so the class-level
  // pseudo-class pattern below does not also match them.
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
 * Projects a rule set into one color mode: keeps unprefixed rules and rules for
 * this mode, stripping the prefix so both worlds are directly comparable.
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

// ---------------------------------------------------------------------------
// Flatten CSS → rules
// ---------------------------------------------------------------------------

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

/** Resolve the cascade: key → winning declaration. */
function resolve_(rules, tokens) {
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
        // Substitute var() so "before" hex and "after" var() compare equal.
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

const tokens = tokenTable();
const files = findScss(SRC).sort();
const blocked = [];
const changes = [];

for (const file of files) {
  const rel = relative(SRC, file);
  let before, after;
  try {
    before = compile(file, false);
  } catch (e) {
    console.error(
      `  !! ${rel} fails to compile as-is: ${e.message.split("\n")[0]}`,
    );
    process.exitCode = 1;
    continue;
  }
  try {
    after = compile(file, true);
  } catch (e) {
    blocked.push({ rel, message: e.message.split("\n").slice(0, 3).join(" ") });
    continue;
  }

  const beforeRules = flatten(before);
  const afterRules = flatten(after);

  for (const mode of MODES) {
    const b = resolve_(projectMode(beforeRules, mode), tokens[mode]);
    const a = resolve_(projectMode(afterRules, mode), tokens[mode]);
    for (const [key, bw] of b) {
      const aw = a.get(key);
      if (!aw) {
        changes.push({
          rel,
          mode,
          key,
          from: bw.value,
          to: "(declaration disappeared)",
        });
      } else if (aw.value !== bw.value) {
        changes.push({
          rel,
          mode,
          key,
          from: bw.value,
          to: aw.value,
          was: bw.selector,
          now: aw.selector,
        });
      }
    }
  }
}

console.log(
  `\ncascade-diff — ${files.length} stylesheets, modes: ${MODES.join(", ")}\n`,
);

if (blocked.length) {
  console.log(
    `BLOCKED (${blocked.length}) — cannot compile with colorModed() as a passthrough.`,
  );
  console.log(
    `These call Sass color functions on m() output and need derived tokens first:\n`,
  );
  for (const b of blocked)
    console.log(`  ${b.rel}${VERBOSE ? `\n      ${b.message}` : ""}`);
  console.log("");
}

if (changes.length) {
  console.log(
    `CASCADE CHANGES (${changes.length}) — a different declaration wins:\n`,
  );
  for (const c of changes) {
    const [media, compound, prop] = c.key.split("|");
    console.log(
      `  ${c.rel} [${c.mode}] ${compound} { ${prop} }${media ? `  @media ${media}` : ""}`,
    );
    console.log(`      before: ${c.from}   (${c.was ?? "?"})`);
    console.log(`      after:  ${c.to}   (${c.now ?? "?"})`);
  }
  console.log("");
  process.exitCode = 1;
} else if (!blocked.length) {
  console.log("No cascade changes. The bridge is behaviour-neutral.\n");
} else {
  console.log("No cascade changes among the files that compiled.\n");
}
