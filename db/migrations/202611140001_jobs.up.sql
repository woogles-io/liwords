-- Background work, run once across all API servers (pkg/jobs).
--
-- A row is either a schedule (schedule set, run again after each run) or a
-- one-off job (schedule null, done_at set when it finishes). A runner claims
-- a due row by setting lease_until and lease_token; another runner can claim
-- it only once the lease has expired.
CREATE TABLE jobs (
    id          BIGSERIAL PRIMARY KEY,
    kind        TEXT NOT NULL,
    key         TEXT NOT NULL DEFAULT '',
    schedule    TEXT,
    run_at      TIMESTAMPTZ NOT NULL,
    lease_until TIMESTAMPTZ,
    lease_token UUID,
    done_at     TIMESTAMPTZ,
    attempts    INTEGER NOT NULL DEFAULT 0,
    last_run_at TIMESTAMPTZ,
    last_ms     INTEGER,
    last_error  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kind, key)
);

-- The runner's claim and its "when is the next one due" both read only the
-- rows not done.
CREATE INDEX jobs_due_idx ON jobs (run_at) WHERE done_at IS NULL;
