package mod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gaddag"
	"github.com/domino14/macondo/game"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog/log"
)

type NotorietyStore interface {
	AddNotoriousGame(gameID string, playerID string, gameType int, time int64) error
	GetNotoriousGames(playerID string, limit int) ([]*ms.NotoriousGame, error)
	DeleteNotoriousGames(playerID string) error
}

var BehaviorToScore map[ms.NotoriousGameType]int = map[ms.NotoriousGameType]int{
	ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE: 10,
	ms.NotoriousGameType_EXCESSIVE_PHONIES:    8,
	ms.NotoriousGameType_NO_PLAY:              6,
	ms.NotoriousGameType_SITTING:              4,
	ms.NotoriousGameType_SANDBAG:              4,
}

var BehaviorToString map[ms.NotoriousGameType]string = map[ms.NotoriousGameType]string{
	ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE: "No Play (Denied Nudge)",
	ms.NotoriousGameType_EXCESSIVE_PHONIES:    "Excessive Phonies",
	ms.NotoriousGameType_NO_PLAY:              "No Play",
	ms.NotoriousGameType_SITTING:              "Sitting",
	ms.NotoriousGameType_SANDBAG:              "Sandbagging",
}

var IsTesting = strings.HasSuffix(os.Args[0], ".test")

var AutomodUserId string = "AUTOMOD"
var SandbaggingThreshold int = 3

// var InsurmountablePerTurnScore int = 70
var NotorietyThreshold int = 10
var NotorietyDecrement int = 1
var DurationMultiplier int = 24 * 60 * 60
var UnreasonableTime int = 5 * 60
var ExcessivePhonyThreshold float64 = 0.5
var ExcessivePhonyMinimum int = 3
var testTimestamp int64 = 1

func Automod(ctx context.Context, us user.Store, ns NotorietyStore, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	totalGameTime := g.GameReq.InitialTimeSeconds + (60 * g.GameReq.MaxOvertimeMinutes)
	lngt := ms.NotoriousGameType_GOOD
	wngt := ms.NotoriousGameType_GOOD
	history := g.History()
	// Perhaps too cute, but solves cases where g.LoserIdex is -1
	nonNegativeLoserIdx := g.LoserIdx * g.LoserIdx
	loserId := history.Players[nonNegativeLoserIdx].UserId
	winnerIdx := 1 - nonNegativeLoserIdx

	macondoConfig, err := config.GetMacondoConfig(ctx)
	if err != nil {
		return err
	}

	isBotGame := u0.IsBot || u1.IsBot

	if (g.GameEndReason == ipc.GameEndReason_TIME || g.GameEndReason == ipc.GameEndReason_RESIGNED) &&
		totalGameTime > int32(UnreasonableTime) && !isBotGame {
		// g.LoserIdx should never be -1, but if it is somehow, then the whole app will
		// crash, so let's just be sure
		if g.LoserIdx == -1 {
			return errors.New("game ended in resignation but does not have a winner")
		}
		// Someone lost on time, determine if the loser made no plays at all
		var loserLastEvent *pb.GameEvent
		for i := len(history.Events) - 1; i >= 0; i-- {
			evt := history.Events[i]
			if evt.PlayerIndex == uint32(nonNegativeLoserIdx) && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
				evt.Type == pb.GameEvent_EXCHANGE ||
				evt.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS ||
				evt.Type == pb.GameEvent_CHALLENGE) {
				loserLastEvent = evt
				break
			}
		}

		if loserLastEvent == nil {
			// The loser didn't make a play, this is rude
			// If the loser also denied an abort or adjudication,
			// this is even ruder
			if loserDeniedNudge(g, loserId) {
				lngt = ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE
			} else {
				lngt = ms.NotoriousGameType_NO_PLAY
			}
		} else if g.GameEndReason == ipc.GameEndReason_RESIGNED {
			timeOfResignation := int32(g.Timers.TimeRemaining[g.LoserIdx])
			if unreasonableTime(loserLastEvent.MillisRemaining - timeOfResignation) {
				lngt = ms.NotoriousGameType_SITTING
			}
		} else if unreasonableTime(loserLastEvent.MillisRemaining) {
			// The loser let their clock run down, this is rude
			lngt = ms.NotoriousGameType_SITTING
		}
	}

	// Check for excessive phonies
	if wngt == ms.NotoriousGameType_GOOD {
		excessive, err := excessivePhonies(history, macondoConfig, winnerIdx)
		if err != nil {
			return err
		}
		if excessive {
			wngt = ms.NotoriousGameType_EXCESSIVE_PHONIES
		}
	}

	if lngt == ms.NotoriousGameType_GOOD {
		excessive, err := excessivePhonies(history, macondoConfig, nonNegativeLoserIdx)
		if err != nil {
			return err
		}
		if excessive {
			lngt = ms.NotoriousGameType_EXCESSIVE_PHONIES
		}
	}

	// Now check for sandbagging
	if g.GameEndReason == ipc.GameEndReason_RESIGNED && lngt == ms.NotoriousGameType_GOOD {
		// This could be a case of sandbagging
		totalMoves := 0
		for i := 0; i < len(history.Events); i++ {
			evt := history.Events[i]
			if evt.PlayerIndex == uint32(nonNegativeLoserIdx) && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
				evt.Type == pb.GameEvent_EXCHANGE) {
				totalMoves++
			}
		}
		// scoreDifference := int(g.Quickdata.FinalScores[g.WinnerIdx] - g.Quickdata.FinalScores[g.LoserIdx])
		// if totalMoves < SandbaggingThreshold && scoreDifference/totalMoves < InsurmountablePerTurnScore {
		if totalMoves < SandbaggingThreshold {
			lngt = ms.NotoriousGameType_SANDBAG
		}
	}

	luser := u0
	wuser := u1

	if nonNegativeLoserIdx == 1 {
		luser, wuser = wuser, luser
	}

	if !wuser.IsBot {
		err := updateNotoriety(ctx, us, ns, wuser, g.Uid(), wngt)
		if err != nil {
			return err
		}
	}

	if !luser.IsBot {
		err := updateNotoriety(ctx, us, ns, luser, g.Uid(), lngt)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetNotorietyReport(ctx context.Context, us user.Store, ns NotorietyStore, uuid string, limit int) (int, []*ms.NotoriousGame, error) {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return 0, nil, err
	}
	games, err := ns.GetNotoriousGames(uuid, limit)
	if err != nil {
		return 0, nil, err
	}
	return user.Notoriety, games, nil
}

func FormatNotorietyReport(ns NotorietyStore, uuid string, limit int) (string, error) {
	games, err := ns.GetNotoriousGames(uuid, limit)
	if err != nil {
		return "", err
	}

	var report strings.Builder
	for _, game := range games {
		fmt.Fprintf(&report, "%s (%d): <https://woogles.io/game/%s>\n", BehaviorToString[game.Type], BehaviorToScore[game.Type], game.Id)
	}
	return report.String(), nil
}

func ResetNotoriety(ctx context.Context, us user.Store, ns NotorietyStore, uuid string) error {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	err = ns.DeleteNotoriousGames(user.UUID)
	if err != nil {
		return err
	}
	return us.SetNotoriety(ctx, user, 0)
}

func updateNotoriety(ctx context.Context, us user.Store, ns NotorietyStore, user *entity.User, guid string, ngt ms.NotoriousGameType) error {

	previousNotorietyScore := user.Notoriety
	newNotoriety := user.Notoriety
	if ngt != ms.NotoriousGameType_GOOD {

		// The user misbehaved, add this game to the list of notorious games
		err := ns.AddNotoriousGame(user.UUID, guid, int(ngt), notoriousGameTimestamp())
		if err != nil {
			return err
		}
		gameScore, ok := BehaviorToScore[ngt]
		if ok {
			newNotoriety += gameScore
		}
		if newNotoriety > NotorietyThreshold {
			action := &ms.ModAction{UserId: user.UUID,
				Type:          ms.ModActionType_SUSPEND_RATED_GAMES,
				StartTime:     ptypes.TimestampNow(),
				ApplierUserId: AutomodUserId,
				Duration:      int32(DurationMultiplier * (newNotoriety - NotorietyThreshold))}
			err = setCurrentAction(user, action)
			if err != nil {
				return err
			}
			err = us.Set(ctx, user)
			if err != nil {
				return err
			}
			notorietyReport, err := FormatNotorietyReport(ns, user.UUID, 10)
			// Failing to get the report should not be fatal since it would just be
			// an inconvenience for the moderators, so just log the error and move on
			if err != nil {
				notorietyReport = err.Error()
				log.Err(err).Str("error", err.Error()).Msg("notoriety-report-error")
			}
			moderatorMessage := fmt.Sprintf("\nNotoriety Report:\n%s\nCurrent Notoriety: %d", notorietyReport, newNotoriety)
			sendNotification(ctx, us, user, action, moderatorMessage)
		}
	} else if newNotoriety > 0 {
		newNotoriety -= NotorietyDecrement
		if newNotoriety < 0 {
			newNotoriety = 0
		}
	}

	if previousNotorietyScore != newNotoriety {
		log.Debug().Str("username", user.Username).
			Int("previous-notoriety", previousNotorietyScore).
			Int32("notorious-game-type", int32(ngt)).
			Int("new-notoriety", newNotoriety).Msg("updating")
		return us.SetNotoriety(ctx, user, newNotoriety)
	}
	return nil
}

func excessivePhonies(history *pb.GameHistory, cfg *macondoconfig.Config, pidx int) (bool, error) {
	totalTileMoves := 0
	totalPhonies := 0
	for i := 0; i < len(history.Events); i++ {
		evt := history.Events[i]
		if evt.PlayerIndex == uint32(pidx) && evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE {
			totalTileMoves++
			isPhony, err := isPhonyEvent(evt, history, cfg)
			if err != nil {
				return false, err
			}
			if isPhony {
				totalPhonies++
			}
		}
	}
	return totalPhonies >= ExcessivePhonyMinimum && float64(totalPhonies)/float64(totalTileMoves) > ExcessivePhonyThreshold, nil
}

func unreasonableTime(millisRemaining int32) bool {
	return millisRemaining > int32(1000*UnreasonableTime)
}

func loserDeniedNudge(g *entity.Game, userId string) bool {
	for _, evt := range g.MetaEvents.Events {
		if evt.PlayerId == userId &&
			(evt.Type == ipc.GameMetaEvent_ABORT_DENIED ||
				evt.Type == ipc.GameMetaEvent_ADJUDICATION_DENIED) {
			return true
		}
	}
	return false
}

func isPhonyEvent(event *pb.GameEvent,
	history *pb.GameHistory,
	cfg *macondoconfig.Config) (bool, error) {
	phony := false
	dawg, err := gaddag.GetDawg(cfg, history.Lexicon)
	if err != nil {
		return phony, err
	}
	for _, word := range event.WordsFormed {
		phony, err := isPhony(dawg, word, history.Variant)
		if err != nil {
			return false, err
		}
		if phony {
			return phony, nil
		}
	}
	return false, nil
}

func isPhony(gd gaddag.GenericDawg, word, variant string) (bool, error) {
	lex := gaddag.Lexicon{GenericDawg: gd}
	machineWord, err := alphabet.ToMachineWord(word, lex.GetAlphabet())
	if err != nil {
		return false, err
	}
	var valid bool
	switch string(variant) {
	case string(game.VarWordSmog):
		valid = lex.HasAnagram(machineWord)
	default:
		valid = lex.HasWord(machineWord)
	}
	return !valid, nil
}

func notoriousGameTimestamp() int64 {
	if !IsTesting {
		return time.Now().Unix()
	} else {
		testTimestamp++
		return testTimestamp
	}
}
