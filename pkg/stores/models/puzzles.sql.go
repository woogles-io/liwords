// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: puzzles.sql

package models

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/woogles-io/liwords/pkg/entity"
)

const getPotentialPuzzleGames = `-- name: GetPotentialPuzzleGames :many
SELECT games.uuid FROM games
LEFT JOIN puzzles on puzzles.game_id = games.id
WHERE puzzles.id IS NULL 
    AND games.created_at BETWEEN $1 AND $2
    AND (stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND games.request LIKE $3  -- %lexicon%
    AND games.request NOT LIKE '%classic_super%'
    AND games.request NOT LIKE '%wordsmog%'
    -- 0: none, 5: aborted, 7: canceled
    AND game_end_reason not in (0, 5, 7)
    AND type = 0
    
    ORDER BY games.id DESC 
    LIMIT $4 OFFSET $5
`

type GetPotentialPuzzleGamesParams struct {
	CreatedAt   pgtype.Timestamptz
	CreatedAt_2 pgtype.Timestamptz
	Request     entity.GameRequest
	Limit       int32
	Offset      int32
}

func (q *Queries) GetPotentialPuzzleGames(ctx context.Context, arg GetPotentialPuzzleGamesParams) ([]pgtype.Text, error) {
	rows, err := q.db.Query(ctx, getPotentialPuzzleGames,
		arg.CreatedAt,
		arg.CreatedAt_2,
		arg.Request,
		arg.Limit,
		arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.Text
	for rows.Next() {
		var uuid pgtype.Text
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		items = append(items, uuid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPotentialPuzzleGamesAvoidBots = `-- name: GetPotentialPuzzleGamesAvoidBots :many

SELECT games.uuid FROM games
LEFT JOIN puzzles on puzzles.game_id = games.id
WHERE puzzles.id IS NULL 
    AND games.created_at BETWEEN $1 AND $2
    AND (stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND games.request LIKE $3  -- %lexicon%
    AND games.request NOT LIKE '%classic_super%'
    AND games.request NOT LIKE '%wordsmog%'
    -- 0: none, 5: aborted, 7: canceled
    AND game_end_reason not in (0, 5, 7)
    AND NOT (quickdata @> '{"pi": [{"is_bot": true}]}'::jsonb)
    AND type = 0

    ORDER BY games.id DESC 
    LIMIT $4 OFFSET $5
`

type GetPotentialPuzzleGamesAvoidBotsParams struct {
	CreatedAt   pgtype.Timestamptz
	CreatedAt_2 pgtype.Timestamptz
	Request     entity.GameRequest
	Limit       int32
	Offset      int32
}

// puzzle generation
func (q *Queries) GetPotentialPuzzleGamesAvoidBots(ctx context.Context, arg GetPotentialPuzzleGamesAvoidBotsParams) ([]pgtype.Text, error) {
	rows, err := q.db.Query(ctx, getPotentialPuzzleGamesAvoidBots,
		arg.CreatedAt,
		arg.CreatedAt_2,
		arg.Request,
		arg.Limit,
		arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pgtype.Text
	for rows.Next() {
		var uuid pgtype.Text
		if err := rows.Scan(&uuid); err != nil {
			return nil, err
		}
		items = append(items, uuid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
