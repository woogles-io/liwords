package soughtgame

import (
	"context"
	"testing"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/entity"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	"github.com/matryer/is"
)

func TestSession(t *testing.T) {
	pool, err := commondb.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	store, err := NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	is := is.New(t)
	ctx := context.Background()

	sg := &entity.SoughtGame{
		SeekRequest: &common.DefaultSeekRequest,
	}

	err = store.New(ctx, sg)
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

	// For a change in receiver state here

	sgUpdateForReceiver, err = store.UpdateForReceiver(ctx, sg.SeekRequest.ReceivingUser.UserId)
	is.NoErr(err)
}
