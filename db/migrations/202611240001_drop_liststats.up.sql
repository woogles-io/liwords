-- liststats has not been written since #2380, and nothing ever read it.
-- Its rows (77M, ~12 GB) were dumped to S3 before this ran.
--
-- No lock_timeout on purpose. Once #2380 is deployed, the only lock on
-- liststats this can wait for is the daily pg_dump's, and waiting there
-- blocks no one else, since nothing else touches the table. A timeout would
-- instead fail the migration and leave it dirty, which blocks every later
-- deploy until it is forced by hand. Run it outside the pg_dump window so it
-- doesn't wait at all.
BEGIN;
DROP TABLE IF EXISTS liststats;
COMMIT;
