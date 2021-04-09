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
}

var AutomodUserId string = "AUTOMOD"
var NotorietyThreshold int = 10
var NotorietyDecrement int = 1
var DurationMultiplier int = 24 * 60 * 60

func Automod(ctx context.Context, us user.Store, u0 *entity.User, u1 *entity.User, g *entity.Game) error {
	// The behavior we currently check for is fairly primitive
	// This will be expanded in the future
	if g.GameEndReason == realtime.GameEndReason_TIME {
		// Someone lost on time, determine if the loser made no plays at all
		history := g.History()
		loserNickname := history.Players[1-history.Winner].Nickname
		// This should even be possible but might as well check
		if u0.Username != loserNickname && u1.Username != loserNickname {
			return fmt.Errorf("loser (%s) not found in players (%s, %s)", loserNickname, u0.Username, u1.Username)
		}
		var loserLastEvent *pb.GameEvent
		for i := len(history.Events) - 1; i >= 0; i-- {
			evt := history.Events[i]
			if evt.Nickname == loserNickname {
				loserLastEvent = evt
				break
			}
		}

		ngt := ms.NotoriousGameType_GOOD
		if loserLastEvent != nil {
			// The loser didn't make a play, this is rude
			ngt = ms.NotoriousGameType_NO_PLAY
		} else if unreasonableTime(loserLastEvent.MillisRemaining) {
			// The loser let their clock run down, this is rude
			ngt = ms.NotoriousGameType_SITTING
		}

		loserUser := u0
		if u1.Username == loserNickname {
			loserUser = u1
		}

		instantiateNotoriety(loserUser)

		if ngt != ms.NotoriousGameType_GOOD {
			// The user misbehaved, add this game to the list of
			createdAtTime, err := ptypes.TimestampProto(time.Now())
			if err != nil {
				return err
			}
			addNotoriousGame(loserUser, &ms.NotoriousGame{Id: g.Uid(), Type: ngt, CreatedAt: createdAtTime})

			gameScore, ok := BehaviorToScore[ngt]
			if ok {
				loserUser.Notoriety.Score += gameScore
			}

			if loserUser.Notoriety.Score > NotorietyThreshold {
				err = setCurrentAction(loserUser, &ms.ModAction{UserId: AutomodUserId,
					Type:     ms.ModActionType_SUSPEND_GAMES,
					Duration: int32(DurationMultiplier * loserUser.Notoriety.Score)})
				if err != nil {
					return err
				}
			}
		} else if loserUser.Notoriety.Score > 0 {
			loserUser.Notoriety.Score -= NotorietyDecrement
		}

		if ngt != ms.NotoriousGameType_GOOD || loserUser.Notoriety.Score > 0 {
			err := us.Set(ctx, loserUser)
			if err != nil {
				return err
			}
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

func addNotoriousGame(u *entity.User, ng *ms.NotoriousGame) {
	u.Notoriety.Games = append(u.Notoriety.Games, ng)
}

func unreasonableTime(millisRemaining int32) bool {
	return millisRemaining < 1000*60*5
}

func instantiateNotoriety(u *entity.User) {
	if u.Notoriety == nil {
		u.Notoriety = &entity.Notoriety{Score: 0, Games: []*ms.NotoriousGame{}}
	}
}
