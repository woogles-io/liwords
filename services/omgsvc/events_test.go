package omgsvc

import (
	"context"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	}
	return nil, nil
}

func (m *mockBus) RequestProto(subject string, msg, resp protoreflect.ProtoMessage, opts ...ipc.Option) error {
	return nil
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
