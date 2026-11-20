---
name: league-late-join
description: Add a player to a league season after registration closed (late join) by writing the prod SQL — confirms the user and season, checks the season hasn't started, picks the division the placement code would have chosen, grants the League Player role, and inserts the registration. Use when the user asks to get, add, put or squeeze someone into a league season (CSW, NWL, Deutsch…) or says a player missed registration.
---

# Late-joining a league player

A late join is only possible while the season is **SCHEDULED** (status 0):
divisions are built but games don't exist yet. At `start_date` the hourly
runner reads each division's registrations fresh and generates the round robin,
so a row inserted before then gets paired like everyone else.

There is no RPC for this. `RegisterForSeason` only works during
REGISTRATION_OPEN and is self-service. `MovePlayerToDivision` needs the player
already in a division. Both require SCHEDULED or earlier. So it's raw SQL.

Run read-only checks with `bin/proddb -c "..."`. If running prod queries gets
denied, hand them to the user. **Always give writes to the user to run; never
run them yourself.**

## 1. Pin down the player and the season

Users often paste the wrong UUID. On 2026-11-19 the "user UUID" given was
really the season UUID. Match a pasted UUID against both `users.uuid` (a 22-char
shortuuid) and `league_seasons.uuid` (a standard UUID).

```sql
SELECT u.id, u.uuid, u.username, u.internal_bot, p.first_name, p.last_name, p.country_code
FROM users u LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.uuid = '<given>' OR u.username ILIKE '%<name>%'
   OR p.first_name ILIKE '%<name>%' OR p.last_name ILIKE '%<name>%';

-- status: 0=SCHEDULED 1=ACTIVE 2=COMPLETED 3=CANCELLED 4=REGISTRATION_OPEN
SELECT now(), l.slug, l.settings->>'lexicon' AS lex, s.uuid AS season_uuid,
       s.season_number, s.status, s.start_date, s.divisions_prepared_at, s.started_at
FROM leagues l JOIN league_seasons s ON s.league_id = l.uuid
WHERE l.is_active
  AND s.season_number >= (SELECT max(season_number) - 1 FROM league_seasons WHERE league_id = l.uuid)
ORDER BY l.slug, s.season_number DESC;
```

- Identify the league by `settings->>'lexicon'`, not the slug.
- If more than one user matches the name, stop and ask which one.
- If `internal_bot` is true, **stop**: bots can't play in leagues. Season start
  relies on that, and sends a bot no move request (`RegisterPlayer` refuses
  bots for the same reason).
- `current_season_id` is NULL between the old season closing and the new one
  starting. That's normal, not a sign anything is broken.

Then branch on the season's status:

| status | what to do |
|---|---|
| 4 REGISTRATION_OPEN | Nothing special. The player registers on the site, or you insert with `division_id` NULL and placement assigns a division. Still grant the role (step 3). |
| 0 SCHEDULED, `now() < start_date` | Continue below. Tell the user how long they have before `start_date`. |
| 1 ACTIVE, or `started_at` set | **Stop.** Pairings already exist. Adding a player means regenerating the division's games, which this skill doesn't cover. |

## 2. Pick the division

Get the player's history in this league and any existing registration:

```sql
SELECT s.season_number, s.status AS season_status, d.division_number,
       r.placement_status, r.division_id
FROM league_registrations r
JOIN league_seasons s ON s.uuid = r.season_id
LEFT JOIN league_divisions d ON d.uuid = r.division_id
WHERE r.user_id = <user_id> AND s.league_id = (SELECT uuid FROM leagues WHERE slug = '<slug>')
ORDER BY s.season_number;
```

If they already have a row for this season, it's a fix-up, not a late join. Look
before writing anything.

Then look at the divisions and where this season's comparable players went:

```sql
SELECT d.division_number, d.uuid, count(r.*) AS n,
       count(*) FILTER (WHERE r.placement_status = 1) AS new,
       count(*) FILTER (WHERE r.placement_status IN (6,7)) AS returning
FROM league_divisions d LEFT JOIN league_registrations r ON r.division_id = d.uuid
WHERE d.season_id = '<season_uuid>' GROUP BY 1,2 ORDER BY 1;
```

Placement statuses (`ipc.PlacementStatus`): 1 NEW, 3 PROMOTED, 4 RELEGATED,
5 STAYED, 6 SHORT_HIATUS_RETURNING (away 1–3 seasons), 7 LONG_HIATUS_RETURNING
(away 4+). Mirror what `UpdatePlacementStatuses` in `pkg/league/rebalance.go`
would have done:

- **Never played this league:** `placement_status` 1, `seasons_away` 0,
  `previous_division_rank` NULL. Rookies end up in the bottom division, so use the
  division where this season's NEW players went (normally the highest number).
- **Played before, skipped the last season:**
  `seasons_away = new_season_number - last_played_season_number - 1`, status 6
  or 7 to match. Copy `previous_division_rank` from their last registration. Put
  them in the division with the same number they last played in. If that
  division no longer exists, use the bottom one, as `placement.go` does.
- **Played last season** (a dropped registration): status 3/4/5 comes from last
  season's standings. Ask the user before placing them, since it's a judgment
  call.

Mention the resulting division size. Divisions are usually 13–17 players. Ask
before pushing a division well past the others.

Check whether they have the League Player role (`can_play_leagues`); without it
they can't play:

```sql
SELECT ro.id, ro.name FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ur.user_id = <user_id>;
SELECT id FROM roles WHERE name = 'League Player';   -- 7 in prod as of 2026-11
```

Copy the row shape from an existing registration in the same division and same
placement status (`SELECT * FROM league_registrations WHERE division_id = ... AND placement_status = ... LIMIT 1`).
Don't guess the column values.

## 3. The write (the user runs it)

```sql
BEGIN;

INSERT INTO user_roles (user_id, role_id) VALUES (<user_id>, <league_player_role_id>)
ON CONFLICT DO NOTHING;

INSERT INTO league_registrations
  (user_id, season_id, division_id, registration_date, firsts_count, status,
   placement_status, previous_division_rank, seasons_away)
VALUES
  (<user_id>, '<season_uuid>', '<division_uuid>', now(), 0, 'REGISTERED',
   <placement_status>, <prev_rank_or_NULL>, <seasons_away>);

-- sanity check: expected count / new count for the division
SELECT count(*), count(*) FILTER (WHERE placement_status = <placement_status>)
FROM league_registrations WHERE division_id = '<division_uuid>';

COMMIT;
```

Don't create `league_standings` rows. They don't exist before the season starts;
game start creates them.

## Tell the user

- The resolved identities: username, real name, user id, league/lexicon, season number.
- The deadline (`start_date`). If the write runs after the season starts, **don't
  run it**: the division's pairings are already set.
- The player gets no registration confirmation email, because that's only sent
  by the RPC.
- After `start_date`, confirm the player got games:
  `SELECT count(*) FROM games WHERE (player0_id = <id> OR player1_id = <id>) AND league_division_id = '<division_uuid>';`
