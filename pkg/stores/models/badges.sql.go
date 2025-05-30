// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: badges.sql

package models

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addBadge = `-- name: AddBadge :exec
INSERT INTO badges (code, description)
VALUES ($1, $2)
`

type AddBadgeParams struct {
	Code        string
	Description string
}

func (q *Queries) AddBadge(ctx context.Context, arg AddBadgeParams) error {
	_, err := q.db.Exec(ctx, addBadge, arg.Code, arg.Description)
	return err
}

const addUserBadge = `-- name: AddUserBadge :exec
INSERT INTO user_badges (user_id, badge_id)
VALUES ((SELECT id FROM users where lower(username) = lower($1)), (SELECT id from badges where code = $2))
`

type AddUserBadgeParams struct {
	Username string
	Code     string
}

func (q *Queries) AddUserBadge(ctx context.Context, arg AddUserBadgeParams) error {
	_, err := q.db.Exec(ctx, addUserBadge, arg.Username, arg.Code)
	return err
}

const bulkRemoveBadges = `-- name: BulkRemoveBadges :exec
DELETE FROM user_badges
WHERE badge_id IN (
  SELECT id
  FROM badges
  WHERE code = ANY($1::text[])
)
`

func (q *Queries) BulkRemoveBadges(ctx context.Context, badgeCodes []string) error {
	_, err := q.db.Exec(ctx, bulkRemoveBadges, badgeCodes)
	return err
}

const getBadgeDescription = `-- name: GetBadgeDescription :one
SELECT description FROM badges
WHERE code = $1
`

func (q *Queries) GetBadgeDescription(ctx context.Context, code string) (string, error) {
	row := q.db.QueryRow(ctx, getBadgeDescription, code)
	var description string
	err := row.Scan(&description)
	return description, err
}

const getBadgesForUser = `-- name: GetBadgesForUser :many
SELECT badges.code FROM user_badges
JOIN badges on badges.id = user_badges.badge_id
WHERE user_badges.user_id = (SELECT id from users where uuid = $1)
`

func (q *Queries) GetBadgesForUser(ctx context.Context, uuid pgtype.Text) ([]string, error) {
	rows, err := q.db.Query(ctx, getBadgesForUser, uuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		items = append(items, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getBadgesMetadata = `-- name: GetBadgesMetadata :many
SELECT code, description FROM badges
`

type GetBadgesMetadataRow struct {
	Code        string
	Description string
}

func (q *Queries) GetBadgesMetadata(ctx context.Context) ([]GetBadgesMetadataRow, error) {
	rows, err := q.db.Query(ctx, getBadgesMetadata)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetBadgesMetadataRow
	for rows.Next() {
		var i GetBadgesMetadataRow
		if err := rows.Scan(&i.Code, &i.Description); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUsersForBadge = `-- name: GetUsersForBadge :many
SELECT users.username FROM user_badges
JOIN users on users.id = user_badges.user_id
WHERE user_badges.badge_id = (SELECT id from badges where code = $1)
ORDER BY users.username
`

func (q *Queries) GetUsersForBadge(ctx context.Context, code string) ([]pgtype.Text, error) {
	rows, err := q.db.Query(ctx, getUsersForBadge, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.Text
	for rows.Next() {
		var username pgtype.Text
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		items = append(items, username)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeUserBadge = `-- name: RemoveUserBadge :exec
DELETE FROM user_badges
WHERE user_id = (SELECT id from users where lower(username) = lower($1))
AND badge_id = (SELECT id from badges where code = $2)
`

type RemoveUserBadgeParams struct {
	Username string
	Code     string
}

func (q *Queries) RemoveUserBadge(ctx context.Context, arg RemoveUserBadgeParams) error {
	_, err := q.db.Exec(ctx, removeUserBadge, arg.Username, arg.Code)
	return err
}

const upsertPatreonBadges = `-- name: UpsertPatreonBadges :execrows
WITH patreon_map AS (
  SELECT
    elem->>'patreon_user_id' AS patreon_user_id,
    elem->>'badge_code' AS badge_code
  FROM jsonb_array_elements($1::jsonb) AS elem
)
INSERT INTO user_badges (user_id, badge_id)
SELECT i.user_id, b.id  -- Fetch badge_id using badge_code
FROM integrations i
JOIN patreon_map m
  ON i.data->>'patreon_user_id' = m.patreon_user_id
JOIN badges b
  ON b.code = m.badge_code
`

func (q *Queries) UpsertPatreonBadges(ctx context.Context, dollar_1 []byte) (int64, error) {
	result, err := q.db.Exec(ctx, upsertPatreonBadges, dollar_1)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
