-- Drop the old request column (bytea/protobuf) now that we use game_request (jsonb)
ALTER TABLE games DROP COLUMN IF EXISTS request;