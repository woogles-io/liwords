package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/user"

	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

func createActionKey(action *ms.ModAction) string {
	return fmt.Sprintf("%s:%d:%s", action.UserId, action.StartTime.Seconds, action.Type.String())
}

func getAllActions(ctx context.Context, dbPool *pgxpool.Pool) (map[string]bool, error) {
	tx, err := dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT users.uuid, user_actions.start_time, user_actions.action_type FROM user_actions JOIN users ON
	user_actions.user_id = users.id`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	allActions := map[string]bool{}
	for rows.Next() {
		var user_uuid string
		var action_type int
		var start_time pgtype.Timestamptz

		if err := rows.Scan(&user_uuid, &start_time, &action_type); err != nil {
			return nil, err
		}

		var startTime *timestamppb.Timestamp = nil
		if start_time.Valid {
			startTime = timestamppb.New(start_time.Time)
		}

		modAction := &ms.ModAction{
			UserId:    user_uuid,
			Type:      ms.ModActionType(action_type),
			StartTime: startTime}

		_, exists := allActions[createActionKey(modAction)]
		if exists {
			log.Info().Msgf("duplicate actions in database: %s, %d, %s", modAction.UserId, modAction.StartTime.Seconds, modAction.Type.String())
			continue
		}
		allActions[createActionKey(modAction)] = true
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return allActions, nil
}

func addUserUUID(ctx context.Context, tx pgx.Tx, userUUID string, userUUIDtoDBID map[string]int64) error {
	_, exists := userUUIDtoDBID[userUUID]
	if exists {
		return nil
	}
	userDBID, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return err
	}
	userUUIDtoDBID[userUUID] = userDBID
	return nil
}

func migrateActions(ctx context.Context, dbPool *pgxpool.Pool, actionsToMigrate []*ms.ModAction) error {
	tx, err := dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userUUIDtoDBIDs := map[string]int64{}

	for _, action := range actionsToMigrate {
		err = addUserUUID(ctx, tx, action.UserId, userUUIDtoDBIDs)
		if err != nil {
			return err
		}

		userDBID, userExists := userUUIDtoDBIDs[action.UserId]
		if !userExists {
			return fmt.Errorf("user DBID not found: %s\n", action.UserId)
		}

		applierDBIDOption := pgtype.Int8{Valid: false}

		if action.ApplierUserId != "" && action.ApplierUserId != "AUTOMOD" {
			err = addUserUUID(ctx, tx, action.ApplierUserId, userUUIDtoDBIDs)
			if err != nil {
				return err
			}
			applierDBID, applierExists := userUUIDtoDBIDs[action.ApplierUserId]
			if !applierExists {
				return fmt.Errorf("applier DBID not found: %s\n", action.ApplierUserId)
			}
			applierDBIDOption.Int64 = applierDBID
			applierDBIDOption.Valid = true
		}

		removerDBIDOption := pgtype.Int8{Valid: false}

		if action.RemoverUserId != "" && action.RemoverUserId != "AUTOMOD" {
			err = addUserUUID(ctx, tx, action.RemoverUserId, userUUIDtoDBIDs)
			if err != nil {
				return err
			}
			removerDBID, removerExists := userUUIDtoDBIDs[action.RemoverUserId]
			if !removerExists {
				return fmt.Errorf("remover DBID not found: %s\n", action.RemoverUserId)
			}
			removerDBIDOption.Int64 = removerDBID
			removerDBIDOption.Valid = true
		}

		err = user.ApplySingleActionDB(ctx, tx, userDBID, applierDBIDOption, removerDBIDOption, action)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	ids, err := userStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}
	log.Info().Int("ids", len(ids)).Msg("count-user-ids")

	jsonActions := map[string]*ms.ModAction{}
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		panic(err)
	}

	for _, uid := range ids {
		user, err := common.GetUserBy(
			ctx, tx, &common.CommonDBConfig{
				TableType:      common.UsersTable,
				SelectByType:   common.SelectByUUID,
				Value:          uid,
				IncludeProfile: false})
		if err != nil {
			panic(err)
		}
		for _, currentAction := range user.Actions.Current {
			actionKey := createActionKey(currentAction)
			existingAction, exists := jsonActions[actionKey]
			if exists {
				log.Info().Msgf("current action already exists: %v\n%v", currentAction, existingAction)
				continue
			}
			jsonActions[actionKey] = currentAction
		}
		for _, historicAction := range user.Actions.History {
			actionKey := createActionKey(historicAction)
			existingAction, exists := jsonActions[actionKey]
			if exists {
				log.Info().Msgf("historic action already exists: %v\n%v", historicAction, existingAction)
				continue
			}
			jsonActions[actionKey] = historicAction
		}
	}
	tx.Rollback(ctx)

	dbActions, err := getAllActions(ctx, pool)
	if err != nil {
		panic(err)
	}

	actionsToMigrate := []*ms.ModAction{}
	var earliestJsonAction *ms.ModAction
	var latestJsonAction *ms.ModAction
	for key, jsonAction := range jsonActions {
		_, existsInDB := dbActions[key]
		if !existsInDB {
			actionsToMigrate = append(actionsToMigrate, jsonAction)
			if earliestJsonAction == nil || jsonAction.StartTime.AsTime().Before(earliestJsonAction.StartTime.AsTime()) {
				earliestJsonAction = jsonAction
			}
			if latestJsonAction == nil || jsonAction.StartTime.AsTime().After(latestJsonAction.StartTime.AsTime()) {
				latestJsonAction = jsonAction
			}
		}
	}

	if len(actionsToMigrate) > 0 {
		if earliestJsonAction == nil {
			panic("earliest action is nil")
		}
		if latestJsonAction == nil {
			panic("latest action is nil")
		}
		fmt.Printf("Found %d actions to migrate, from %s to %s\n", len(actionsToMigrate), earliestJsonAction.StartTime.AsTime().String(), latestJsonAction.StartTime.AsTime().String())
		if len(os.Args) == 2 && os.Args[1] == "write" {
			fmt.Println("Writing actions to db...")
			err = migrateActions(ctx, pool, actionsToMigrate)
			if err != nil {
				panic(err)
			}
		}
	} else {
		fmt.Println("No actions to migrate found")
	}
}
