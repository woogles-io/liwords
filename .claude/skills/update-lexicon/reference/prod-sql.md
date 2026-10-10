# Production SQL for a lexicon update

Run these only with the user's approval, against prod, inside a transaction. Look at the rows with a SELECT before running any UPDATE.

## Clubs whose default lexicon is an old edition

```sql
-- look first
SELECT id, slug, extra_meta->'defaultClubSettings'->>'lexicon' AS lex
FROM tournaments
WHERE type = 'club'
  AND extra_meta->'defaultClubSettings'->>'lexicon' IN ('NSF21','NSF22','NSF23','NSF25');

BEGIN;
UPDATE tournaments
SET extra_meta = jsonb_set(extra_meta, '{defaultClubSettings,lexicon}', '"NSF26"', false)
WHERE type = 'club'
  AND extra_meta->'defaultClubSettings'->>'lexicon' IN ('NSF21','NSF22','NSF23','NSF25');
COMMIT;
```

Use an explicit `IN (...)` list of the family's old editions. `LIKE 'CSW%'` would also move `CSW24X` clubs.

As a migration instead: `./gen_migration.sh <name>` from the repo root (or hand-create `YYYYMMDDHHmm_<name>.{up,down}.sql`, UTC). The up file holds the UPDATE above. The down file can only map NEW back to OLD, which is lossy (see `202402150312_default_club_settings_nwl23.down.sql`).

## Report only; don't change without asking

```sql
-- leagues whose next season would start on the old edition
SELECT id, slug, settings->>'lexicon' AS lex FROM leagues
WHERE settings->>'lexicon' IN ('NSF25');

-- unfinished non-club tournaments on the old edition
SELECT id, slug, type FROM tournaments
WHERE type <> 'club' AND NOT is_finished
  AND extra_meta::text LIKE '%"NSF25"%';
```

Check the actual column names (`\d leagues`, `\d tournaments`) before running these. They were written from the migrations and may have drifted.
