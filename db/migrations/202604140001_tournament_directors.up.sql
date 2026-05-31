BEGIN;

CREATE TYPE tournament_director_role AS ENUM ('director', 'readonly_director');

CREATE TABLE tournament_directors (
  tournament_id integer NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
  user_id       integer NOT NULL REFERENCES users(id)       ON DELETE CASCADE,
  role          tournament_director_role NOT NULL DEFAULT 'director',
  created_at    timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tournament_id, user_id)
);

CREATE INDEX idx_tournament_directors_user_id ON tournament_directors(user_id);

-- Backfill from tournaments.directors JSON.
-- Persons shape: { "persons": [ { "id": "uuid:username", "rating": int } ] }
INSERT INTO tournament_directors (tournament_id, user_id, role)
SELECT
  t.id,
  u.id,
  CASE
    WHEN (p->>'rating')::int = -1 THEN 'readonly_director'::tournament_director_role
    ELSE 'director'::tournament_director_role
  END
FROM tournaments t
CROSS JOIN LATERAL jsonb_array_elements(t.directors::jsonb -> 'persons') AS p
JOIN users u ON u.uuid = split_part(p->>'id', ':', 1)
ON CONFLICT DO NOTHING;

COMMIT;
