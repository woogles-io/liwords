-- name: SetProfileAvatarUrl :exec
UPDATE profiles SET avatar_url = @avatar_url, updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);

-- name: SetProfilePersonalInfo :exec
UPDATE profiles
   SET first_name = @first_name, last_name = @last_name,
       birth_date = @birth_date, country_code = @country_code,
       about = @about, updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);

-- name: ResetProfileStats :exec
UPDATE profiles SET stats = @stats, updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);

-- name: ResetProfileRatings :exec
UPDATE profiles SET ratings = @ratings, updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);

-- name: ResetProfileStatsAndRatings :exec
UPDATE profiles SET stats = @stats, ratings = @ratings, updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);

-- name: ResetProfilePersonalInfo :exec
UPDATE profiles
   SET first_name = '', last_name = '', about = '', title = '',
       avatar_url = '', country_code = '', updated_at = NOW()
 WHERE user_id = (SELECT id FROM users WHERE uuid = @uuid);
