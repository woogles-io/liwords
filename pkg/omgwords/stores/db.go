package stores

import (
	"context"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type DBStore struct {
	dbPool *pgxpool.Pool
}

type BroadcastGame struct {
	GameUUID        string
	CreatorUUID     string
	CreatorUsername string
	Private         bool
	Finished        bool
	Players         []*ipc.PlayerInfo
	Lexicon         string
	Created         time.Time
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	return &DBStore{dbPool: p}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *DBStore) CreateAnnotatedGame(ctx context.Context, creatorUUID string, gameUUID string,
	private bool, quickdata *entity.Quickdata, req *ipc.GameRequest) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
	INSERT INTO annotated_game_metadata (game_uuid, creator_uuid, private_broadcast, done)
	VALUES ($1, $2, $3, $4)`, gameUUID, creatorUUID, private, false)

	if err != nil {
		return err
	}
	// Create a fake game request. XXX this is only to make it work with the rest
	// of the system. Otherwise, metadata API doesn't work. We will have to migrate
	// this.
	request, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	// Also insert it into the old games table. We will need to migrate this.
	_, err = tx.Exec(ctx, `
	INSERT INTO games (created_at, updated_at, uuid, type, quickdata, request)
	VALUES (NOW(), NOW(), $1, $2, $3, $4)
	`, gameUUID, ipc.GameType_ANNOTATED, quickdata, request)
	if err != nil {
		return err
	}
	// All other fields are blank. We will update the quickdata field with
	// necessary metadata
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) UpdateAnnotatedGameQuickdata(ctx context.Context, uuid string, quickdata *entity.Quickdata) error {
	_, err := s.dbPool.Exec(ctx, `
		UPDATE games SET quickdata = $1, updated_at = NOW() 
		WHERE uuid = $2`, quickdata, uuid)
	if err != nil {
		return err
	}
	return nil
}

func (s *DBStore) DeleteAnnotatedGame(ctx context.Context, uuid string) error {
	var gameID int
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT id FROM games WHERE uuid = $1`, uuid).Scan(&gameID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE from game_comments WHERE game_id = $1`, gameID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE from annotated_game_metadata WHERE game_uuid = $1`, uuid)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM games WHERE uuid = $1`, uuid)
	if err != nil {
		return err
	}
	// All other fields are blank. We will update the quickdata field with
	// necessary metadata
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) MarkAnnotatedGameDone(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE annotated_game_metadata SET done = TRUE 
					WHERE game_uuid = $1`, uuid)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE games SET updated_at = NOW() where uuid = $1`,
		uuid)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// OutstandingGames returns a list of game IDs for games that are not yet done being
// annotated. The system will only allow a certain number of games to remain
// undone for an annotator.
func (s *DBStore) OutstandingGames(ctx context.Context, creatorUUID string) ([]*BroadcastGame, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `SELECT game_uuid, private_broadcast FROM annotated_game_metadata 
	WHERE creator_uuid = $1 AND done = 'f'`

	rows, err := tx.Query(ctx, query, creatorUUID)
	if err == pgx.ErrNoRows {
		log.Debug().Str("creatorUUID", creatorUUID).Msg("outstanding games - no rows match")
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []*BroadcastGame{}
	for rows.Next() {
		var uuid string
		var private bool
		if err := rows.Scan(&uuid, &private); err != nil {
			return nil, err
		}
		games = append(games, &BroadcastGame{
			GameUUID:    uuid,
			CreatorUUID: creatorUUID,
			Private:     private,
			Finished:    false,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return games, nil
}

func (s *DBStore) GameOwnedBy(ctx context.Context, gid, uid string) (bool, error) {
	var ct int
	err := s.dbPool.QueryRow(ctx, `SELECT 1 FROM annotated_game_metadata
		WHERE creator_uuid = $1 AND game_uuid = $2 LIMIT 1`, uid, gid).Scan(&ct)
	if err != nil {
		return false, err
	}
	if ct == 1 {
		return true, nil
	}
	return false, nil
}

func (s *DBStore) GamesForEditor(ctx context.Context, editorID string, unfinished bool, limit, offset int) ([]*BroadcastGame, error) {

	var rows pgx.Rows
	var err error
	var query string
	if editorID == "" {
		query = `
		SELECT game_uuid, creator_uuid, username, 
			private_broadcast, quickdata, request, games.created_at 
		FROM annotated_game_metadata 
		JOIN games ON games.uuid = annotated_game_metadata.game_uuid
		JOIN users ON users.uuid = annotated_game_metadata.creator_uuid
		WHERE done = $1
		ORDER BY games.created_at DESC
		LIMIT $2 OFFSET $3
		`
		if rows, err = s.dbPool.Query(ctx, query, !unfinished, limit, offset); err != nil {
			if err == pgx.ErrNoRows {
				log.Debug().Msg("no games!")
				return nil, nil
			}
			return nil, err
		}
	} else {
		query = `
		SELECT game_uuid, creator_uuid, 'dummyusername', private_broadcast, quickdata, request, created_at FROM annotated_game_metadata 
		JOIN games ON games.uuid = annotated_game_metadata.game_uuid
		WHERE creator_uuid = $1 AND done = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
		if rows, err = s.dbPool.Query(ctx, query, editorID, !unfinished, limit, offset); err != nil {
			if err == pgx.ErrNoRows {
				log.Debug().Str("creatorUUID", editorID).Msg("no games for this editor")
				return nil, nil
			}
			return nil, err
		}
	}

	defer rows.Close()

	games := []*BroadcastGame{}
	for rows.Next() {
		var uuid string
		var creatorUUID string
		var creatorUsername string
		var private bool
		var quickdata *entity.Quickdata
		var request *entity.GameRequest
		var created time.Time

		if err := rows.Scan(&uuid, &creatorUUID, &creatorUsername, &private, &quickdata, &request, &created); err != nil {
			return nil, err
		}
		if quickdata == nil || request == nil {
			continue // although this shouldn't happen
		}
		games = append(games, &BroadcastGame{
			GameUUID:        uuid,
			CreatorUUID:     creatorUUID,
			CreatorUsername: creatorUsername,
			Private:         private,
			Finished:        !unfinished,
			Players:         quickdata.PlayerInfo,
			Lexicon:         request.Lexicon,
			Created:         created,
		})
	}
	return games, nil
}

func (s *DBStore) GameIsDone(ctx context.Context, gid string) (bool, error) {
	var done bool
	err := s.dbPool.QueryRow(ctx, `SELECT done FROM annotated_game_metadata
		WHERE game_uuid = $1`, gid).Scan(&done)
	if err != nil {
		return false, err
	}
	return done, nil
}
