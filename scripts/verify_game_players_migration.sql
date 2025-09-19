-- Verification queries for game_players migration

-- 1. Check HastyBot's distribution in game_players table
-- Should be roughly 50/50 if the migration is correct
SELECT
    CASE WHEN player_index = 0 THEN 'first_player' ELSE 'second_player' END as position,
    COUNT(*) as count
FROM game_players gp
JOIN users u ON gp.player_id = u.id
WHERE u.id = 230 -- HastyBot
GROUP BY player_index
ORDER BY player_index;

-- 2. Compare a sample of games between old and new approach
-- Check if player ordering matches between quickdata and game_players
SELECT
    g.uuid,
    g.quickdata->'PlayerInfo'->0->>'Nickname' as quickdata_first_player,
    g.quickdata->'PlayerInfo'->1->>'Nickname' as quickdata_second_player,
    u0.username as game_players_first_player,
    u1.username as game_players_second_player
FROM games g
LEFT JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
LEFT JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
LEFT JOIN users u0 ON gp0.player_id = u0.id
LEFT JOIN users u1 ON gp1.player_id = u1.id
WHERE g.game_end_reason NOT IN (0, 7)
    AND g.quickdata IS NOT NULL
    AND gp0.player_id IS NOT NULL
    AND gp1.player_id IS NOT NULL
ORDER BY g.created_at DESC
LIMIT 10;

-- 3. Count total migrated rows
SELECT COUNT(*) as total_game_players_rows FROM game_players;

-- 4. Check for any mismatches (should return 0 if migration is perfect)
SELECT COUNT(*) as mismatches
FROM games g
JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
JOIN users u0 ON gp0.player_id = u0.id
JOIN users u1 ON gp1.player_id = u1.id
WHERE g.game_end_reason NOT IN (0, 7)
    AND g.quickdata IS NOT NULL
    AND (
        g.quickdata->'PlayerInfo'->0->>'Nickname' != u0.username
        OR g.quickdata->'PlayerInfo'->1->>'Nickname' != u1.username
    );

-- 5. Check score accuracy (sample)
SELECT
    g.uuid,
    g.quickdata->>'finalScores' as quickdata_scores,
    gp0.score as first_player_score,
    gp1.score as second_player_score
FROM games g
JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
WHERE g.quickdata->>'finalScores' IS NOT NULL
ORDER BY g.created_at DESC
LIMIT 5;