#!/usr/bin/env python3
"""Render the monthly report's HTML body to a self-contained PDF.

Called by monthly_report.sh after render_report.py, so the same report that goes
out as email is also attached as a PDF that can be posted elsewhere (Discord).

Two things the email body needs before it can be printed:

  * Charts are referenced as <img src="cid:KEY">, because send_mail.py attaches
    the PNGs with matching Content-IDs. Outside an email client those resolve to
    nothing, so each is inlined as a base64 data: URI and the PDF stands alone.

  * A table wider than the page is CLIPPED in print, not wrapped, and
    games_per_month is 40 columns. Widening the page to fit it was tried and
    rejected: a 42in-wide page previews in Discord as an unreadable strip and
    strands the charts in whitespace. Instead the page stays landscape letter-ish
    and an over-wide table is split into column groups that each fit at full
    size, repeating the date column as the key. Tables are measured in a real
    headless-browser pass; estimating widths from character counts was off ~2x.

Usage: report_pdf.py <body.html> <out.pdf> [subtitle]
       body.html's sibling <key>.png files are the charts to inline.
"""
import base64
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path

CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
CSS_PX_PER_IN = 96          # Chrome lays out CSS px at 96/in regardless of @page
PAGE_W_IN = 17.0
PAGE_H_IN = 11.0
MARGIN_IN = 0.45
CHART_W_IN = 15.0
# Printable height is PAGE_H - 2*MARGIN; leave room for the title block and the
# section heading above the chart, or it gets bumped to a page of its own.
CHART_MAX_H_IN = 8.6
AVAIL_PX = (PAGE_W_IN - 2 * MARGIN_IN) * CSS_PX_PER_IN
# Below this, shrinking makes 8px text illegible - split into column groups instead.
MIN_SCALE = 0.8


def split_wide(html, i, width):
    """Split table i into column groups that each fit, repeating column 0."""
    m = re.search(r'(<table data-i="%d"[^>]*>)(.*?)</table>' % i, html, flags=re.S)
    if not m:
        return html
    open_tag, guts = m.group(1), m.group(2)
    grid = [re.findall(r"(<t[dh][^>]*>.*?</t[dh]>)", r, flags=re.S)
            for r in re.findall(r"<tr>(.*?)</tr>", guts, flags=re.S)]
    grid = [r for r in grid if r]
    if not grid:
        return html
    ncols = max(len(r) for r in grid)
    per = max(2, int(ncols * AVAIL_PX / width) - 1)
    groups = [list(range(s, min(s + per, ncols))) for s in range(1, ncols, per)]
    out = []
    for cols in groups:
        body = "".join(
            "<tr>" + r[0] + "".join(r[c] for c in cols if c < len(r)) + "</tr>"
            for r in grid)
        out.append(
            f'<p style="font:11px -apple-system,Helvetica,Arial,sans-serif;'
            f'color:#666;margin:10px 0 2px;">columns {cols[0]}-{cols[-1]} '
            f'of {ncols - 1}</p>{open_tag}{body}</table>')
    print(f"  table {i}: {width}px / {ncols} cols -> {len(groups)} column groups",
          file=sys.stderr)
    return html.replace(m.group(0), "".join(out))


def main():
    body_html, out_pdf = sys.argv[1], sys.argv[2]
    subtitle = sys.argv[3] if len(sys.argv) > 3 else ""
    img_dir = Path(body_html).parent
    html = Path(body_html).read_text()

    def inline(m):
        png = img_dir / f"{m.group(1)}.png"
        if not png.exists():
            return m.group(0)
        b64 = base64.b64encode(png.read_bytes()).decode()
        return f'src="data:image/png;base64,{b64}"'

    html, n_charts = re.subn(r'src="cid:([^"]+)"', inline, html)

    # Index the tables so the measuring pass can report widths back per table.
    idx = [0]

    def tag(m):
        i = idx[0]
        idx[0] += 1
        return f'<table data-i="{i}" '

    html = re.sub(r"<table ", tag, html)

    def build(doc, measure, scales=None):
        if scales:
            def wrap(m):
                s = scales.get(int(m.group(1)), 1.0)
                return (f'<div style="transform:scale({s:.4f});'
                        f'transform-origin:left top"><table data-i="{m.group(1)}" ')
            doc = re.sub(r'<table data-i="(\d+)" ', wrap, doc)
            doc = doc.replace("</table>", "</table></div>")
        style = f"""
  @page {{ size: {PAGE_W_IN:.1f}in {PAGE_H_IN}in; margin: {MARGIN_IN}in; }}
  body {{ font: 13px/1.5 -apple-system, Helvetica, Arial, sans-serif; color:#111; }}
  h1 {{ font-size: 20px; margin: 0 0 2px; }}
  h3 {{ page-break-after: avoid; font-size: 17px !important; }}
  h3 ~ h3 {{ page-break-before: always; }}
  /* Constrain height as well as width: a chart taller than the printable area
     gets pushed whole to the next page by page-break-inside, leaving a blank
     one behind. max-height keeps the aspect ratio and guarantees it fits. */
  img {{ width: auto !important; max-width: {CHART_W_IN}in;
         max-height: {CHART_MAX_H_IN}in; page-break-inside: avoid;
         display: block; margin: 4px 0 10px; }}
  table {{ border-collapse: collapse; }}
  table td, table th {{ padding: 2px 5px !important; font-size: 8px !important;
                        white-space: nowrap; }}
  div[style*="overflow-x"] {{ overflow: visible !important; }}
  .sub {{ color: #666; font-size: 12px; margin: 0 0 14px; }}
"""
        probe = ("<script>window.onload=function(){var o={};"
                 "document.querySelectorAll('table').forEach(function(t){"
                 "o[t.dataset.i]=Math.ceil(t.getBoundingClientRect().width)});"
                 "document.title='W:'+JSON.stringify(o)}</script>") if measure else ""
        return f"""<!doctype html>
<html><head><meta charset="utf-8"><title>Woogles monthly reporting</title>
<style>{style}</style>{probe}</head><body>
<h1>Woogles monthly reporting</h1>
<p class="sub">{subtitle}</p>
{doc}
</body></html>"""

    def chrome(args, timeout):
        return subprocess.run([CHROME, "--headless", "--disable-gpu"] + args,
                              capture_output=True, text=True, timeout=timeout)

    tmp = Path(tempfile.mkdtemp())

    def measure(doc):
        p = tmp / "probe.html"
        p.write_text(build(doc, measure=True))
        dom = chrome(["--virtual-time-budget=4000", "--dump-dom", f"file://{p}"],
                     120).stdout
        m = re.search(r"<title>W:(\{.*?\})</title>", dom)
        if not m:
            print("WARNING: table measuring pass failed", file=sys.stderr)
            return {}
        return {int(k): v for k, v in json.loads(m.group(1)).items()}

    widths = measure(html)
    for i, w in sorted(widths.items()):
        if w > AVAIL_PX and AVAIL_PX / w < MIN_SCALE:
            html = split_wide(html, i, w)
            widths = measure(html)          # indices are stable; widths change

    # Anything still a little over the page gets scaled the rest of the way.
    scales = {i: (min(1.0, AVAIL_PX / w) if w > AVAIL_PX else 1.0)
              for i, w in widths.items()}
    for i, s in sorted(scales.items()):
        if s < 1.0:
            print(f"  table {i}: scaled to {s:.2f} to fit", file=sys.stderr)
    print(f"{n_charts} charts, {len(widths)} tables", file=sys.stderr)

    # Kept next to the PDF: the exact page Chrome printed, for debugging layout.
    # Must be absolute - a relative file:// URL silently renders Chrome's error
    # page into the PDF instead of the report.
    final = Path(out_pdf).resolve().with_suffix(".print.html")
    final.write_text(build(html, measure=False, scales=scales))
    chrome(["--no-pdf-header-footer", f"--print-to-pdf={out_pdf}",
            f"file://{final}"], 180)
    if not Path(out_pdf).exists():
        print("ERROR: chrome produced no PDF", file=sys.stderr)
        return 1
    print(out_pdf)
    return 0


if __name__ == "__main__":
    sys.exit(main())
