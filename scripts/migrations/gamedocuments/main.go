package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

// there's a small number of game documents for now. Update them all in a
// single transaction.
func migrate(cfg *config.Config, pool *pgxpool.Pool) error {
	ctx := context.Background()
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Note: rewrite this if we get more than a few hundred rows and we need to
	// run this again! Do it in batches. This locks the whole table!
	query := "SELECT document FROM game_documents WHERE document->>'version' = '1' FOR UPDATE"
	updateQuery := "UPDATE game_documents SET document = $1 WHERE game_id = $2"

	updateStmt, err := tx.Prepare(ctx, "update", updateQuery)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	uo := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	ct := 0
	docs := []*ipc.GameDocument{}
	for rows.Next() {
		gdoc := &ipc.GameDocument{}
		var bts []byte
		if err := rows.Scan(&bts); err != nil {
			return err
		}
		err = uo.Unmarshal(bts, gdoc)
		if err != nil {
			return err
		}
		err = stores.MigrateGameDocument(cfg, gdoc)
		if err != nil {
			return err
		}

		docs = append(docs, gdoc)
	}

	for idx := range docs {
		remarshalled, err := protojson.Marshal(docs[idx])
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, updateStmt.Name, remarshalled, docs[idx].Uid)

		if err != nil {
			return err
		}
		ct++
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	log.Info().Int("count", ct).Msg("updated-documents")
	return nil
}

func main() {
	// Migrate all game documents to next version
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	err = migrate(cfg, pool)
	if err != nil {
		panic(err)
	}
}
