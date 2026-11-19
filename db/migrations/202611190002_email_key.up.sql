-- email_key is what two email addresses must share to reach the same inbox,
-- as far as we can tell:
--   - case doesn't matter;
--   - most providers deliver name+anything@ to name@;
--   - Gmail also ignores dots before the @, and googlemail.com is gmail.com.
-- Signing up with an address that has the same key as another account's,
-- or as a suspended account's current or past address, is refused.
CREATE OR REPLACE FUNCTION email_key(email text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$
    SELECT CASE
        WHEN substring(k FROM '@([^@]*)$') IN ('gmail.com', 'googlemail.com')
            THEN replace(regexp_replace(k, '@[^@]*$', ''), '.', '') || '@gmail.com'
        ELSE k
    END
    FROM (SELECT regexp_replace(lower(email), '\+[^@]*@', '@') AS k) AS tagless
    $$;
