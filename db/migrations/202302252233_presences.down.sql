BEGIN;

DROP TABLE public.user_channel_presences;
DROP INDEX idx_uc_presences_combined;
DROP INDEX idx_uc_presences_channel;
DROP INDEX idx_uc_presences_connid;
DROP INDEX idx_uc_presences_last_seen_at;

COMMIT;