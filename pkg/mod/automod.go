package mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/golang/protobuf/ptypes"
)

type NotorietyStore interface {
	AddNotoriousGame(gameID string, playerID string, gameType int, time int64) error
	GetNotoriousGames(playerID string) ([]*ms.NotoriousGame, error)
	DeleteNotoriousGames(playerID string) error
}

var BehaviorToScore map[ms.NotoriousGameType]int = map[ms.NotoriousGameType]int{
	ms.NotoriousGameType_NO_PLAY_IGNORE_NUDGE: 10,
	ms.NotoriousGameType_NO_PLAY:              6,
	ms.NotoriousGameType_SITTING:              4,
	ms.NotoriousGameType_SANDBAG:              4,
}

var AutomodUserId string = "AUTOMOD"
var SandbaggingThreshold int = 3

// var InsurmountablePerTurnScore int = 70
var NotorietyThreshold int = 10
var NotorietyDecrement int = 1
var DurationMultiplier int = 24 * 60 * 60
var UnreasonableTime int = 5 * 60

func Automod(ctx context.Context, us user.Store, ns NotorietyStore, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	totalGameTime := g.GameReq.InitialTimeSeconds + (60 * g.GameReq.MaxOvertimeMinutes)
	lngt := ms.NotoriousGameType_GOOD
	wngt := ms.NotoriousGameType_GOOD
	history := g.History()
	// Perhaps too cute, but solves cases where g.LoserIdex is -1
	loserNickname := history.Players[g.LoserIdx*g.LoserIdx].Nickname
	loserId := history.Players[g.LoserIdx*g.LoserIdx].UserId
	// This should not even be possible but might as well check
	if u0.Username != loserNickname && u1.Username != loserNickname {
		return fmt.Errorf("loser (%s) not found in players (%s, %s)", loserNickname, u0.Username, u1.Username)
	}

	isBotGame := u0.IsBot || u1.IsBot

	if (g.GameEndReason == realtime.GameEndReason_TIME || g.GameEndReason == realtime.GameEndReason_RESIGNED) &&
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
			if evt.Nickname == loserNickname && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
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
				lngt = ms.NotoriousGameType_NO_PLAY_IGNORE_NUDGE
			} else {
				lngt = ms.NotoriousGameType_NO_PLAY
			}
		} else if g.GameEndReason == realtime.GameEndReason_RESIGNED {
			timeOfResignation := int32(g.Timers.TimeRemaining[g.LoserIdx])
			if unreasonableTime(loserLastEvent.MillisRemaining - timeOfResignation) {
				lngt = ms.NotoriousGameType_SITTING
			}
		} else if unreasonableTime(loserLastEvent.MillisRemaining) {
			// The loser let their clock run down, this is rude
			lngt = ms.NotoriousGameType_SITTING
		}
	} else if g.GameEndReason == realtime.GameEndReason_RESIGNED {
		// This could be a case of sandbagging
		totalMoves := 0
		for i := 0; i < len(history.Events); i++ {
			evt := history.Events[i]
			if evt.Nickname == loserNickname && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
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

	if u1.Username == loserNickname {
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

func GetNotorietyReport(ctx context.Context, us user.Store, ns NotorietyStore, uuid string) (int, []*ms.NotoriousGame, error) {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return 0, nil, err
	}
	games, err := ns.GetNotoriousGames(uuid)
	if err != nil {
		return 0, nil, err
	}
	return user.Notoriety, games, nil
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
		err := ns.AddNotoriousGame(user.UUID, guid, int(ngt), time.Now().Unix())
		if err != nil {
			return err
		}
		gameScore, ok := BehaviorToScore[ngt]
		if ok {
			newNotoriety += gameScore
		}

		if newNotoriety > NotorietyThreshold {
			err = setCurrentAction(user, &ms.ModAction{UserId: user.UUID,
				Type:          ms.ModActionType_SUSPEND_GAMES,
				StartTime:     ptypes.TimestampNow(),
				ApplierUserId: AutomodUserId,
				Duration:      int32(DurationMultiplier * (newNotoriety - NotorietyThreshold))})
			if err != nil {
				return err
			}
		}
	} else if newNotoriety > 0 {
		newNotoriety -= NotorietyDecrement
		if newNotoriety < 0 {
			newNotoriety = 0
		}
	}

	if previousNotorietyScore != newNotoriety {
		return us.SetNotoriety(ctx, user, newNotoriety)
	}
	return nil
}

func unreasonableTime(millisRemaining int32) bool {
	return millisRemaining > int32(1000*UnreasonableTime)
}

func loserDeniedNudge(g *entity.Game, userId string) bool {
	for _, evt := range g.MetaEvents.Events {
		if evt.PlayerId == userId &&
			(evt.Type == realtime.GameMetaEvent_ABORT_DENIED ||
				evt.Type == realtime.GameMetaEvent_ADJUDICATION_DENIED) {
			return true
		}
	}
	return false
}
