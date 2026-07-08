select
	count(distinct user_id) as users_with_badges_count,
	count(distinct case when badge_id in (11,12,13) then user_id else null end) as patreon_user_count
from public.user_badges
