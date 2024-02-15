BEGIN;

-- Not strictly a backwards migration, but what can you do.
UPDATE
    tournaments
SET
    extra_meta = jsonb_set(
        extra_meta,
        '{defaultClubSettings,lexicon}',
        '"NWL20"',
        true
    )
WHERE
    type = 'club'
    AND extra_meta -> 'defaultClubSettings' ->> 'lexicon' = 'NWL23';

COMMIT;