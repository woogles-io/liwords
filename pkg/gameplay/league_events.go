package gameplay

import (
	"strings"

	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// LeagueChannel names the channel a league's page listens on: the bus
// registers the page in realm "channel-" + LeagueChannel(id), and events for
// the league go out on that channel.
func LeagueChannel(leagueID uuid.UUID) string {
	return "league-" + strings.ReplaceAll(leagueID.String(), "-", "")
}

// leagueGameEvent tells a league's page that one of its games has a new move
// or has ended, so the page can reload what the game changes.
func leagueGameEvent(leagueID, seasonID, divisionID uuid.UUID, gameID string,
	playerIDs []string, ended bool) *entity.EventWrapper {

	evt := entity.WrapEvent(&pb.LeagueGameUpdatedEvent{
		LeagueId:   leagueID.String(),
		SeasonId:   seasonID.String(),
		DivisionId: divisionID.String(),
		GameId:     gameID,
		PlayerIds:  playerIDs,
		Ended:      ended,
	}, pb.MessageType_LEAGUE_GAME_UPDATED)
	evt.AddAudience(entity.AudChannel, LeagueChannel(leagueID))
	return evt
}

// sendLeagueGameUpdated sends leagueGameEvent for g, if g is a league game.
func sendLeagueGameUpdated(g *entity.Game, ended bool) {
	if g.LeagueID == nil || g.SeasonID == nil || g.LeagueDivisionID == nil {
		return
	}
	g.SendChange(leagueGameEvent(*g.LeagueID, *g.SeasonID, *g.LeagueDivisionID,
		g.GameID(), players(g), ended))
}
