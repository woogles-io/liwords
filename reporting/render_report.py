#!/usr/bin/env python3
"""Render monthly reporting CSVs into an HTML email body + charts.

Reusable across reports: add an entry to REPORTS keyed by the query basename
(the part of the CSV filename before the trailing _<timestamp>) to control
which fields get charted. CSVs without an entry still get the HTML table,
just no chart.

Usage (called by monthly_report.sh):
    render_report.py --out-html body.html --out-dir _results csv [csv ...]

Writes one <basename>.png per charted CSV into --out-dir, prints each PNG
path on stdout (one per line), and writes the full HTML body to --out-html.
The HTML references each chart as <img src="cid:<basename>"> — send_mail.py
attaches the PNGs inline with matching Content-IDs.
"""
import argparse
import re
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import pandas as pd
import seaborn as sns

# Seaborn's colorblind palette: validated CVD-safe (worst adjacent-pair
# deltaE 37 protan) on a white surface. Assign hues in this fixed order.
# Slot 5 swaps the default tan (#ca9161, fails the chroma floor — reads gray)
# for the palette's sky blue, revalidated as a 6-slot set.
PALETTE = sns.color_palette("colorblind")
PALETTE[5] = sns.color_palette("colorblind")[9]

TABLE_ROWS = 12  # most recent months shown in the email body

REPORTS = {
    "games_per_month": {
        "date_col": "month",
        "chart_title": "Woogles games per month",
        # First `split` fields go on the big top panel, the rest on the smaller
        # bottom panel: the totals (~350k) would flatten the niche game types
        # (~5k) into the baseline on a shared y-axis.
        "chart_fields": [
            "game_count",
            "pvp_game_count",
            "correspondence_game_count",
            "league_game_count",
            "tournament_game_count",
            "annotated_count",
        ],
        "split": 2,
        "ylabel": "games",
    },
    "mau_reporting": {
        "date_col": "month",
        "chart_title": "Woogles monthly active users",
        # Sitewide/OMGWords MAU (~15k) on top; puzzle and annotation MAU
        # (hundreds to low thousands) below so they stay readable.
        "chart_fields": [
            "mau",
            "mau_omgwords",
            "mau_omgwords_vs_human",
            "mau_puzzles",
            "mau_annotators",
        ],
        "split": 3,
        "ylabel": "active users",
    },
}


def report_key(csv_path):
    """games_per_month_20260705_105458.csv -> games_per_month"""
    stem = Path(csv_path).stem
    return re.sub(r"_\d{8}(_\d{6})?$", "", stem)


def pretty_col(name):
    return name.replace("_", " ")


def load_csv(csv_path, date_col):
    df = pd.read_csv(csv_path)
    if date_col and date_col in df.columns:
        df[date_col] = pd.to_datetime(df[date_col], utc=True, format="mixed")
        df = df.sort_values(date_col)
    return df


def html_table(df, date_col):
    recent = df.tail(TABLE_ROWS).iloc[::-1]  # newest month first
    td = ('padding:3px 8px;border-bottom:1px solid #e5e5e0;'
          'font:12px/1.4 Menlo,Consolas,monospace;text-align:right;'
          'white-space:nowrap;')
    th = td + 'color:#666;font-weight:600;border-bottom:2px solid #ccc;'

    cells = []
    cells.append("<tr>" + "".join(
        f'<th style="{th}">{pretty_col(c)}</th>' for c in recent.columns) + "</tr>")
    for _, row in recent.iterrows():
        tds = []
        for col in recent.columns:
            v = row[col]
            if col == date_col:
                partial = v.strftime("%Y-%m") == pd.Timestamp.now(tz="UTC").strftime("%Y-%m")
                v = v.strftime("%Y-%m") + (" (partial)" if partial else "")
            elif isinstance(v, float) and v == int(v):
                v = f"{int(v):,}"
            elif isinstance(v, float):
                v = f"{v:,.2f}"
            elif isinstance(v, int):
                v = f"{v:,}"
            tds.append(f'<td style="{td}">{v}</td>')
        cells.append("<tr>" + "".join(tds) + "</tr>")
    return ('<table style="border-collapse:collapse;margin:8px 0 16px;">'
            + "".join(cells) + "</table>")


def render_chart(df, key, cfg, out_dir):
    date_col = cfg["date_col"]
    fields = cfg["chart_fields"]
    if date_col not in df.columns or not any(f in df.columns for f in fields):
        return None

    # Drop the in-progress current month: a partial count plotted next to
    # complete months reads as a cliff-dive, not as "month not over yet".
    month_start = pd.Timestamp.now(tz="UTC").normalize().replace(day=1)
    plot_df = df[df[date_col] < month_start]
    if plot_df.empty:
        plot_df = df

    # Where series magnitudes differ wildly, one panel would flatten the small
    # series into the baseline. Two aligned panels, same x-axis, hues fixed
    # per series across panels; cfg["split"] says where the big panel ends.
    # Missing columns (older CSVs) are skipped per panel, but each field keeps
    # the hue of its chart_fields position so colors are stable across runs.
    split = cfg.get("split", 2)
    big = [f for f in fields[:split] if f in df.columns]
    small = [f for f in fields[split:] if f in df.columns]
    panel_fields_list = [p for p in (big, small) if p]
    sns.set_theme(style="whitegrid", context="notebook")
    if len(panel_fields_list) == 2:
        fig, axes = plt.subplots(
            2, 1, figsize=(11, 7.5), sharex=True, height_ratios=[3, 2])
    else:
        fig, ax = plt.subplots(figsize=(11, 5.5))
        axes = [ax]
    ax_top = axes[0]
    panels = list(zip(axes, panel_fields_list))

    # {:g} keeps 1.5k / drops 2.0k -> 2k; plain rounding would label both
    # 1,500 and 2,000 as "2k" on low-magnitude axes.
    kfmt = mticker.FuncFormatter(
        lambda v, _: f"{round(v/1000, 1):g}k" if v >= 1000 else f"{v:,.0f}")
    for ax, panel_fields in panels:
        for field in panel_fields:
            ax.plot(plot_df[date_col], plot_df[field],
                    color=PALETTE[fields.index(field) % len(PALETTE)],
                    linewidth=2, label=pretty_col(field))
        ax.yaxis.set_major_formatter(kfmt)
        ax.set_ylabel(cfg.get("ylabel", ""))
        ax.set_xlabel("")
        ax.margins(x=0.01)
        ax.set_ylim(bottom=0)
        ax.legend(loc="upper left", frameon=True, framealpha=0.9)

    last_full = plot_df[date_col].max()
    ax_top.set_title(
        f'{cfg.get("chart_title", pretty_col(key))} '
        f'(complete months through {last_full:%b %Y})', fontsize=14, pad=12)
    sns.despine(fig)
    fig.tight_layout()

    out_png = Path(out_dir) / f"{key}.png"
    fig.savefig(out_png, dpi=160)
    plt.close(fig)
    return out_png


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("csvs", nargs="+")
    parser.add_argument("--out-html", required=True)
    parser.add_argument("--out-dir", required=True)
    parser.add_argument("--heading", default="")
    args = parser.parse_args()

    sections = []
    if args.heading:
        sections.append(
            f'<p style="font:14px/1.5 -apple-system,Helvetica,Arial,sans-serif;">'
            f'{args.heading}</p>')

    pngs = []
    for csv_path in args.csvs:
        key = report_key(csv_path)
        cfg = REPORTS.get(key, {})
        df = load_csv(csv_path, cfg.get("date_col"))

        sections.append(
            f'<h3 style="font:600 15px -apple-system,Helvetica,Arial,sans-serif;'
            f'margin:20px 0 4px;">{pretty_col(key)}</h3>')

        if cfg:
            png = render_chart(df, key, cfg, args.out_dir)
            if png:
                pngs.append(png)
                sections.append(
                    f'<img src="cid:{key}" alt="{pretty_col(key)} chart" '
                    f'style="max-width:100%;height:auto;" width="880">')

        date_col = cfg.get("date_col")
        sections.append(
            f'<p style="font:12px -apple-system,Helvetica,Arial,sans-serif;'
            f'color:#666;margin:10px 0 2px;">Most recent {TABLE_ROWS} months '
            f'(full history in the attached CSV):</p>')
        sections.append(html_table(df, date_col))

    Path(args.out_html).write_text(
        '<div style="overflow-x:auto;">' + "\n".join(sections) + "</div>")
    for p in pngs:
        print(p)


if __name__ == "__main__":
    main()
