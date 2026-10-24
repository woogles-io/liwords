#!/usr/bin/env python3
"""Scan recent Jaeger traces for performance smells.

Pulls a sample of traces for every operation of a service through Jaeger's
HTTP API, then reports:

  repeats    the same query run many times in one request (N+1 loops)
  dupes      the same query with the same arguments run more than once
  self       spans whose time isn't covered by any child span -- uninstrumented
             work such as outbound HTTP calls or CPU-heavy loops
  endpoints  slowest root operations (p50 / p95 / max)
  queries    slowest individual queries

Query parameters are used for duplicate detection but never printed: they
include session IDs and email addresses.

Usage:
  scripts/jaeger-audit.py                       # last 7 days, 40 traces/op
  scripts/jaeger-audit.py --lookback 1d --limit 100
  scripts/jaeger-audit.py --cache-dir /tmp/traces   # reuse a previous fetch
  scripts/jaeger-audit.py --only repeats,self
  scripts/jaeger-audit.py --trace <trace id>     # print one trace as a tree
"""

import argparse
import collections
import concurrent.futures
import json
import os
import re
import statistics
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request

# Operations that are never the root of a request; skip fetching by them.
NON_ROOT_PREFIXES = ("query", "prepare", "pool.", "middleware", "S3.", "nats.",
                     "broadcast.rpc", "game.", "cache.")
# Spans that only wrap the next one; their self time is meaningless.
WRAPPER_PREFIXES = ("middleware.", "POST ", "GET ")
REPORTS = ("repeats", "dupes", "self", "endpoints", "queries")


def get_json(url, timeout=60):
    with urllib.request.urlopen(url, timeout=timeout) as resp:
        return json.load(resp)


def fetch(args):
    base = args.url.rstrip("/") + "/api"
    ops = get_json(f"{base}/services/{urllib.parse.quote(args.service)}/operations")["data"]
    ops = [o for o in ops if not o.startswith(NON_ROOT_PREFIXES)]
    print(f"fetching up to {args.limit} traces for {len(ops)} operations "
          f"(lookback {args.lookback}) into {args.cache_dir}", file=sys.stderr)

    def one(op):
        q = urllib.parse.urlencode({"service": args.service, "operation": op,
                                    "limit": args.limit, "lookback": args.lookback})
        try:
            traces = get_json(f"{base}/traces?{q}")["data"] or []
        except Exception as e:  # keep going; one bad operation shouldn't stop the audit
            print(f"  {op[:70]}: {e}", file=sys.stderr)
            return
        for t in traces:
            path = os.path.join(args.cache_dir, t["traceID"] + ".json")
            if not os.path.exists(path):
                with open(path, "w") as f:
                    json.dump(t, f)

    with concurrent.futures.ThreadPoolExecutor(8) as ex:
        list(ex.map(one, ops))


def span_name(op):
    m = re.search(r"name: (\w+)", op)
    if m:
        return "q:" + m.group(1)
    return op.split("\n")[0][:80]


def tags(span):
    return {t["key"]: t["value"] for t in span.get("tags", [])}


def parent_id(span):
    refs = span.get("references") or []
    return refs[0]["spanID"] if refs else None


def self_time(span, children):
    """Span duration minus the union of its children's intervals (µs)."""
    start, end = span["startTime"], span["startTime"] + span["duration"]
    covered, cur_s, cur_e = 0, None, None
    for c in sorted(children, key=lambda c: c["startTime"]):
        s, e = max(c["startTime"], start), min(c["startTime"] + c["duration"], end)
        if e <= s:
            continue
        if cur_e is None or s > cur_e:
            if cur_e is not None:
                covered += cur_e - cur_s
            cur_s, cur_e = s, e
        else:
            cur_e = max(cur_e, e)
    if cur_e is not None:
        covered += cur_e - cur_s
    return span["duration"] - covered


def load(cache_dir):
    for f in os.listdir(cache_dir):
        if f.endswith(".json"):
            with open(os.path.join(cache_dir, f)) as fh:
                yield json.load(fh)


def pct(values, p):
    v = sorted(values)
    return v[int(p * (len(v) - 1))]


def analyze(args):
    root_durs = collections.defaultdict(list)  # root -> [ms]
    repeats = collections.defaultdict(list)    # (root, query) -> [(count, distinct args, trace)]
    dupes = collections.defaultdict(lambda: [0, 0])  # (root, query) -> [traces, extra calls]
    selfs = collections.defaultdict(list)      # (root, span) -> [(self ms, trace)]
    queries = collections.defaultdict(list)    # query -> [(ms, root, trace)]

    for t in load(args.cache_dir):
        spans = t["spans"]
        roots = [s for s in spans if not s.get("references")]
        if not roots:
            continue
        root = max(roots, key=lambda s: s["duration"])
        rname = span_name(root["operationName"])
        tid = t["traceID"]
        root_durs[rname].append(root["duration"] / 1000)

        kids = collections.defaultdict(list)
        for s in spans:
            kids[parent_id(s)].append(s)

        counts = collections.Counter()
        arg_counts = collections.Counter()
        best_self = None
        for s in spans:
            n = span_name(s["operationName"])
            if n.startswith("q:"):
                counts[n] += 1
                arg_counts[(n, tags(s).get("pgx.query.parameters", ""))] += 1
                queries[n].append((s["duration"] / 1000, rname, tid))
            elif s is not root and not n.startswith(WRAPPER_PREFIXES) and n != "pool.acquire":
                st = self_time(s, kids[s["spanID"]])
                if best_self is None or st > best_self[0]:
                    best_self = (st, n)

        for n, k in counts.items():
            if k >= args.repeat_threshold:
                distinct = sum(1 for (qn, _) in arg_counts if qn == n)
                repeats[(rname, n)].append((k, distinct, tid))
        seen = set()
        for (n, _), k in arg_counts.items():
            if k > 1:
                d = dupes[(rname, n)]
                d[1] += k - 1
                if n not in seen:
                    d[0] += 1
                    seen.add(n)
        if best_self:
            selfs[(rname, best_self[1])].append((best_self[0] / 1000, tid))

    link = lambda tid: f"{args.url.rstrip('/')}/trace/{tid}"
    only = set(args.only.split(",")) if args.only else set(REPORTS)
    top = args.top

    if "repeats" in only:
        print(f"\n== Repeated queries in one request (>= {args.repeat_threshold}x) — likely N+1")
        print(f"{'max':>5} {'med':>5} {'traces':>9}  endpoint :: query  [distinct args in worst]")
        rows = sorted(repeats.items(), key=lambda kv: -max(x[0] for x in kv[1]))
        for (r, n), v in rows[:top]:
            worst = max(v)
            print(f"{worst[0]:>5} {statistics.median(x[0] for x in v):>5.0f} "
                  f"{len(v):>4}/{len(root_durs[r]):<4}  {r} :: {n}  [{worst[1]}]  {link(worst[2])}")

    if "dupes" in only:
        print("\n== Identical query + arguments repeated in one request — cacheable or redundant")
        rows = sorted(dupes.items(), key=lambda kv: -kv[1][1])
        for (r, n), (tr, extra) in rows[:top]:
            print(f"{tr:>4}/{len(root_durs[r]):<4} traces, {extra:>5} extra calls  {r} :: {n}")

    if "self" in only:
        print(f"\n== Uninstrumented time (span time not covered by children, median >= {args.self_ms}ms)")
        print(f"{'p50':>8} {'max':>8} {'n':>4}  endpoint :: span")
        rows = [(statistics.median(x[0] for x in v), k, v) for k, v in selfs.items()]
        rows = sorted((r for r in rows if r[0] >= args.self_ms), key=lambda r: -r[0])
        for med, (r, n), v in rows[:top]:
            worst = max(v)
            print(f"{med:>8.1f} {worst[0]:>8.1f} {len(v):>4}  {r} :: {n}  {link(worst[1])}")

    if "endpoints" in only:
        print("\n== Slowest endpoints (ms)")
        print(f"{'p50':>9} {'p95':>9} {'max':>9} {'n':>4}  endpoint")
        rows = [(statistics.median(v), r, v) for r, v in root_durs.items() if len(v) >= args.min_samples]
        for med, r, v in sorted(rows, key=lambda x: -x[0])[:top]:
            print(f"{med:>9.1f} {pct(v, .95):>9.1f} {max(v):>9.1f} {len(v):>4}  {r}")

    if "queries" in only:
        print("\n== Slowest queries (ms)")
        print(f"{'p50':>8} {'max':>8} {'n':>5}  query  (worst endpoint)")
        rows = sorted(queries.items(), key=lambda kv: -max(x[0] for x in kv[1]))
        for n, v in rows[:top]:
            worst = max(v)
            print(f"{statistics.median(x[0] for x in v):>8.1f} {worst[0]:>8.1f} {len(v):>5}  "
                  f"{n}  ({worst[1]})  {link(worst[2])}")


def print_tree(args):
    base = args.url.rstrip("/") + "/api"
    try:
        t = get_json(f"{base}/traces/{args.trace}")["data"][0]
    except urllib.error.HTTPError as e:
        sys.exit(f"trace {args.trace}: {e}")
    spans = t["spans"]
    kids = collections.defaultdict(list)
    for s in spans:
        kids[parent_id(s)].append(s)
    t0 = min(s["startTime"] for s in spans)
    print(f"{'start':>8} {'dur':>8} {'self':>8}  (ms)")

    def show(pid, depth):
        for s in sorted(kids[pid], key=lambda s: s["startTime"]):
            n = span_name(s["operationName"])
            if n == "pool.acquire":
                continue
            st = self_time(s, kids[s["spanID"]])
            print(f"{(s['startTime'] - t0) / 1000:>8.1f} {s['duration'] / 1000:>8.1f} "
                  f"{st / 1000:>8.1f}  {'  ' * depth}{n}")
            show(s["spanID"], depth + 1)

    show(None, 0)


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--url", default="http://jaeger.woogles.io")
    p.add_argument("--service", default="liwords")
    p.add_argument("--lookback", default="7d", help="e.g. 1h, 1d, 7d")
    p.add_argument("--limit", type=int, default=40, help="traces to fetch per operation")
    p.add_argument("--cache-dir", help="where traces are stored; reused if it already has some")
    p.add_argument("--refetch", action="store_true", help="fetch even if --cache-dir has traces")
    p.add_argument("--only", help=f"comma-separated subset of: {','.join(REPORTS)}")
    p.add_argument("--top", type=int, default=25, help="rows per report")
    p.add_argument("--repeat-threshold", type=int, default=4)
    p.add_argument("--self-ms", type=float, default=20, help="min median self time to report")
    p.add_argument("--min-samples", type=int, default=3, help="min traces for an endpoint to be ranked")
    p.add_argument("--trace", help="print one trace as a tree and exit")
    args = p.parse_args()

    if args.trace:
        print_tree(args)
        return
    if not args.cache_dir:
        args.cache_dir = tempfile.mkdtemp(prefix="jaeger-audit-")
    os.makedirs(args.cache_dir, exist_ok=True)
    if args.refetch or not any(f.endswith(".json") for f in os.listdir(args.cache_dir)):
        fetch(args)
    analyze(args)


if __name__ == "__main__":
    main()
