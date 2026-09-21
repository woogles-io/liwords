package mod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/macondo/game"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

type NotorietyStore interface {
	// RecordAutomodVerdict records that automod has judged this player in this
	// game, returning false if it already had. See automod_verdicts.
	RecordAutomodVerdict(ctx context.Context, playerID string, gameID string, verdict int) (bool, error)
	AddNotoriousGame(ctx context.Context, gameID string, playerID string, gameType int, time int64) error
	GetNotoriousGames(ctx context.Context, playerID string, limit int) ([]*ms.NotoriousGame, error)
	DeleteNotoriousGames(ctx context.Context, playerID string) error
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
	ms.NotoriousGameType_SANDBAG:              "Premature Resignation",
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

// Verdict is what a finished game says about each player's conduct.
//
// It is a description, not an effect: deciding is separate from acting on the
// decision, so the decision can be tested against a game without a database, a
// user store, or a game that was ever really played.
type Verdict struct {
	// LoserIdx is the player the game was lost by, never negative -- a drawn
	// game reports 0, which is what the original code's `LoserIdx * LoserIdx`
	// arrived at less obviously.
	LoserIdx int
	Winner   ms.NotoriousGameType
	Loser    ms.NotoriousGameType
}

// Classify decides what a finished game says about each player.
//
// Pure: it reads the game and nothing else, and changes nothing. Everything it
// looks at is on the game -- the end reason, the loser, the event log, the
// timers and the meta events. Notably it does *not* look at whose turn it is;
// a game is over, and the loser is whoever the loser is.
func Classify(g *entity.Game, cfg *wglconfig.Config, isBotGame bool) (Verdict, error) {
	v := Verdict{Winner: ms.NotoriousGameType_GOOD, Loser: ms.NotoriousGameType_GOOD}

	history := g.History()
	// Perhaps too cute, but solves cases where g.LoserIdx is -1
	v.LoserIdx = g.LoserIdx * g.LoserIdx
	loserId := history.Players[v.LoserIdx].UserId
	winnerIdx := 1 - v.LoserIdx

	totalGameTime := g.GameReq.InitialTimeSeconds + (60 * g.GameReq.MaxOvertimeMinutes)

	if (g.GameEndReason == ipc.GameEndReason_TIME || g.GameEndReason == ipc.GameEndReason_RESIGNED) &&
		totalGameTime > int32(UnreasonableTime) && !isBotGame {
		// g.LoserIdx should never be -1, but if it is somehow, then the whole app will
		// crash, so let's just be sure
		if g.LoserIdx == -1 {
			return v, errors.New("game ended in resignation but does not have a winner")
		}
		// Someone lost on time, determine if the loser made no plays at all
		var loserLastEvent *pb.GameEvent
		for i := len(history.Events) - 1; i >= 0; i-- {
			evt := history.Events[i]
			if evt.PlayerIndex == uint32(v.LoserIdx) && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
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
				v.Loser = ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE
			} else {
				v.Loser = ms.NotoriousGameType_NO_PLAY
			}
		} else if g.GameEndReason == ipc.GameEndReason_RESIGNED {
			timeOfResignation := int32(g.Timers.TimeRemaining[g.LoserIdx])
			if unreasonableTime(loserLastEvent.MillisRemaining - timeOfResignation) {
				v.Loser = ms.NotoriousGameType_SITTING
			}
		} else if unreasonableTime(loserLastEvent.MillisRemaining) {
			// The loser let their clock run down, this is rude
			v.Loser = ms.NotoriousGameType_SITTING
		}
	}

	// Check for excessive phonies
	if v.Winner == ms.NotoriousGameType_GOOD {
		excessive, err := excessivePhonies(history, cfg, winnerIdx)
		if err != nil {
			return v, err
		}
		if excessive {
			v.Winner = ms.NotoriousGameType_EXCESSIVE_PHONIES
		}
	}

	if v.Loser == ms.NotoriousGameType_GOOD {
		excessive, err := excessivePhonies(history, cfg, v.LoserIdx)
		if err != nil {
			return v, err
		}
		if excessive {
			v.Loser = ms.NotoriousGameType_EXCESSIVE_PHONIES
		}
	}

	// Now check for sandbagging
	if g.GameEndReason == ipc.GameEndReason_RESIGNED && v.Loser == ms.NotoriousGameType_GOOD {
		// This could be a case of sandbagging
		totalMoves := 0
		for i := 0; i < len(history.Events); i++ {
			evt := history.Events[i]
			if evt.PlayerIndex == uint32(v.LoserIdx) && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
				evt.Type == pb.GameEvent_EXCHANGE) {
				totalMoves++
			}
		}
		// scoreDifference := int(g.Quickdata.FinalScores[g.WinnerIdx] - g.Quickdata.FinalScores[g.LoserIdx])
		// if totalMoves < SandbaggingThreshold && scoreDifference/totalMoves < InsurmountablePerTurnScore {
		if totalMoves < SandbaggingThreshold {
			v.Loser = ms.NotoriousGameType_SANDBAG
		}
	}

	return v, nil
}

// Automod judges a finished game and applies the result to both players.
func Automod(ctx context.Context, us user.Store, ns NotorietyStore, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	cfg, err := config.Ctx(ctx)
	if err != nil {
		return err
	}

	v, err := Classify(g, cfg.WGLConfig(), u0.IsBot || u1.IsBot)
	if err != nil {
		return err
	}

	luser := u0
	wuser := u1

	if v.LoserIdx == 1 {
		luser, wuser = wuser, luser
	}

	if !wuser.IsBot {
		if err := updateNotoriety(ctx, us, ns, wuser, g.Uid(), v.Winner); err != nil {
			return err
		}
	}

	if !luser.IsBot {
		if err := updateNotoriety(ctx, us, ns, luser, g.Uid(), v.Loser); err != nil {
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
	games, err := ns.GetNotoriousGames(ctx, uuid, limit)
	if err != nil {
		return 0, nil, err
	}
	return user.Notoriety, games, nil
}

func FormatNotorietyReport(ctx context.Context, ns NotorietyStore, uuid string, limit int) (string, error) {
	games, err := ns.GetNotoriousGames(ctx, uuid, limit)
	if err != nil {
		return "", err
	}

	var report strings.Builder
	for _, game := range games {
		fmt.Fprintf(&report, "%s (%d): <https://woogles.io/game/%s> (%s)\n",
			BehaviorToString[game.Type], BehaviorToScore[game.Type], game.Id,
			game.CreatedAt.AsTime().UTC().Format(time.RFC1123Z))
	}
	return report.String(), nil
}

func ResetNotoriety(ctx context.Context, us user.Store, ns NotorietyStore, uuid string) error {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	err = ns.DeleteNotoriousGames(ctx, user.UUID)
	if err != nil {
		return err
	}
	return us.SetNotoriety(ctx, user.UUID, 0)
}

// updateNotoriety applies one verdict to one player, once.
//
// Its effects are not idempotent -- a bad game adds to the score and files a
// row, a good one subtracts -- so it records the verdict first and does nothing
// if that verdict was already recorded. Before this, running automod twice for
// a game penalised a player twice, and the only thing preventing it was that
// every caller of performEndgameDuties happened to check whether the game had
// already ended.
func updateNotoriety(ctx context.Context, us user.Store, ns NotorietyStore, user *entity.User, guid string, ngt ms.NotoriousGameType) error {
	recorded, err := ns.RecordAutomodVerdict(ctx, user.UUID, guid, int(ngt))
	if err != nil {
		return err
	}
	if !recorded {
		log.Debug().Str("username", user.Username).Str("gameID", guid).
			Msg("automod-already-applied")
		return nil
	}

	previousNotorietyScore := user.Notoriety
	newNotoriety := user.Notoriety
	if ngt != ms.NotoriousGameType_GOOD {

		// The user misbehaved, add this game to the list of notorious games
		err := ns.AddNotoriousGame(ctx, user.UUID, guid, int(ngt), notoriousGameTimestamp())
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
				StartTime:     timestamppb.Now(),
				ApplierUserId: AutomodUserId,
				Duration:      int32(DurationMultiplier * (newNotoriety - NotorietyThreshold))}
			err = ApplyActions(ctx, us, nil, "", []*ms.ModAction{action})
			if err != nil {
				return err
			}
			notorietyReport, err := FormatNotorietyReport(ctx, ns, user.UUID, 10)
			// Failing to get the report should not be fatal since it would just be
			// an inconvenience for the moderators, so just log the error and move on
			if err != nil {
				notorietyReport = err.Error()
				log.Err(err).Str("error", err.Error()).Msg("notoriety-report-error")
			}
			moderatorMessage := fmt.Sprintf("\n### Notoriety Report:\n%s\nCurrent Notoriety: %d", notorietyReport, newNotoriety)
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
		return us.SetNotoriety(ctx, user.UUID, newNotoriety)
	}
	return nil
}

func excessivePhonies(history *pb.GameHistory, cfg *wglconfig.Config, pidx int) (bool, error) {
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
	cfg *wglconfig.Config) (bool, error) {
	// An event that formed no words cannot have formed a phony, so decide that
	// before loading a lexicon. Loading it first meant a game whose lexicon is
	// missing or unnamed failed the phony check outright rather than answering
	// the question it could answer.
	if len(event.WordsFormed) == 0 {
		return false, nil
	}
	phony := false
	gd, err := kwg.GetKWG(cfg, history.Lexicon)
	if err != nil {
		return phony, err
	}
	for _, word := range event.WordsFormed {
		phony, err := isPhony(gd, word, history.Variant)
		if err != nil {
			return false, err
		}
		if phony {
			return phony, nil
		}
	}
	return false, nil
}

func isPhony(gd *kwg.KWG, word, variant string) (bool, error) {
	lex := kwg.Lexicon{KWG: *gd}
	machineWord, err := tilemapping.ToMachineWord(word, lex.GetAlphabet())
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
