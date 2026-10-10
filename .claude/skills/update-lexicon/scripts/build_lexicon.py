#!/usr/bin/env python3
"""Build and verify the word-graph files for a woogles lexicon.

Takes a raw word list as published by a word-list authority, cleans it up,
compiles it with kwgc into NAME.kwg (GADDAWG) and NAME.kad (alpha DAWG, for
WordSmog), then reads both back and proves they hold exactly the words that
went in. Optionally diffs the new edition against the previous one.

Nothing here touches the liwords repo; everything is written to --out.

Example:
    build_lexicon.py --input ~/Downloads/nsf2026.txt --name NSF26 \
        --alphabet norwegian --prev liwords-ui/public/wasm/2024/NSF25.kwg \
        --out /tmp/nsf26
"""

import argparse
import collections
import json
import os
import subprocess
import sys
import unicodedata

HERE = os.path.dirname(os.path.abspath(__file__))
LIWORDS = os.path.abspath(os.path.join(HERE, "..", "..", "..", ".."))
LD_DIR = os.path.join(LIWORDS, "pkg", "memento", "letterdistributions")
KWGC_ALPHABETS = {
    "english", "catalan", "dutch", "french", "german", "norwegian",
    "polish", "slovene", "spanish", "swedish",
}


def die(msg):
    print(f"FATAL: {msg}", file=sys.stderr)
    sys.exit(1)


def load_alphabet(name):
    """Returns {tile_label: count} from liwords' letter distribution file."""
    path = os.path.join(LD_DIR, name)
    if not os.path.exists(path):
        die(f"no letter distribution at {path}")
    tiles = {}
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            label, count = line.split(",")[:2]
            if label == "?":
                continue
            tiles[unicodedata.normalize("NFC", label)] = int(count)
    return tiles


def tokenize(word, labels_longest_first):
    """Splits an uppercased word into tiles. Returns None on an unknown char."""
    out = []
    i = 0
    while i < len(word):
        for lab in labels_longest_first:
            if word.startswith(lab, i):
                out.append(lab)
                i += len(lab)
                break
        else:
            return None
    return out


def run_kwgc(kwgc, args, outfile=None):
    # kwgc exits 0 even when it fails ("bad tile at offset N" + usage), so
    # success is judged by stderr and by the output file actually appearing.
    if outfile and os.path.exists(outfile):
        os.remove(outfile)
    p = subprocess.run([kwgc] + args, capture_output=True, text=True)
    if "bad tile" in p.stderr or "args:" in p.stdout or p.returncode != 0:
        die(f"kwgc {' '.join(args)} failed:\n{p.stderr}\n{p.stdout[:500]}")
    if outfile and (not os.path.exists(outfile) or os.path.getsize(outfile) == 0):
        die(f"kwgc {' '.join(args)} produced no {outfile}:\n{p.stderr}")
    return p.stdout


def dump_lines(stdout):
    return [l for l in stdout.split("\n") if l and not l.startswith("time taken")]


def sorted_file(lines, path):
    lines = sorted(set(lines), key=lambda s: s.encode("utf-8"))
    with open(path, "w", encoding="utf-8") as f:
        f.writelines(l + "\n" for l in lines)
    return lines


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--input", required=True, help="raw word list (one word per line; "
                    "anything after the first whitespace, e.g. a definition, is ignored)")
    ap.add_argument("--name", required=True, help="lexicon code, e.g. NSF26")
    ap.add_argument("--alphabet", required=True,
                    help="letter distribution name, e.g. norwegian (see pkg/memento/letterdistributions)")
    ap.add_argument("--out", required=True, help="output directory")
    ap.add_argument("--min-len", type=int, default=2,
                    help="shortest playable word, in tiles (default 2)")
    ap.add_argument("--max-len", type=int, default=15,
                    help="longest playable word, in tiles. 15 for a 15x15 board; use 21 "
                    "for lexica that are also played on SuperCrosswordGame, 0 = no limit")
    ap.add_argument("--prev", help="previous edition's .kwg, to diff against")
    ap.add_argument("--drop-invalid", action="store_true",
                    help="drop words with characters outside the alphabet instead of aborting")
    ap.add_argument("--kwgc", default=os.environ.get("KWGC", os.path.expanduser("~/code/kwgc/kwgc")))
    ap.add_argument("--skip-gaddag-check", action="store_true",
                    help="skip the full GADDAG read-back (it takes a minute or two on big lists)")
    a = ap.parse_args()

    if a.alphabet not in KWGC_ALPHABETS:
        die(f"kwgc has no '{a.alphabet}' tileset; it knows {sorted(KWGC_ALPHABETS)}. "
            "A new alphabet needs a kwgc/wolges change first.")
    if not os.access(a.kwgc, os.X_OK):
        die(f"kwgc not found at {a.kwgc} (cd ~/code/kwgc && make)")
    os.makedirs(a.out, exist_ok=True)
    report = {"name": a.name, "alphabet": a.alphabet, "input": os.path.abspath(a.input)}

    alphabet = load_alphabet(a.alphabet)
    labels = sorted(alphabet, key=len, reverse=True)

    # ---- 1. read and normalize
    raw_lines = 0
    blank_lines = 0
    words = {}  # uppercased word -> tiles
    dupes = 0
    invalid = []
    changed_by_nfc = 0
    with open(a.input, encoding="utf-8-sig") as f:
        for line in f:
            raw_lines += 1
            parts = line.split()
            if not parts:
                blank_lines += 1
                continue
            w = parts[0]
            n = unicodedata.normalize("NFC", w)
            if n != w:
                changed_by_nfc += 1
            w = n.upper()
            if w in words:
                dupes += 1
                continue
            tiles = tokenize(w, labels)
            if tiles is None:
                invalid.append(w)
                continue
            words[w] = tiles
    report.update(raw_lines=raw_lines, blank_lines=blank_lines, duplicates=dupes,
                  nfc_normalized=changed_by_nfc, invalid_chars=len(invalid),
                  invalid_examples=invalid[:20])
    if invalid and not a.drop_invalid:
        bad = collections.Counter(c for w in invalid for c in w
                                  if tokenize(c, labels) is None)
        die(f"{len(invalid)} words contain characters outside the {a.alphabet} alphabet, "
            f"e.g. {invalid[:10]}; offending chars: {dict(bad.most_common(20))}. "
            "Fix the input, pick the right --alphabet, or pass --drop-invalid.")

    # ---- 2. filter by playable length
    too_short = sorted(w for w, t in words.items() if len(t) < a.min_len)
    too_long = sorted(w for w, t in words.items() if a.max_len and len(t) > a.max_len)
    drop = set(too_short) | set(too_long)
    keep = {w: t for w, t in words.items() if w not in drop}
    zero_count = sorted(l for l, c in alphabet.items() if c == 0)
    blank_only = [w for w, t in keep.items() if any(alphabet[x] == 0 for x in t)]
    lens = collections.Counter(len(t) for t in keep.values())
    report.update(too_short=len(too_short), too_short_words=too_short[:50],
                  too_long=len(too_long), too_long_examples=too_long[:10],
                  kept=len(keep), length_histogram=dict(sorted(lens.items())),
                  zero_count_tiles=zero_count, blank_only_words=len(blank_only),
                  blank_only_examples=sorted(blank_only)[:10])
    with open(os.path.join(a.out, "dropped.txt"), "w", encoding="utf-8") as f:
        for w in too_short:
            f.write(f"{w}\ttoo short\n")
        for w in too_long:
            f.write(f"{w}\ttoo long\n")
        for w in invalid:
            f.write(f"{w}\tinvalid chars\n")

    src = os.path.join(a.out, f"{a.name}.txt")
    kept_sorted = sorted_file(list(keep), src)

    # ---- 3. compile
    kwg = os.path.join(a.out, f"{a.name}.kwg")
    kad = os.path.join(a.out, f"{a.name}.kad")
    run_kwgc(a.kwgc, [f"{a.alphabet}-kwg", src, kwg], kwg)
    run_kwgc(a.kwgc, [f"{a.alphabet}-kwg-alpha", src, kad], kad)
    report.update(kwg_bytes=os.path.getsize(kwg), kad_bytes=os.path.getsize(kad))

    # ---- 4. verify: DAWG half of the kwg holds exactly the kept words
    checks = {}
    dawg = set(dump_lines(run_kwgc(a.kwgc, [f"{a.alphabet}-read-kwg", kwg])))
    kept_set = set(kept_sorted)
    checks["kwg_dawg_roundtrip"] = {
        "ok": dawg == kept_set, "count": len(dawg),
        "missing": sorted(kept_set - dawg)[:20], "extra": sorted(dawg - kept_set)[:20]}

    # ---- 5. verify: alpha DAWG holds exactly the sorted-tile anagram keys
    order = {lab: i for i, lab in enumerate(
        [l.strip().split(",")[0] for l in open(os.path.join(LD_DIR, a.alphabet), encoding="utf-8")
         if l.strip()])}
    alphas = {"".join(sorted(t, key=lambda x: order[x])) for t in keep.values()}
    got_alpha = set(dump_lines(run_kwgc(a.kwgc, [f"{a.alphabet}-read-kwg", kad])))
    checks["kad_roundtrip"] = {
        "ok": got_alpha == alphas, "count": len(got_alpha), "expected": len(alphas),
        "missing": sorted(alphas - got_alpha)[:20], "extra": sorted(got_alpha - alphas)[:20]}

    # ---- 6. verify: GADDAG half. For each word and each split point i, the
    # entry is reverse(tiles[:i]) + "?" + tiles[i:] ("?" omitted when i == n).
    if not a.skip_gaddag_check:
        exp_path = os.path.join(a.out, ".gaddag.expected")
        got_path = os.path.join(a.out, ".gaddag.got")
        with open(exp_path, "w", encoding="utf-8") as f:
            for t in keep.values():
                n = len(t)
                for i in range(1, n + 1):
                    f.write("".join(reversed(t[:i])) + ("?" + "".join(t[i:]) if i < n else "") + "\n")
        with open(got_path, "w", encoding="utf-8") as f:
            p = subprocess.run([a.kwgc, f"{a.alphabet}-read-kwg-gaddag", kwg], stdout=f,
                               stderr=subprocess.PIPE, text=True)
        env = dict(os.environ, LC_ALL="C")
        for pth in (exp_path, got_path):
            subprocess.run(["sort", "-u", "-o", pth, pth], env=env, check=True)
        subprocess.run(["sed", "-i", "", "/^time taken/d", got_path], check=True)
        same = subprocess.run(["cmp", "-s", exp_path, got_path]).returncode == 0
        n_got = sum(1 for _ in open(got_path, encoding="utf-8"))
        checks["kwg_gaddag_roundtrip"] = {"ok": same, "entries": n_got}
        if same:
            os.remove(exp_path)
            os.remove(got_path)
    report["checks"] = checks

    # ---- 7. diff against the previous edition
    if a.prev:
        prev = set(dump_lines(run_kwgc(a.kwgc, [f"{a.alphabet}-read-kwg", a.prev])))
        prev_tiles = {w: tokenize(w, labels) for w in prev}
        in_window = {w for w, t in prev_tiles.items()
                     if t and len(t) >= a.min_len and (not a.max_len or len(t) <= a.max_len)}
        added = sorted(kept_set - prev)
        removed = sorted(in_window - kept_set)
        out_of_window = len(prev) - len(in_window)
        for fn, lst in (("added.txt", added), ("removed.txt", removed)):
            with open(os.path.join(a.out, fn), "w", encoding="utf-8") as f:
                f.writelines(w + "\n" for w in lst)
        report["diff_vs_prev"] = {
            "prev": os.path.abspath(a.prev), "prev_words": len(prev),
            "prev_words_outside_length_window": out_of_window,
            "added": len(added), "removed": len(removed),
            "added_examples": added[:: max(1, len(added) // 15)][:15],
            "removed_examples": removed[:: max(1, len(removed) // 15)][:15]}

    with open(os.path.join(a.out, "report.json"), "w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    print(json.dumps(report, indent=2, ensure_ascii=False))
    failed = [k for k, v in checks.items() if not v["ok"]]
    if failed:
        die(f"verification failed: {failed}")
    print(f"\nOK: {len(keep)} words in {kwg} and {kad}, all read-back checks passed.")


if __name__ == "__main__":
    main()
