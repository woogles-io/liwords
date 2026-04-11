# =============================================================================
# Woogles OBS Text Source Updater
# =============================================================================
#
# WHAT THIS IS
# ------------
# An alternative to the browser-source URLs provided on the broadcast director
# panel.  Browser sources each run a full Chromium renderer; this script uses
# one SSE connection per slot and pushes values directly into lightweight OBS
# Text (GDI+ / FreeType 2) sources instead, which are far cheaper to render.
#
# The browser-source URLs still work exactly as before — this is purely an
# opt-in alternative for setups where browser-source overhead matters (e.g.
# when several webcams and capture cards are already active).
#
# LIMITATIONS vs BROWSER SOURCES
# --------------------------------
# • blank1 / blank2: colored lowercase letters (blank designations) are not
#   possible in a plain text source.  Use a browser source for those fields.
# • last_play marquee: the browser source scrolls automatically.  With a text
#   source you can apply OBS's built-in "Scroll" filter to the source instead.
#
# SETUP
# -----
# 1. In OBS, create one Text (GDI+) or Text (FreeType 2) source for each field
#    you want to display.  Name them however you like.
# 2. Go to Tools → Scripts → "+" → select this file.
# 3. Fill in the fields in the Scripts panel on the right:
#      Base URL      — e.g. https://woogles.io
#      Broadcast slug — the short identifier for your broadcast event
#      Slot name     — which OBS slot to follow (e.g. "main", "table1")
#      Source: <field> — type the exact name of the OBS source for that field;
#                        leave blank to ignore that field.
# 4. The script connects immediately; sources update as plays are made.
#
# PYTHON REQUIREMENT
# ------------------
# OBS → Tools → Scripts → Python Settings must point to a Python 3.6+ install.
# No third-party packages are needed — only stdlib (urllib, threading, json).
#
# =============================================================================

import obspython as obs
import threading
import urllib.request
import json
import time


# ---------------------------------------------------------------------------
# Fields served by the Woogles SSE endpoint
# ---------------------------------------------------------------------------

FIELDS = [
    "score",
    "p1_score",
    "p2_score",
    "unseen_tiles",
    "unseen_count",
    "last_play",
    "blank1",    # plain text only — no color; use browser source for coloring
    "blank2",    # plain text only — no color; use browser source for coloring
]

FIELD_LABELS = {
    "score":        "score  (combined, e.g. 345 - 298)",
    "p1_score":     "p1_score  (player 1 score)",
    "p2_score":     "p2_score  (player 2 score)",
    "unseen_tiles": "unseen_tiles  (tile list)",
    "unseen_count": "unseen_count  (tile/vowel/consonant counts)",
    "last_play":    "last_play  (add a Scroll filter for marquee)",
    "blank1":       "blank1  (no colored letters — use browser src for that)",
    "blank2":       "blank2  (no colored letters — use browser src for that)",
}


# ---------------------------------------------------------------------------
# Module-level state  (all written on the OBS main thread except _latest)
# ---------------------------------------------------------------------------

_lock             = threading.Lock()
_latest: dict     = {}          # field → most recent value from SSE
_sse_thread       = None
_stop_event       = threading.Event()

# Snapshot of settings written by script_update (main thread) and read by
# _flush_to_sources (OBS timer, also main thread) — no lock needed.
_cfg: dict        = {}


# ---------------------------------------------------------------------------
# SSE connection (runs on a daemon thread)
# ---------------------------------------------------------------------------

def _sse_loop(url: str, stop_event: threading.Event) -> None:
    """Open the SSE /events endpoint and buffer every JSON payload."""
    while not stop_event.is_set():
        try:
            req = urllib.request.Request(
                url,
                headers={"Accept": "text/event-stream", "Cache-Control": "no-cache"},
            )
            with urllib.request.urlopen(req, timeout=60) as resp:
                buf = b""
                while not stop_event.is_set():
                    chunk = resp.read(4096)
                    if not chunk:
                        break
                    buf += chunk
                    # SSE messages are separated by blank lines (\n\n)
                    while b"\n\n" in buf:
                        msg, buf = buf.split(b"\n\n", 1)
                        for line in msg.split(b"\n"):
                            if line.startswith(b"data:"):
                                raw = line[5:].strip()
                                if not raw or raw == b":heartbeat":
                                    continue
                                try:
                                    data = json.loads(raw)
                                    with _lock:
                                        _latest.update(data)
                                except Exception:
                                    pass
        except Exception:
            pass  # network error, server restart, etc.
        if not stop_event.is_set():
            time.sleep(5)   # wait before reconnecting


def _start_sse(base_url: str, slug: str, slot: str) -> None:
    global _sse_thread, _stop_event
    # Stop any existing connection first
    _stop_event.set()
    if _sse_thread and _sse_thread.is_alive():
        _sse_thread.join(timeout=3)
    _stop_event = threading.Event()
    url = f"{base_url.rstrip('/')}/api/broadcasts/obs/{slug}/{slot}/events"
    _sse_thread = threading.Thread(
        target=_sse_loop, args=(url, _stop_event), daemon=True, name="woogles-sse"
    )
    _sse_thread.start()


def _stop_sse() -> None:
    global _sse_thread
    _stop_event.set()
    if _sse_thread and _sse_thread.is_alive():
        _sse_thread.join(timeout=3)
    _sse_thread = None


# ---------------------------------------------------------------------------
# OBS timer callback: flush buffered SSE values into text sources
# Runs on the OBS main thread every 100 ms.
# ---------------------------------------------------------------------------

def _flush_to_sources() -> None:
    with _lock:
        snapshot = dict(_latest)

    for field, value in snapshot.items():
        source_name = _cfg.get(f"source_{field}", "").strip()
        if not source_name:
            continue
        source = obs.obs_get_source_by_name(source_name)
        if source is None:
            continue
        try:
            settings = obs.obs_data_create()
            obs.obs_data_set_string(settings, "text", value)
            obs.obs_source_update(source, settings)
            obs.obs_data_release(settings)
        finally:
            obs.obs_source_release(source)


# ---------------------------------------------------------------------------
# OBS script interface
# ---------------------------------------------------------------------------

def script_description() -> str:
    return (
        "<b>Woogles Text Source Updater</b><br><br>"
        "Connects to a Woogles broadcast slot and updates OBS Text sources "
        "directly — no browser source required for most fields.<br><br>"
        "For each field, enter the <i>exact</i> name of an existing OBS Text "
        "source.  Leave blank to skip that field.<br><br>"
        "<b>Tip:</b> for last_play scrolling, add OBS's built-in "
        "<i>Scroll</i> filter to the text source.<br>"
        "<b>Note:</b> blank1/blank2 colored letters need a browser source."
    )


def script_properties():
    props = obs.obs_properties_create()

    obs.obs_properties_add_text(props, "base_url", "Base URL",        obs.OBS_TEXT_DEFAULT)
    obs.obs_properties_add_text(props, "slug",     "Broadcast slug",  obs.OBS_TEXT_DEFAULT)
    obs.obs_properties_add_text(props, "slot",     "Slot name",       obs.OBS_TEXT_DEFAULT)

    obs.obs_properties_add_separator(props)

    for field in FIELDS:
        label = f"Source → {FIELD_LABELS[field]}"
        obs.obs_properties_add_text(props, f"source_{field}", label, obs.OBS_TEXT_DEFAULT)

    return props


def script_defaults(settings) -> None:
    obs.obs_data_set_default_string(settings, "base_url", "https://woogles.io")


def script_update(settings) -> None:
    global _cfg
    _cfg = {
        "base_url": obs.obs_data_get_string(settings, "base_url").strip(),
        "slug":     obs.obs_data_get_string(settings, "slug").strip(),
        "slot":     obs.obs_data_get_string(settings, "slot").strip(),
    }
    for field in FIELDS:
        _cfg[f"source_{field}"] = obs.obs_data_get_string(settings, f"source_{field}").strip()

    base_url = _cfg["base_url"]
    slug     = _cfg["slug"]
    slot     = _cfg["slot"]

    if base_url and slug and slot:
        _start_sse(base_url, slug, slot)
    else:
        _stop_sse()


def script_load(settings) -> None:
    obs.timer_add(_flush_to_sources, 100)   # 10 updates/sec is plenty for text
    script_update(settings)


def script_unload() -> None:
    obs.timer_remove(_flush_to_sources)
    _stop_sse()
