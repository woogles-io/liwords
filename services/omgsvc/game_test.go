package omgsvc

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

var NatsUrl = os.Getenv("NATS_URL")

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	os.Exit(m.Run())
}

type mockBus struct {
	published map[string][][]byte
}

func (m *mockBus) Request(subject string, data []byte, opts ...ipc.Option) ([]byte, error) {
	if subject == "storesvc.omgwords.newgame" {
		ig := &pb.InstantiateGame{}
		err := proto.Unmarshal(data, ig)
		if err != nil {
			return nil, err
		}

		resp := &pb.InstantiateGameResponse{
			Id: "uniq-game-id", GameInfo: &pb.GameInfoResponse{
				Players: []*pb.PlayerInfo{
					{UserId: ig.UserIds[0], Nickname: "cesar"},
					{UserId: ig.UserIds[1], Nickname: "josh"},
				},
				TournamentId: "galactic-wespac",
			}}
		return proto.Marshal(resp)
	} else if subject == "storesvc.omgwords.readygame" {
		resp := &pb.ReadyForGameResponse{}
		resp.ReadyToStart = true
		log.Info().Msg("marshalling rts")
		return proto.Marshal(resp)
	} else if subject == "storesvc.omgwords.startgame" {
		resp := &pb.ResetTimersAndStartResponse{}
		resp.GameHistoryRefresher = &pb.GameHistoryRefresher{
			History: &macondopb.GameHistory{
				Uid: "some-game-id",
				Players: []*macondopb.PlayerInfo{
					{UserId: "uid-0", Nickname: "cesar"},
					{UserId: "uid-1", Nickname: "josh"},
				},
				LastKnownRacks: []string{"DDIIKS?", "AEEINST"},
			},
			TimePlayer1: 10000,
			TimePlayer2: 12000,
		}
		return proto.Marshal(resp)
	}
	return nil, errors.New(subject + " not handled")
}

func (m *mockBus) PublishToTopic(topic string, data []byte) error {
	if m.published == nil {
		m.published = make(map[string][][]byte)
	}
	if m.published[topic] == nil {
		m.published[topic] = make([][]byte, 0)
	}
	m.published[topic] = append(m.published[topic], data)
	return nil
}

func (m *mockBus) PublishToUser(user string, data []byte, optionalChannel string) error {
	var fullChannel string
	if optionalChannel == "" {
		fullChannel = "user." + user
	} else {
		fullChannel = "user." + user + "." + optionalChannel
	}
	return m.PublishToTopic(fullChannel, data)
}

func TestGameInstantiation(t *testing.T) {
	is := is.New(t)

	bus := &mockBus{}
	// 44 = Game instantiation (see ipc.proto)
	i := &pb.InstantiateGame{
		UserIds: []string{"some-user-id", "user-id-2"},
		ConnIds: []string{"some-conn-id", "conn-id-2"},
		GameRequest: &pb.GameRequest{
			Lexicon: "CSW21",
			Rules: &pb.GameRules{
				BoardLayoutName:        "CrosswordGame",
				LetterDistributionName: "english",
				VariantName:            "classic",
			},
			InitialTimeSeconds: 600,
			RequestId:          "abcdef",
		},
		AssignedFirst: 1,
		TournamentData: &pb.TournamentDataForGame{
			Tid: "galactic-wespac",
		},
	}
	bts, err := proto.Marshal(i)
	is.NoErr(err)
	err = MsgHandler(
		context.Background(), bus,
		"omgsvc.pb.44.auth.some-user-id.some-conn-id", bts,
		"replychan")
	is.NoErr(err)
	is.Equal(len(bus.published), 5)

	resp := &pb.InstantiateGameResponse{
		Id: "uniq-game-id", GameInfo: &pb.GameInfoResponse{
			Players: []*pb.PlayerInfo{
				{UserId: "some-user-id", Nickname: "cesar"},
				{UserId: "user-id-2", Nickname: "josh"},
			},
			TournamentId: "galactic-wespac",
		}}
	bts, err = proto.Marshal(resp)
	is.NoErr(err)
	is.Equal(bus.published["replychan"][0], bts)

	userMsg := &pb.NewGameEvent{
		GameId:       "uniq-game-id",
		AccepterCid:  "some-conn-id",
		RequesterCid: "conn-id-2",
	}
	bts, err = proto.Marshal(userMsg)
	is.NoErr(err)
	is.Equal(bus.published["user.pb.8.some-user-id"], [][]byte{bts})
	is.Equal(bus.published["user.pb.8.user-id-2"], [][]byte{bts})

	bts, err = proto.Marshal(resp.GameInfo)
	is.NoErr(err)
	is.Equal(bus.published["lobby.pb.12.newLiveGame"], [][]byte{bts})
	is.Equal(bus.published["tournament.pb.12.galactic-wespac.newLiveGame"], [][]byte{bts})

}

func TestReadyForGame(t *testing.T) {
	is := is.New(t)

	bus := &mockBus{}

	ready := &pb.ReadyForGame{}
	ready.GameId = "some-game-id"
	ready.UserId = "uid-0"
	ready.ConnId = "some-conn-id"
	bts, err := proto.Marshal(ready)
	is.NoErr(err)

	err = MsgHandler(
		context.Background(), bus,
		"omgsvc.pb.25.auth.uid-0.some-conn-id", bts,
		"replychan")
	is.NoErr(err)
	is.Equal(len(bus.published), 3)

	ghr := &pb.GameHistoryRefresher{
		History: &macondopb.GameHistory{
			Uid: "some-game-id",
			Players: []*macondopb.PlayerInfo{
				{UserId: "uid-0", Nickname: "cesar"},
				{UserId: "uid-1", Nickname: "josh"},
			},
			LastKnownRacks: []string{"DDIIKS?", "AEEINST"},
		},
		TimePlayer1: 10000,
		TimePlayer2: 12000,
	}
	bts, err = proto.Marshal(ghr)
	is.NoErr(err)
	is.Equal(bus.published["gametv.pb.6.some-game-id"], [][]byte{bts})

	ghrNoRack0 := proto.Clone(ghr).(*pb.GameHistoryRefresher)
	ghrNoRack0.History.LastKnownRacks[0] = ""
	bts0, err := proto.Marshal(ghrNoRack0)
	is.NoErr(err)
	ghrNoRack1 := proto.Clone(ghr).(*pb.GameHistoryRefresher)
	ghrNoRack1.History.LastKnownRacks[1] = ""
	bts1, err := proto.Marshal(ghrNoRack1)
	is.NoErr(err)

	is.Equal(bus.published["user.pb.6.uid-0.game.some-game-id"], [][]byte{bts1})
	is.Equal(bus.published["user.pb.6.uid-1.game.some-game-id"], [][]byte{bts0})

}
