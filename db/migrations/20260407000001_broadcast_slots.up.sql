CREATE TABLE broadcast_slots (
    broadcast_id  BIGINT NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    slot_name     TEXT   NOT NULL,
    division      TEXT   NOT NULL,
    round         INT    NOT NULL DEFAULT 1,
    table_number  INT    NOT NULL DEFAULT 1,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (broadcast_id, slot_name)
);
