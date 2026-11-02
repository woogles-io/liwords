package gameplay

import (
	"context"
	"errors"
	"fmt"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/config"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCalculateReturnedTiles(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()

	testcases := []struct {
		letdist          string
		playerRack       string
		lastEventRack    string
		lastEventTiles   string
		expectedReturned string
	}{
		{"english", "ABCCDEF", "CCDJMUY", "JUM.Y", "ABEF"},
		{"english", "ZY??YVA", "VYAYNKE", "KA.VYEN", "??AVYZ"},
		{"english", "BEFITAR", "DANGLES", "GEN..DL.A....S", "ABEFIRT"},
		// machine letter order puts the cedilla C right after the C, where
		// the order of the labels' bytes would put it after the Z
		{"catalan", "ÇAZ?C", "", "", "?ACÇZ"},
	}

	for _, tc := range testcases {
		tiles, err := calculateReturnedTiles(cfg, tc.letdist, tc.playerRack, tc.lastEventRack, tc.lastEventTiles)
		is.NoErr(err)
		is.Equal(tiles, tc.expectedReturned)
	}

}

func TestChallengeResultLine(t *testing.T) {
	is := is.New(t)
	is.Equal(challengeResultLine(true, ""), "Challenged play was valid")
	// the tiles stay in calculateReturnedTiles's machine letter order, the
	// blanks first
	is.Equal(challengeResultLine(false, "??AVYZ"), "Play was challenged off the board! Returned tiles: ??AVYZ")
	// a play that drew nothing, from an empty bag, returns nothing
	is.Equal(challengeResultLine(false, ""), "Play was challenged off the board!")
}

type fakeChats struct {
	added []*pb.ChatMessage
	fail  string // the channel whose AddChat fails
}

func (f *fakeChats) AddChat(ctx context.Context, senderUsername, senderUID, msg, channel,
	channelFriendly string, regulateChat bool) (*pb.ChatMessage, error) {
	if channel == f.fail {
		return nil, errors.New("no redis")
	}
	if regulateChat {
		return nil, errors.New("a server line must not be regulated")
	}
	cm := &pb.ChatMessage{Username: senderUsername, UserId: senderUID, Channel: channel,
		Message: msg, Timestamp: int64(len(f.added) + 1), Id: fmt.Sprintf("%d-0", len(f.added)+1)}
	f.added = append(f.added, cm)
	return cm, nil
}

func TestPostChallengeResult(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	line := "Play was challenged off the board! Returned tiles: ?EIR"
	chats := &fakeChats{}
	evts, posted := postChallengeResult(ctx, chats, "g1", line)
	is.True(posted)
	is.Equal(len(chats.added), 2)
	is.Equal(len(evts), 2)
	for i, want := range []struct{ channel, audience string }{
		{"chat.game.g1", "game.g1"},
		{"chat.gametv.g1", "gametv.g1"},
	} {
		cm := chats.added[i]
		is.Equal(cm.Channel, want.channel)
		is.Equal(cm.Username, "")
		is.Equal(cm.UserId, "")
		is.Equal(cm.Message, line)
		is.Equal(evts[i].Type, pb.MessageType_CHAT_MESSAGE)
		is.Equal(evts[i].Event, cm)
		is.Equal(evts[i].Audience(), []string{want.audience})
	}

	// if either line cannot be stored, nothing goes out, the result says so,
	// and the page shows the line itself
	for _, fail := range []string{"chat.game.g1", "chat.gametv.g1"} {
		chats := &fakeChats{fail: fail}
		evts, posted := postChallengeResult(ctx, chats, "g1", line)
		is.True(!posted)
		is.Equal(len(evts), 0)
	}
}

func TestSanitizeGCGNicknames(t *testing.T) {
	is := is.New(t)
	hist := &macondopb.GameHistory{
		Players: []*macondopb.PlayerInfo{
			{Nickname: "Richards, Nigel"},
			{Nickname: "Iamudom, Charnwit"},
		},
	}
	sanitizeGCGNicknames(hist)
	is.Equal(hist.Players[0].Nickname, "RichardsNigel")
	is.Equal(hist.Players[1].Nickname, "IamudomCharnwit")
}
