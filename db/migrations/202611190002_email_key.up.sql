-- email_key is what two email addresses must share to reach the same inbox,
-- as far as we can tell: case doesn't matter, and most providers deliver
-- name+anything@ to name@. Signing up with name+x@ when name@ has an
-- account, or reusing a suspended account's address that way, is refused
-- by comparing keys.
CREATE OR REPLACE FUNCTION email_key(email text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$ SELECT regexp_replace(lower(email), '\+[^@]*@', '@') $$;
