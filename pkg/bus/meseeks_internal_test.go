package bus

// Internal (package bus) tests for shouldIncludeSeek and its helpers. These live in the
// `bus` package, rather than `bus_test`, so they can exercise the unexported filtering
// logic directly without needing a live NATS connection (NewBus requires one; a bare
// &Bus{stores: ...} does not, since shouldIncludeSeek/blockedByUUIDSet/seekersByUUID only
// touch the DB stores).
//
// This specifically guards the block-filtering refactor: the original code called
// blockExists(seeker, receiver) per seek and only filtered when the *seeker* blocks the
// *receiver* (block == 0), not the reverse. The refactor replaced that with a single
// precomputed "who blocks the receiver" set (blockedByUUIDSet) checked via membership, and
// batches seeker profile lookups (seekersByUUID). These tests confirm the precomputed-set
// version preserves that exact asymmetric semantics.

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const internalTestPkg = "bus_internal_test"

func recreateInternalTestDB() *stores.Stores {
	if err := common.RecreateTestDB(internalTestPkg); err != nil {
		panic(err)
	}
	pool, err := common.OpenTestingDB(internalTestPkg)
	if err != nil {
		panic(err)
	}
	cfg := config.DefaultConfig()
	cfg.DBConnDSN = common.TestingPostgresConnDSN(internalTestPkg)
	st, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		panic(err)
	}
	return st
}

func seekFrom(seekerUUID string, receiverIsPermanent bool) *entity.SoughtGame {
	return &entity.SoughtGame{
		SeekRequest: &pb.SeekRequest{
			User:                &pb.MatchUser{UserId: seekerUUID},
			ReceiverIsPermanent: receiverIsPermanent,
			GameRequest: &pb.GameRequest{
				Lexicon: "CSW21",
			},
		},
	}
}

func TestShouldIncludeSeek_BlockAsymmetry(t *testing.T) {
	is := is.New(t)
	st := recreateInternalTestDB()
	defer st.Disconnect()
	ctx := context.Background()

	users := []*entity.User{
		{Username: "receiver1", Email: "receiver1@woogles.io", UUID: "receiver1uuid"},
		{Username: "blockingseeker", Email: "blockingseeker@woogles.io", UUID: "blockingseekeruuid"},
		{Username: "blockedseeker", Email: "blockedseeker@woogles.io", UUID: "blockedseekeruuid"},
		{Username: "normalseeker", Email: "normalseeker@woogles.io", UUID: "normalseekeruuid"},
	}
	for _, u := range users {
		is.NoErr(st.UserStore.New(ctx, u))
	}
	receiver, err := st.UserStore.GetByUUID(ctx, "receiver1uuid")
	is.NoErr(err)
	blockingSeeker, err := st.UserStore.GetByUUID(ctx, "blockingseekeruuid")
	is.NoErr(err)
	blockedSeeker, err := st.UserStore.GetByUUID(ctx, "blockedseekeruuid")
	is.NoErr(err)
	normalSeeker, err := st.UserStore.GetByUUID(ctx, "normalseekeruuid")
	is.NoErr(err)

	// blockingSeeker blocks receiver: should be filtered out of receiver's seek list.
	is.NoErr(st.UserStore.AddBlock(ctx, receiver.ID, blockingSeeker.ID))
	// receiver blocks blockedSeeker (the reverse direction): per original semantics this
	// does NOT filter the seek out (only "seeker blocks receiver" does).
	is.NoErr(st.UserStore.AddBlock(ctx, blockedSeeker.ID, receiver.ID))

	b := &Bus{stores: st}

	sgs := []*entity.SoughtGame{
		seekFrom(blockingSeeker.UUID, true),
		seekFrom(blockedSeeker.UUID, true),
		seekFrom(normalSeeker.UUID, true),
		seekFrom(receiver.UUID, true), // receiver's own seek
	}

	blockedByReceiver, err := b.blockedByUUIDSet(ctx, receiver.ID)
	is.NoErr(err)
	is.True(blockedByReceiver[blockingSeeker.UUID]) // blockingSeeker blocks receiver
	is.True(!blockedByReceiver[blockedSeeker.UUID]) // receiver blocks blockedSeeker, not the reverse
	is.True(!blockedByReceiver[normalSeeker.UUID])

	seekers, err := b.seekersByUUID(ctx, sgs)
	is.NoErr(err)
	is.Equal(len(seekers), 4)

	is.True(!b.shouldIncludeSeek(ctx, sgs[0], receiver, blockedByReceiver, seekers)) // blockingSeeker: filtered
	is.True(b.shouldIncludeSeek(ctx, sgs[1], receiver, blockedByReceiver, seekers))  // blockedSeeker: NOT filtered (reverse block doesn't count)
	is.True(b.shouldIncludeSeek(ctx, sgs[2], receiver, blockedByReceiver, seekers))  // normalSeeker: included
	is.True(b.shouldIncludeSeek(ctx, sgs[3], receiver, blockedByReceiver, seekers))  // own seek: always included
}

func TestShouldIncludeSeek_DirectMatchAndNilReceiver(t *testing.T) {
	is := is.New(t)
	st := recreateInternalTestDB()
	defer st.Disconnect()
	ctx := context.Background()

	users := []*entity.User{
		{Username: "receiver2", Email: "receiver2@woogles.io", UUID: "receiver2uuid"},
		{Username: "blockingseeker2", Email: "blockingseeker2@woogles.io", UUID: "blockingseeker2uuid"},
	}
	for _, u := range users {
		is.NoErr(st.UserStore.New(ctx, u))
	}
	receiver, err := st.UserStore.GetByUUID(ctx, "receiver2uuid")
	is.NoErr(err)
	blockingSeeker, err := st.UserStore.GetByUUID(ctx, "blockingseeker2uuid")
	is.NoErr(err)
	is.NoErr(st.UserStore.AddBlock(ctx, receiver.ID, blockingSeeker.ID))

	b := &Bus{stores: st}

	// Nil receiver (e.g. anonymous connecting user): never filtered.
	sg := seekFrom(blockingSeeker.UUID, true)
	is.True(b.shouldIncludeSeek(ctx, sg, nil, nil, nil))

	// Direct match request to the receiver bypasses the block filter, matching original
	// behavior ("Always show match requests where you're the receiver").
	directMatch := &entity.SoughtGame{
		SeekRequest: &pb.SeekRequest{
			User:                &pb.MatchUser{UserId: blockingSeeker.UUID},
			ReceivingUser:       &pb.MatchUser{UserId: receiver.UUID},
			ReceiverIsPermanent: true,
			GameRequest:         &pb.GameRequest{Lexicon: "CSW21"},
		},
	}
	blockedByReceiver, err := b.blockedByUUIDSet(ctx, receiver.ID)
	is.NoErr(err)
	is.True(blockedByReceiver[blockingSeeker.UUID])
	is.True(b.shouldIncludeSeek(ctx, directMatch, receiver, blockedByReceiver, nil))
}
