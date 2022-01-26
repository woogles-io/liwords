package omgsvc

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/runner"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/entity/omgwords"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

// InstantiateNewGame instantiates a game and returns it.
func InstantiateNewGame(ctx context.Context, cfg *config.Config,
	users [2]*entity.User, assignedFirst int, req *pb.GameRequest,
	tdata *omgwords.TournamentData) (*omgwords.Game, error) {

	var players []*macondopb.PlayerInfo
	var dbids [2]uint

	for idx, u := range users {
		players = append(players, &macondopb.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			RealName: u.RealNameIfNotYouth(),
		})
		dbids[idx] = u.ID
	}

	if req.Rules == nil {
		return nil, errors.New("no rules")
	}

	log.Debug().Interface("req-rules", req.Rules).Msg("new-game-rules")

	firstAssigned := false
	if assignedFirst != -1 {
		firstAssigned = true
	}

	rules, err := game.NewBasicGameRules(
		&cfg.MacondoConfig, req.Lexicon, req.Rules.BoardLayoutName,
		req.Rules.LetterDistributionName, game.CrossScoreOnly,
		game.Variant(req.Rules.VariantName))
	if err != nil {
		return nil, err
	}

	var gameRunner *runner.GameRunner
	for {
		gameRunner, err = runner.NewGameRunnerFromRules(&runner.GameOptions{
			FirstIsAssigned: firstAssigned,
			GoesFirst:       assignedFirst,
			ChallengeRule:   req.ChallengeRule,
		}, players, rules)
		if err != nil {
			return nil, err
		}

		exists, err := gameStore.Exists(ctx, gameRunner.Game.Uid())
		if err != nil {
			return nil, err
		}
		if exists {
			continue
			// This UUID exists in the database. This is only possible because
			// we are purposely shortening the UUID in macondo for nicer URLs.
			// 57^8 should still give us 111 trillion games. (and we can add more
			// characters if we get close to that number)
		}
		break
		// There's still a chance of a race condition here if another thread
		// creates the same game ID at the same time, but the chances
		// of that are so astronomically unlikely we won't bother.
	}

	entGame := omgwords.NewGame(&gameRunner.Game, req)
	entGame.PlayerDBIDs = dbids
	entGame.TournamentData = tdata

	ratingKey, err := entGame.RatingKey()
	if err != nil {
		return nil, err
	}

	// Create player info in entGame.Quickdata
	playerinfos := make([]*gs.PlayerInfo, len(players))

	for idx, u := range users {
		playerinfos[idx] = &gs.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			Rating:   u.GetRelevantRating(ratingKey),
			IsBot:    u.IsBot,
			First:    gameRunner.FirstPlayer().UserId == u.UUID,
		}
	}

	// Create the Quickdata now with the original player info.
	entGame.Quickdata = &omgwords.Quickdata{
		OriginalRequestId: req.OriginalRequestId,
		PlayerInfo:        playerinfos,
	}
	// This timestamp will be very close to whatever gets saved in the DB
	// as the CreatedAt date. We need to put it here though in order to
	// keep the cached version in sync with the saved version at the beginning.
	entGame.CreatedAt = time.Now()

	entGame.MetaEvents = &omgwords.MetaEventData{}

	// Save the game to the store.
	if err = gameStore.Create(ctx, entGame); err != nil {
		return nil, err
	}
	return entGame, nil
	// We return the instantiated game. Although the tiles have technically been
	// dealt out, we need to call StartGame to actually start the timer
	// and forward game events to the right channels.
}
