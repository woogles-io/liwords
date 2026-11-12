#!/usr/bin/env python3
"""Query prod Jaeger's span store (OpenSearch on woogles-jaeger) over ssh.

Jaeger's own HTTP API loads whole traces into a 512 MiB container and gets
OOM-killed by big searches, so this talks to OpenSearch directly and lets it
aggregate. Every query is bounded by a time window.

  ops      count and latency per operation
  rate     spans per interval for one operation (e.g. reads per second)
  tag      top values of one tag, with latency per value
  parents  which operations call this one
  slow     slowest spans of an operation, with trace links
  trace    one trace as a tree
  fetch    sample whole traces into a directory for scripts/jaeger-audit.py

Times: --since 1h (default) or --from/--to (ISO 8601 or epoch seconds).
Tag filters: --where key=value, repeatable, e.g.
  --where otel.scope.name=tournament-cache --where cache.hit=false

Durations are printed in ms. Never print pgx.query.parameters: it holds
session IDs and email addresses.
"""

import argparse
import collections
import datetime as dt
import json
import os
import re
import subprocess
import sys

HOST = "woogles-jaeger"
INDEX = "jaeger-span-*"
UI = "http://jaeger.woogles.io/trace/"


def search(body, host=HOST):
    body.setdefault("timeout", "30s")
    cmd = ["ssh", "-o", "ConnectTimeout=10", "-o", "LogLevel=ERROR", host,
           f"curl -s -H 'Content-Type: application/json' localhost:9200/{INDEX}/_search -d @-"]
    out = subprocess.run(cmd, input=json.dumps(body), capture_output=True, text=True, timeout=120)
    if out.returncode != 0:
        sys.exit(f"ssh {host} failed: {out.stderr.strip()}")
    res = json.loads(out.stdout)
    if "error" in res:
        sys.exit(json.dumps(res["error"], indent=1)[:2000])
    if res.get("timed_out"):
        print("warning: search timed out; results are partial", file=sys.stderr)
    return res


def parse_time(s):
    if re.fullmatch(r"\d+(\.\d+)?", s):
        return float(s)
    return dt.datetime.fromisoformat(s.replace("Z", "+00:00")).timestamp()


def window(args):
    now = dt.datetime.now(dt.timezone.utc).timestamp()
    if args.frm:
        start = parse_time(args.frm)
    else:
        m = re.fullmatch(r"(\d+)([smhd])", args.since)
        if not m:
            sys.exit("--since looks like 30m, 1h, 2d")
        start = now - int(m.group(1)) * {"s": 1, "m": 60, "h": 3600, "d": 86400}[m.group(2)]
    end = parse_time(args.to) if args.to else now
    return int(start * 1000), int(end * 1000)


def tag_filter(key, value=None):
    """A span tag, or a process tag (resource attribute, such as host.name)."""
    def on(path):
        must = [{"term": {f"{path}.key": key}}]
        if value is not None:
            must.append({"term": {f"{path}.value": value}})
        return {"nested": {"path": path, "query": {"bool": {"must": must}}}}
    return {"bool": {"should": [on("tags"), on("process.tags")], "minimum_should_match": 1}}


def filters(args, op=None):
    start, end = window(args)
    f = [{"range": {"startTimeMillis": {"gte": start, "lt": end}}},
         {"term": {"process.serviceName": args.service}}]
    if op:
        f.append({"term": {"operationName": op}})
    for w in args.where or []:
        k, _, v = w.partition("=")
        f.append(tag_filter(k, v if _ else None))
    return f


def latency_aggs():
    return {"p": {"percentiles": {"field": "duration", "percents": [50, 90, 99]}},
            "mx": {"max": {"field": "duration"}},
            "sum": {"sum": {"field": "duration"}}}


def lat(b):
    p = b["p"]["values"]
    ms = lambda k: (p.get(k) or 0) / 1000
    return ms("50.0"), ms("90.0"), ms("99.0"), (b["mx"]["value"] or 0) / 1000


def tags_of(src):
    return {t["key"]: t["value"] for t in src.get("tags") or []}


def cmd_ops(args):
    f = filters(args)
    if args.prefix:
        f.append({"prefix": {"operationName": args.prefix}})
    order = {"count": {"_count": "desc"}, "total": {"sum": "desc"}, "p99": {"p.99": "desc"}}[args.sort]
    res = search({"size": 0, "query": {"bool": {"filter": f}},
                  "aggs": {"ops": {"terms": {"field": "operationName", "size": args.top, "order": order},
                                   "aggs": latency_aggs()}}})
    print(f"{'n':>9} {'p50':>8} {'p90':>8} {'p99':>8} {'max':>9} {'total s':>9}  operation  (ms)")
    for b in res["aggregations"]["ops"]["buckets"]:
        p50, p90, p99, mx = lat(b)
        print(f"{b['doc_count']:>9} {p50:>8.2f} {p90:>8.2f} {p99:>8.2f} {mx:>9.1f} "
              f"{b['sum']['value'] / 1e6:>9.1f}  {b['key'][:90]}")


def cmd_rate(args):
    res = search({"size": 0, "query": {"bool": {"filter": filters(args, args.op)}},
                  "aggs": {"h": {"date_histogram": {"field": "startTimeMillis", "fixed_interval": args.interval},
                                 "aggs": latency_aggs()}}})
    secs = {"s": 1, "m": 60, "h": 3600, "d": 86400}[args.interval[-1]] * int(args.interval[:-1])
    print(f"{'time (UTC)':<17} {'n':>8} {'per s':>8} {'p50':>8} {'p99':>8}  (ms)")
    for b in res["aggregations"]["h"]["buckets"]:
        p50, _, p99, _ = lat(b) if b["doc_count"] else (0, 0, 0, 0)
        t = dt.datetime.fromtimestamp(b["key"] / 1000, dt.timezone.utc).strftime("%m-%d %H:%M:%S")
        print(f"{t:<17} {b['doc_count']:>8} {b['doc_count'] / secs:>8.2f} {p50:>8.2f} {p99:>8.2f}")


def cmd_tag(args):
    res = search({"size": 0, "track_total_hits": True, "query": {"bool": {"filter": filters(args, args.op)}},
                  "aggs": {"t": {"nested": {"path": "tags"}, "aggs": {
                      "k": {"filter": {"term": {"tags.key": args.key}}, "aggs": {
                          "v": {"terms": {"field": "tags.value", "size": args.top}, "aggs": {
                              "span": {"reverse_nested": {}, "aggs": latency_aggs()}}}}}}}}})
    print(f"{res['hits']['total']['value']} spans")
    print(f"{'n':>8} {'p50':>8} {'p99':>8} {'max':>9}  {args.key}  (ms)")
    for b in res["aggregations"]["t"]["k"]["v"]["buckets"]:
        p50, _, p99, mx = lat(b["span"])
        print(f"{b['doc_count']:>8} {p50:>8.2f} {p99:>8.2f} {mx:>9.1f}  {b['key']}")


def cmd_parents(args):
    res = search({"size": args.sample, "_source": ["references"],
                  "query": {"bool": {"filter": filters(args, args.op)}}})
    hits = res["hits"]["hits"]
    ids = [h["_source"]["references"][0]["spanID"] for h in hits if h["_source"].get("references")]
    print(f"{len(hits)} spans sampled, {len(hits) - len(ids)} with no parent span")
    if not ids:
        return
    start, end = window(args)
    res = search({"size": 0, "query": {"bool": {"filter": [
        {"terms": {"spanID": ids}},
        {"range": {"startTimeMillis": {"gte": start - 3600_000, "lt": end}}}]}},
        "aggs": {"o": {"terms": {"field": "operationName", "size": args.top}}}})
    for b in res["aggregations"]["o"]["buckets"]:
        print(f"{b['doc_count']:>8}  {b['key'][:100]}")


def cmd_slow(args):
    f = filters(args, args.op)
    if args.min_ms:
        f.append({"range": {"duration": {"gte": int(args.min_ms * 1000)}}})
    res = search({"size": args.top, "sort": [{"duration": "desc"}],
                  "_source": ["traceID", "startTimeMillis", "duration", "tags"],
                  "query": {"bool": {"filter": f}}})
    show = [k for k in (args.show or "").split(",") if k and k != "pgx.query.parameters"]
    for h in res["hits"]["hits"]:
        s = h["_source"]
        t = dt.datetime.fromtimestamp(s["startTimeMillis"] / 1000, dt.timezone.utc).strftime("%m-%d %H:%M:%S")
        tg = tags_of(s)
        extra = " ".join(f"{k}={tg.get(k)}" for k in show)
        print(f"{s['duration'] / 1000:>9.1f} ms  {t}  {UI}{s['traceID']}  {extra}")


def spans_of(trace_ids, args):
    start, end = window(args)
    res = search({"size": 10000, "query": {"bool": {"filter": [
        {"terms": {"traceID": trace_ids}},
        {"range": {"startTimeMillis": {"gte": start - 3600_000, "lt": end + 3600_000}}}]}}})
    by = collections.defaultdict(list)
    for h in res["hits"]["hits"]:
        by[h["_source"]["traceID"]].append(h["_source"])
    return by


def cmd_trace(args):
    if not args.frm:
        args.since = "2d"
    spans = spans_of([args.trace_id], args)[args.trace_id]
    if not spans:
        sys.exit("no spans; Jaeger keeps about a day")
    kids = collections.defaultdict(list)
    ids = {s["spanID"] for s in spans}
    for s in spans:
        refs = s.get("references") or []
        p = refs[0]["spanID"] if refs and refs[0]["spanID"] in ids else None
        kids[p].append(s)
    t0 = min(s["startTime"] for s in spans)
    print(f"{'start':>9} {'dur':>9}  (ms)  {UI}{args.trace_id}")

    def show(pid, depth):
        for s in sorted(kids[pid], key=lambda s: s["startTime"]):
            name = s["operationName"].split("\n")[0][:90]
            print(f"{(s['startTime'] - t0) / 1000:>9.1f} {s['duration'] / 1000:>9.1f}  {'  ' * depth}{name}")
            show(s["spanID"], depth + 1)
    show(None, 0)


def no_parent():
    return {"bool": {"must_not": {"nested": {"path": "references", "query": {"exists": {"field": "references.spanID"}}}}}}


def cmd_fetch(args):
    os.makedirs(args.out, exist_ok=True)
    f = filters(args) + [no_parent()]
    if args.prefix:
        f.append({"prefix": {"operationName": args.prefix}})
    res = search({"size": 0, "query": {"bool": {"filter": f}},
                  "aggs": {"ops": {"terms": {"field": "operationName", "size": args.ops}, "aggs": {
                      "s": {"top_hits": {"size": args.per_op, "sort": [{"startTimeMillis": "desc"}],
                                         "_source": ["traceID"]}}}}}})
    trace_ids = sorted({h["_source"]["traceID"]
                        for b in res["aggregations"]["ops"]["buckets"] for h in b["s"]["hits"]["hits"]})
    print(f"{len(trace_ids)} traces from {len(res['aggregations']['ops']['buckets'])} root operations",
          file=sys.stderr)
    written = 0
    for i in range(0, len(trace_ids), 50):
        for tid, spans in spans_of(trace_ids[i:i + 50], args).items():
            procs = {}  # serviceName -> (processID, process), as Jaeger's API returns them
            for s in spans:
                p = s.pop("process", None) or {}
                name = p.get("serviceName")
                if name not in procs:
                    procs[name] = (f"p{len(procs) + 1}", p)
                s["processID"] = procs[name][0]
            with open(os.path.join(args.out, tid + ".json"), "w") as fh:
                json.dump({"traceID": tid, "spans": spans,
                           "processes": {pid: p for pid, p in procs.values()}}, fh)
            written += 1
    print(f"wrote {written} traces to {args.out}; now run: "
          f"scripts/jaeger-audit.py --cache-dir {args.out}", file=sys.stderr)


def main():
    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--since", default="1h", help="window ending now: 30m, 1h, 2d (default 1h)")
    common.add_argument("--from", dest="frm", help="window start, ISO 8601 or epoch seconds")
    common.add_argument("--to", help="window end (default now)")
    common.add_argument("--service", default="liwords")
    common.add_argument("--where", action="append", metavar="KEY=VALUE", help="tag filter, repeatable")
    common.add_argument("--top", type=int, default=30)

    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("ops", parents=[common])
    s.add_argument("--prefix", help="only operations starting with this")
    s.add_argument("--sort", choices=["count", "total", "p99"], default="count")
    s = sub.add_parser("rate", parents=[common])
    s.add_argument("op")
    s.add_argument("--interval", default="5m", help="bucket size: 30s, 1m, 5m, 1h")
    s = sub.add_parser("tag", parents=[common])
    s.add_argument("op")
    s.add_argument("key")
    s = sub.add_parser("parents", parents=[common])
    s.add_argument("op")
    s.add_argument("--sample", type=int, default=2000, help="spans to look up parents for (max 10000)")
    s = sub.add_parser("slow", parents=[common])
    s.add_argument("op")
    s.add_argument("--min-ms", type=float)
    s.add_argument("--show", help="comma-separated tags to print per span")
    s = sub.add_parser("trace", parents=[common])
    s.add_argument("trace_id")
    s = sub.add_parser("fetch", parents=[common])
    s.add_argument("--out", required=True)
    s.add_argument("--ops", type=int, default=60, help="root operations to sample")
    s.add_argument("--per-op", type=int, default=20, help="latest traces per root operation")
    s.add_argument("--prefix", help="only root operations starting with this")
    args = p.parse_args()
    globals()["cmd_" + args.cmd](args)


if __name__ == "__main__":
    main()
