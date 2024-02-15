BEGIN;

UPDATE
    tournaments
SET
    extra_meta = jsonb_set(
        extra_meta,
        '{defaultClubSettings,lexicon}',
        '"NWL23"',
        true
    )
WHERE
    type = 'club'
    AND extra_meta -> 'defaultClubSettings' ->> 'lexicon' LIKE 'NWL%';

COMMIT;