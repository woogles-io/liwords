r"""
nb_helper.py — direct-connect query helper for Woogles reporting notebooks.

Purpose
-------
A reusable, AI-native alternative to the CSV round-trip for reviewing query
output in Jupyter/VSCode. Reads connection params from ``reporting/.env`` and the
password from ``~/.pgpass`` (never hardcoded), returns results as a pandas
DataFrame so you can sort/filter/plot immediately.

Usage (in any scratch notebook under reporting/)
------------------------------------------------
    from nb_helper import run_sql, run_file
    df = run_sql("select count(*) from games where created_at > '2025-01-01'")
    df = run_file("omgwords/games_per_month.sql")   # path relative to reporting/

Safety
------
- This is the live PRODUCTION reporting DB — ANALYTICS/REPORTING ONLY. This helper
  is hard-wired read-only: every connection sets ``default_transaction_read_only=on``
  server-side, so INSERT/UPDATE/DELETE/DDL are rejected by Postgres. There is no
  write path here by design. Never add one.
- Requires the "Woogles" WireGuard tunnel to be up first:  scutil --nc start Woogles
- Enforces a per-statement timeout (default 60s) so a runaway query cannot leave
  an orphaned heavy backend on the ~12M-row games table. Override per call or via
  the WOOGLES_STATEMENT_TIMEOUT_MS env var. See the woogles-queries skill for why
  orphaned backends are the #1 "query suddenly got slow" cause.

For psql-specific files (\set, multiple statements, meta-commands) keep using
run_query.sh — this helper runs a single SQL statement via SQLAlchemy/psycopg.

Deps:  pandas  sqlalchemy  "psycopg[binary]"   (psycopg v3)
"""
from __future__ import annotations

import os
from pathlib import Path

import pandas as pd
from sqlalchemy import create_engine, text

_REPORTING_DIR = Path(__file__).resolve().parent.parent
_DEFAULT_TIMEOUT_MS = int(os.environ.get("WOOGLES_STATEMENT_TIMEOUT_MS", "60000"))


def _load_env() -> dict[str, str]:
    env: dict[str, str] = {}
    envfile = _REPORTING_DIR / ".env"
    for line in envfile.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, val = line.split("=", 1)
        env[key.strip()] = val.strip().strip('"').strip("'")
    return env


def _engine(timeout_ms: int | None = None):
    e = _load_env()
    host = e["REPORTING_DB_HOST"]
    port = e.get("REPORTING_DB_PORT", "5432")
    db = e["REPORTING_DB_NAME"]
    user = e["REPORTING_DB_USER"]
    # Password intentionally omitted from the URI -> libpq/psycopg reads ~/.pgpass.
    uri = f"postgresql+psycopg://{user}@{host}:{port}/{db}"

    # Hard-wired read-only: production analytics/reporting only, no write path.
    opts = [
        f"-c statement_timeout={timeout_ms or _DEFAULT_TIMEOUT_MS}",
        "-c default_transaction_read_only=on",
    ]
    connect_args = {"options": " ".join(opts)}
    return create_engine(uri, connect_args=connect_args, pool_pre_ping=True)


def run_sql(sql: str, params: dict | None = None,
            timeout_ms: int | None = None) -> pd.DataFrame:
    """Run one read-only SQL statement and return a DataFrame.

    The connection is read-only at the server level, so only SELECT / EXPLAIN /
    read-only statements will succeed; a write raises a Postgres error by design.
    ``timeout_ms`` overrides the default statement timeout for a known-heavy query.
    """
    with _engine(timeout_ms=timeout_ms).connect() as conn:
        return pd.read_sql(text(sql), conn, params=params)


def run_file(relpath: str, timeout_ms: int | None = None) -> pd.DataFrame:
    """Run a ``.sql`` file (path relative to reporting/) and return a DataFrame.

    Plain single-statement SELECT files only. For psql meta-commands / multi-
    statement files, use run_query.sh instead.
    """
    sql = (_REPORTING_DIR / relpath).read_text()
    return run_sql(sql, timeout_ms=timeout_ms)
