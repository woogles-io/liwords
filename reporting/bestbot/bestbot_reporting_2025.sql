WITH duplicated_games AS
((SELECT
    created_at,
    player0_id AS player
FROM public.games
WHERE created_at>'2025-01-01')
UNION ALL
(SELECT
    created_at,
    player1_id AS player
FROM public.games
WHERE created_at>'2025-01-01'))


select
	u.username,
	date_trunc('month',g.created_at),
	count(*)
from duplicated_games g
left join public.users u ON g.player = u.id
where (u.internal_bot OR u.id IN (42,43,44,45,46))
group by 1,2
