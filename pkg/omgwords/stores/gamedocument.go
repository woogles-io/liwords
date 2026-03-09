package stores

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var ErrDoesNotExist = errors.New("does not exist")

type GameDocumentStore struct {
	dbPool *pgxpool.Pool
	cfg    *config.Config
}

func NewGameDocumentStore(cfg *config.Config, db *pgxpool.Pool) (*GameDocumentStore, error) {
	return &GameDocumentStore{cfg: cfg, dbPool: db}, nil
}

// GetDocument gets a game document from PostgreSQL.
//
// CONCURRENCY WARNING: This does NOT prevent concurrent modifications!
// If multi-user editing is needed, implement one of:
//   1. Transaction-based locking: Wrap Get+Update in BEGIN...COMMIT with SELECT FOR UPDATE
//   2. Optimistic locking: Add version column, retry on conflict
//   3. Application-level locking: Serialize edits at a higher level
//
// Currently safe because annotated games are single-editor (only creator can edit).
func (gs *GameDocumentStore) GetDocument(ctx context.Context, uuid string) (*ipc.GameDocument, error) {
	return gs.getFromDatabase(ctx, uuid)
}

// DeleteDocument deletes a game document from PostgreSQL.
func (gs *GameDocumentStore) DeleteDocument(ctx context.Context, uuid string) error {
	_, err := gs.dbPool.Exec(ctx, `DELETE FROM game_documents WHERE game_id = $1`, uuid)
	return err
}

func (gs *GameDocumentStore) getFromDatabase(ctx context.Context, uuid string) (*ipc.GameDocument, error) {
	var bts []byte

	err := gs.dbPool.QueryRow(ctx, `SELECT document FROM game_documents WHERE game_id = $1`, uuid).
		Scan(&bts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrDoesNotExist
		}
		return nil, err
	}
	gdoc := &ipc.GameDocument{}
	uo := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	err = uo.Unmarshal(bts, gdoc)
	if err != nil {
		return nil, err
	}
	err = MigrateGameDocument(gs.cfg, gdoc)
	if err != nil {
		return nil, err
	}
	return gdoc, nil
}

// SetDocument writes the initial document to PostgreSQL.
func (gs *GameDocumentStore) SetDocument(ctx context.Context, gdoc *ipc.GameDocument) error {
	return gs.saveToDatabase(ctx, gdoc)
}

// UpdateDocument updates a document in PostgreSQL.
func (gs *GameDocumentStore) UpdateDocument(ctx context.Context, gdoc *ipc.GameDocument) error {
	return gs.saveToDatabase(ctx, gdoc)
}

func (gs *GameDocumentStore) saveToDatabase(ctx context.Context, gdoc *ipc.GameDocument) error {
	// save as protojson
	data, err := protojson.Marshal(gdoc)
	if err != nil {
		return err
	}
	tx, err := gs.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx,
		`INSERT INTO game_documents (game_id, document) VALUES ($1, $2)
		 ON CONFLICT (game_id) DO UPDATE
		 SET document = $2
		`,
		gdoc.Uid, data)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (gs *GameDocumentStore) DisconnectRDB() {
	gs.dbPool.Close()
}
