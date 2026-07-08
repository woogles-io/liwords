-- What timezone does the server report hours in, and when do the big tables
-- get autovacuumed/analyzed (a background-load contributor)?
SELECT current_setting('TimeZone') AS server_tz, now() AS server_now;

SELECT relname,
       n_live_tup,
       last_autovacuum,
       last_autoanalyze
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC
LIMIT 8;
