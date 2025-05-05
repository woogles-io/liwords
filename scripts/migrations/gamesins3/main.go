package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

func testGDocMigration(cfg *config.Config, pool *pgxpool.Pool) error {
	ctx := context.Background()
	// select all games of type 0 (normal, not annotated), ignore aborted games.
	// Those we should likely eventually delete.
	query := `SELECT id, created_at, uuid FROM games WHERE id > 7000000 and type = 0
		and game_end_reason <> 5  LIMIT 100000`

	qconn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer qconn.Release()

	rows, err := qconn.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	ustore, err := user.NewDBStore(pool)
	if err != nil {
		return err
	}
	gameStore, err := game.NewDBStore(cfg, ustore, pool)
	if err != nil {
		return err
	}

	ct := 0

	for rows.Next() {
		ct++
		var id int64
		var uuid string
		var createdAt pgtype.Timestamptz
		// var createdAt time.Time

		if err := rows.Scan(&id, &createdAt, &uuid); err != nil {
			return err
		}
		g, err := gameStore.Get(ctx, uuid)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting game %s", uuid)
			return err
		}
		gh1 := g.History()
		for _, evt := range gh1.Events {
			// we never migrated this and it is useless except in the front-end
			// for exchange situations with unknown tiles. So let's leave it
			// out of this test.
			evt.NumTilesFromRack = 0
		}
		// convert to gamedocument
		gdoc, err := utilities.ToGameDocument(g, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error converting game %s to gdoc", uuid)
			return err
		}
		// convert back to game history
		gh2, err := utilities.ToGameHistory(gdoc, cfg)
		if err != nil {
			log.Error().Err(err).Msgf("Error converting gdoc %s to game history", uuid)
			return err
		}
		for _, evt := range gh2.Events {
			// see comment above for gh1
			evt.NumTilesFromRack = 0
		}

		if !proto.Equal(gh1, gh2) {
			diff := cmp.Diff(gh1, gh2,
				protocmp.Transform(),  // Handles protobuf messages properly
				cmpopts.EquateEmpty(), // Treats nil slices/maps same as empty ones
			)
			log.Error().
				Str("game_uuid", uuid).
				Str("diff", diff).
				Msg("Game history does not match after conversion")
			fmt.Println("Original game history:")
			j, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(gh1)
			if err != nil {
				log.Error().Err(err).Msgf("Error marshalling game history %s", uuid)
				return err
			}
			fmt.Println(string(j))
			fmt.Println("Converted game history:")
			j2, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(gh2)
			if err != nil {
				log.Error().Err(err).Msgf("Error marshalling converted game history %s", uuid)
				return err
			}
			fmt.Println(string(j2))
			fmt.Println("Created at:", createdAt.Time)

			return fmt.Errorf("game %s history does not match after conversion", uuid)
		}
	}
	log.Info().Int("ct", ct).Msg("migration was successfully tested")
	return nil

}

func main() {
	cfg := &config.Config{}
	cfg.Load(nil)
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	dbCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	dbPool, err := pgxpool.NewWithConfig(ctx, dbCfg)
	if err != nil {
		panic(err)
	}
	if len(os.Args) < 2 {
		panic("need 1 argument: mode (testgdoc)")
	}

	mode := os.Args[1]
	if mode == "testgdoc" {
		err = testGDocMigration(cfg, dbPool)
		if err != nil {
			log.Error().Err(err).Msg("Error in testGDocMigration")
			return
		}
	} else {
		panic("mode not supported yet")
	}

}
