package soughtgame

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lithammer/shortuuid"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"
)

func TestSoughtGame(t *testing.T) {
	pool, store := recreateDB()
	is := is.New(t)
	ctx := context.Background()

	sg := newSoughtGame()

	err := store.New(ctx, sg)
	is.NoErr(err)

	sgID, err := sg.ID()
	is.NoErr(err)

	sgGet, err := store.Get(ctx, sgID)
	is.NoErr(err)
	is.Equal(sg, sgGet)

	seekerConnId, err := sg.SeekerConnID()
	is.NoErr(err)

	sgGetBySeekerConnId, err := store.GetBySeekerConnID(ctx, seekerConnId)
	is.NoErr(err)
	is.Equal(sg, sgGetBySeekerConnId)

	receiverConnId, err := sg.ReceiverConnID()
	is.NoErr(err)

	sgGetByReceiverConnId, err := store.GetByReceiverConnID(ctx, receiverConnId)
	is.NoErr(err)
	is.Equal(sg, sgGetByReceiverConnId)

	newReceiverId := "new_receiver_id"
	sgPresentReceiver := newSoughtGame()
	sgPresentReceiver.SeekRequest.ReceiverState = pb.SeekState_PRESENT
	sgPresentReceiver.SeekRequest.ReceivingUser.UserId = newReceiverId

	err = store.New(ctx, sgPresentReceiver)
	is.NoErr(err)

	sgUpdateForReceiver, err := store.UpdateForReceiver(ctx, newReceiverId)
	is.NoErr(err)
	is.Equal(sgUpdateForReceiver.SeekRequest.ReceiverState, pb.SeekState_ABSENT)

	newReceiverConnId := "new_receiver_conn_id"
	sgPresentReceiverConnId := newSoughtGame()
	sgPresentReceiverConnId.SeekRequest.ReceiverState = pb.SeekState_PRESENT
	sgPresentReceiverConnId.SeekRequest.ReceiverConnectionId = newReceiverConnId

	err = store.New(ctx, sgPresentReceiverConnId)
	is.NoErr(err)

	sgUpdateForReceiverConnId, err := store.UpdateForReceiverConnID(ctx, sgPresentReceiverConnId.SeekRequest.ReceiverConnectionId)
	is.NoErr(err)
	is.Equal(sgUpdateForReceiverConnId.SeekRequest.ReceiverState, pb.SeekState_ABSENT)

	// list open
	listedSoughtGames, err := store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 2)

	tourneyId := "tourney_id"

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, tourneyId)
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 0)

	sgTourney := newSoughtGame()
	sgTourney.SeekRequest.TournamentId = tourneyId

	err = store.New(ctx, sgTourney)
	is.NoErr(err)

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, tourneyId)
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 1)

	exists, err := store.ExistsForUser(ctx, common.DefaultSeekRequest.User.UserId)
	is.NoErr(err)
	is.True(exists)

	exists, err = store.ExistsForUser(ctx, "other_user_id")
	is.NoErr(err)
	is.True(!exists)

	exists, err = store.UserMatchedBy(ctx, common.DefaultSeekRequest.User.UserId, common.DefaultSeekRequest.ReceivingUser.UserId)
	is.NoErr(err)
	is.True(!exists)

	exists, err = store.UserMatchedBy(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, common.DefaultSeekRequest.User.UserId)
	is.NoErr(err)
	is.True(exists)

	err = store.Delete(ctx, sgID)
	is.NoErr(err)

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 2)

	differentSeekerConnId := "some_conn_id"
	sgDifferentSeekerConnId := newSoughtGame()
	sgDifferentSeekerConnId.SeekRequest.SeekerConnectionId = differentSeekerConnId

	err = store.New(ctx, sgDifferentSeekerConnId)
	is.NoErr(err)

	_, err = store.GetBySeekerConnID(ctx, differentSeekerConnId)
	is.NoErr(err)

	sgGottenDifferentSeekerConnId, err := store.DeleteForSeekerConnID(ctx, differentSeekerConnId)
	is.NoErr(err)
	is.Equal(sgDifferentSeekerConnId.SeekRequest.GameRequest.RequestId, sgGottenDifferentSeekerConnId.SeekRequest.GameRequest.RequestId)

	differentSeekerId := "other_seeker_id"
	sgDifferentSeekerId := newSoughtGame()
	sgDifferentSeekerId.SeekRequest.User.UserId = differentSeekerId
	err = store.New(ctx, sgDifferentSeekerId)
	is.NoErr(err)

	sgDeleted, err := store.DeleteForUser(ctx, differentSeekerId)
	is.NoErr(err)
	is.Equal(sgDifferentSeekerId.SeekRequest.GameRequest.RequestId, sgDeleted.SeekRequest.GameRequest.RequestId)

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 2)

	sgOld := newSoughtGame()
	err = store.New(ctx, sgOld)
	is.NoErr(err)
	sgOldUUID, err := sgOld.ID()
	is.NoErr(err)
	err = commondb.UpdateWithPool(ctx, pool, []string{"created_at"}, []interface{}{time.Now().Add(-time.Hour)}, &commondb.CommonDBConfig{TableType: commondb.SoughtGamesTable, SelectByType: commondb.SelectByUUID, Value: sgOldUUID})
	is.NoErr(err)

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 3)

	err = store.ExpireOld(ctx)
	is.NoErr(err)

	listedSoughtGames, err = store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(listedSoughtGames), 2)

	store.Disconnect()
}

func TestSoughtGameNullValues(t *testing.T) {
	pool, store := recreateDB()
	is := is.New(t)
	ctx := context.Background()

	sg := newSoughtGame()

	err := store.New(ctx, sg)
	is.NoErr(err)

	sgID, err := sg.ID()
	is.NoErr(err)

	setNullValues(ctx, pool, []interface{}{sgID})

	sgGet, err := store.Get(ctx, sgID)
	is.NoErr(err)
	is.Equal(&entity.SoughtGame{SeekRequest: &pb.SeekRequest{}}, sgGet)

	soughtGames, err := store.ListOpenSeeks(ctx, common.DefaultSeekRequest.ReceivingUser.UserId, "")
	is.NoErr(err)
	is.Equal(len(soughtGames), 0)

	store.Disconnect()
}

func newSoughtGame() *entity.SoughtGame {
	sg := &entity.SoughtGame{SeekRequest: proto.Clone(&common.DefaultSeekRequest).(*pb.SeekRequest)}
	sg.SeekRequest.GameRequest.RequestId = shortuuid.New()
	return sg
}

func setNullValues(ctx context.Context, pool *pgxpool.Pool, sgIds []interface{}) {
	tx, err := pool.BeginTx(ctx, commondb.DefaultTxOptions)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	inClause := commondb.BuildIn(len(sgIds), 1)

	updateStmt := fmt.Sprintf("UPDATE soughtgames SET seeker = NULL, conn_id = NULL, receiver = NULL, receiver_conn_id = NULL, seeker_conn_id = NULL, request = NULL WHERE uuid IN (%s)", inClause)
	result, err := tx.Exec(ctx, updateStmt, sgIds...)
	if err != nil {
		panic(err)
	}
	if result.RowsAffected() == 0 {
		panic(fmt.Sprintf("no rows affected: %v", sgIds))
	}

	if err := tx.Commit(ctx); err != nil {
		panic(err)
	}
}

func recreateDB() (*pgxpool.Pool, *DBStore) {
	err := commondb.RecreateTestDB()
	if err != nil {
		panic(err)
	}

	pool, err := commondb.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	store, err := NewDBStore(pool)
	if err != nil {
		panic(err)
	}
	return pool, store
}
