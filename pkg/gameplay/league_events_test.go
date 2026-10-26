package gameplay

import (
	"testing"

	"github.com/google/uuid"
	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestLeagueGameEvent(t *testing.T) {
	is := is.New(t)
	league := uuid.MustParse("0b6f3a52-1d4e-4c3a-9f0e-6a1b2c3d4e5f")
	season := uuid.MustParse("11111111-2222-4333-8444-555555555555")
	division := uuid.MustParse("66666666-7777-4888-9999-aaaaaaaaaaaa")

	evt := leagueGameEvent(league, season, division, "g1", []string{"u0", "u1"}, true)

	is.Equal(evt.Type, pb.MessageType_LEAGUE_GAME_UPDATED)
	// Published on the channel the league's page is registered in.
	is.Equal(evt.Audience(), []string{"channel.league-0b6f3a521d4e4c3a9f0e6a1b2c3d4e5f"})
	is.Equal(LeagueChannel(league), "league-0b6f3a521d4e4c3a9f0e6a1b2c3d4e5f")

	msg, ok := evt.Event.(*pb.LeagueGameUpdatedEvent)
	is.True(ok)
	is.Equal(msg.LeagueId, league.String())
	is.Equal(msg.SeasonId, season.String())
	is.Equal(msg.DivisionId, division.String())
	is.Equal(msg.GameId, "g1")
	is.Equal(msg.PlayerIds, []string{"u0", "u1"})
	is.True(msg.Ended)
}
