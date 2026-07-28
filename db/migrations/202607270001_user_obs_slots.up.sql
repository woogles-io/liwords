-- Stream slots: user-owned, persistent OBS pointers.
--
-- A broadcast slot points at a table within one event; its OBS URL contains the
-- broadcast slug, so it changes every event. A stream slot points at whatever
-- the owner chooses, and its URL contains only the owner's username and the
-- slot name -- so it never changes. Switching what you are streaming is a
-- re-point here rather than an edit in OBS.
--
-- target_kind decides how the slot resolves:
--   'none'              -- nothing pointed yet; renders the placeholder
--   'broadcast_slot'    -- mirrors (target_broadcast_id, target_slot_name),
--                          so it follows whatever table that event features
--   'latest_annotation' -- follows the owner's most recently edited annotated
--                          game, for streaming without creating a broadcast
--
-- Keeping these as kinds of one slot rather than separate URL shapes means a
-- slot can move between them without the OBS browser source changing: stream
-- club nights off 'latest_annotation', then re-point the same slot at a
-- broadcast slot when you graduate to a real broadcast.
--
-- target_broadcast_id/target_slot_name are intentionally NOT a composite FK to
-- broadcast_slots: if an event's director renames or deletes the slot we point
-- at, resolution must degrade to the "(no game assigned)" placeholder rather
-- than fail. ON DELETE SET NULL handles the broadcast itself going away.
CREATE TABLE user_obs_slots (
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slot_name           TEXT   NOT NULL,
    target_kind         TEXT   NOT NULL DEFAULT 'none'
        CHECK (target_kind IN ('none', 'broadcast_slot', 'latest_annotation')),
    target_broadcast_id BIGINT REFERENCES broadcasts(id) ON DELETE SET NULL,
    target_slot_name    TEXT   NOT NULL DEFAULT '',
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, slot_name)
);

CREATE INDEX user_obs_slots_target_idx
    ON user_obs_slots (target_broadcast_id, target_slot_name);
