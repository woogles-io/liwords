package mod

import (
	"context"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/golang/protobuf/ptypes"
)

var BehaviorToScore map[ms.NotoriousGameType]int = map[ms.NotoriousGameType]int{
	ms.NotoriousGameType_NO_PLAY: 6,
	ms.NotoriousGameType_SITTING: 4,
	ms.NotoriousGameType_SANDBAG: 4,
}

var AutomodUserId string = "AUTOMOD"
var SandbaggingThreshold int = 3
var NotorietyThreshold int = 10
var NotorietyDecrement int = 1
var DurationMultiplier int = 24 * 60 * 60
var UnreasonableTime int = 5 * 60

func Automod(ctx context.Context, us user.Store, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	totalGameTime := g.GameReq.InitialTimeSeconds + (60 * g.GameReq.MaxOvertimeMinutes)
	lngt := ms.NotoriousGameType_GOOD
	wngt := ms.NotoriousGameType_GOOD
	history := g.History()
	// Perhaps too cute, but solves cases where g.LoserIdex is -1
	loserNickname := history.Players[g.LoserIdx*g.LoserIdx].Nickname
	// This should not even be possible but might as well check
	if u0.Username != loserNickname && u1.Username != loserNickname {
		return fmt.Errorf("loser (%s) not found in players (%s, %s)", loserNickname, u0.Username, u1.Username)
	}

	isBotGame := u0.IsBot || u1.IsBot

	if g.GameEndReason == realtime.GameEndReason_TIME && totalGameTime > int32(UnreasonableTime) && !isBotGame {
		// Someone lost on time, determine if the loser made no plays at all
		var loserLastEvent *pb.GameEvent
		for i := len(history.Events) - 1; i >= 0; i-- {
			evt := history.Events[i]
			if evt.Nickname == loserNickname && (evt.Type == pb.GameEvent_TILE_PLACEMENT_MOVE ||
				evt.Type == pb.GameEvent_PASS ||
				evt.Type == pb.GameEvent_EXCHANGE ||
				evt.Type == pb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS ||
				evt.Type == pb.GameEvent_CHALLENGE) {
				loserLastEvent = evt
				break
			}
		}

		if loserLastEvent == nil {
			// The loser didn't make a play, this is rude
			lngt = ms.NotoriousGameType_NO_PLAY
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
		err := updateNotoriety(ctx, us, wuser, g.Uid(), wngt)
		if err != nil {
			return err
		}
	}

	if !luser.IsBot {
		err := updateNotoriety(ctx, us, luser, g.Uid(), lngt)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetNotorietyReport(ctx context.Context, us user.Store, uuid string) (int, []*ms.NotoriousGame, error) {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return 0, nil, err
	}
	instantiateNotoriety(user)
	return user.Notoriety.Score, user.Notoriety.Games, nil
}

func ResetNotoriety(ctx context.Context, us user.Store, uuid string) error {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	instantiateNotoriety(user)
	user.Notoriety = &entity.Notoriety{Score: 0, Games: []*ms.NotoriousGame{}}
	return us.Set(ctx, user)
}

func updateNotoriety(ctx context.Context, us user.Store, user *entity.User, guid string, ngt ms.NotoriousGameType) error {

	instantiateNotoriety(user)
	previousNotorietyScore := user.Notoriety.Score
	if ngt != ms.NotoriousGameType_GOOD {

		// The user misbehaved, add this game to the list of notorious games
		createdAtTime, err := ptypes.TimestampProto(time.Now())
		if err != nil {
			return err
		}
		addNotoriousGame(user, &ms.NotoriousGame{Id: guid, Type: ngt, CreatedAt: createdAtTime})

		gameScore, ok := BehaviorToScore[ngt]
		if ok {
			user.Notoriety.Score += gameScore
		}

		if user.Notoriety.Score > NotorietyThreshold {
			err = setCurrentAction(user, &ms.ModAction{UserId: user.UUID,
				Type:          ms.ModActionType_SUSPEND_GAMES,
				StartTime:     ptypes.TimestampNow(),
				ApplierUserId: AutomodUserId,
				Duration:      int32(DurationMultiplier * (user.Notoriety.Score - NotorietyThreshold))})
			if err != nil {
				return err
			}
		}
	} else if user.Notoriety.Score > 0 {
		user.Notoriety.Score -= NotorietyDecrement
		if user.Notoriety.Score < 0 {
			user.Notoriety.Score = 0
		}
	}

	if previousNotorietyScore != user.Notoriety.Score {
		return us.Set(ctx, user)
	}
	return nil
}

func addNotoriousGame(u *entity.User, ng *ms.NotoriousGame) {
	u.Notoriety.Games = append(u.Notoriety.Games, ng)
}

func unreasonableTime(millisRemaining int32) bool {
	return millisRemaining > int32(1000*UnreasonableTime)
}

func instantiateNotoriety(u *entity.User) {
	if u.Notoriety == nil {
		u.Notoriety = &entity.Notoriety{Score: 0, Games: []*ms.NotoriousGame{}}
	}
}
