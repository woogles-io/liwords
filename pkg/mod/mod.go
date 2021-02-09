package mod

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
)

var ModActionDispatching = map[ms.ModActionType]func(context.Context, user.Store, *ms.ModAction) (error) {

/*	All types are listed here for clearness
    Types that are commented are not transient
    actions but are applied over a duration of time
    ms.ModActionType_MUTE,
	ms.ModActionType_SUSPEND_ACCOUNT,
	ms.ModActionType_SUSPEND_RATED_GAMES,
	ms.ModActionType_SUSPEND_GAMES,*/
	ms.ModActionType_RESET_RATINGS: resetRatings,
	ms.ModActionType_RESET_STATS: resetStats,
	ms.ModActionType_RESET_STATS_AND_RATINGS: resetStatsAndRatings,
}

func GetActions(ctx context.Context, us user.Store, uuid string) (map[ms.ModActionType]*ms.ModAction, error) {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return nil, err
	}
	return &ms.ModActions{actions: user.CurrentActions}, nil
}

func RemoveActions(ctx context.Context, us user.Store, actions []*ms.ModAction) error {
	for _, action := range actions {
		err := removeAction(ctx, us, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeAction(ctx context.Context, us user.Store, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}

	err = removeCurrentAction(user, action)
	if err != nil {
		return err
	}

	err = us.Set(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func ApplyActions(ctx context.Context, us user.Store, actions []*ms.ModAction) error {
	for _, action := range actions {
		err := applyAction(ctx, us, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func applyAction(ctx context.Context, us user.Store, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}
	modActionFunc, ok := ModActionDispatching[action.Type](user, action)
	if ok { // This ModAction is transient
		err := modActionFunc(ctx, us, action)
		if err != nil {
			return err
		}
		action.Duration = 0
		action.EndTime = action.StartTime
		err = addActionToHistory(user, action)
		if err != nil {
			return err
		}
	} else {
		if action.Duration < 1 {
			return fmt.Errorf("nontransient moderator action has a nonpositive duration: %d", action.Duration)
		}
		err := setEndTime(action)
		if err != nil {
			return err
		}
		err = setCurrentAction(user, action)
		if err != nil {
			return err
		}
	}

	err = us.Set(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func setEndTime(action *ms.ModAction) error {
	if action.StartTime == nil {
		return errors.New("start time for moderator action is undefined")
	}
	convertedStartTime, err := time.TimeStamp(action.StartTime)
	if err != nil {
		return err
	}
	convertedEndTime := convertedStartTime.Add(time.Seconds * action.Duration)
	protoEndTime, err := time.TimestampProto(convertedEndTime)
	if err != nil {
		return err
	}
	action.EndTime = protoEndTime
	return nil
}

func addActionToHistory(user *entity.User, action *ms.ModAction) error {
	user.ActionHistory = append(user.ActionHistory, action)
	return nil
}

func setCurrentAction(user *entity.User, action *ms.ModAction) error {
	user.CurrentAction[action.Type] = action
	return nil
}

func removeCurrentAction(user *entity.User, action *ms.ModAction) error {
	addActionToHistory(user, user.CurrentAction[action.Type])
	user.CurrentAction = nil
	return nil
}

func resetRatings(ctx context.Context, us user.Store, action *ms.ModAction) error {
	return us.ResetRatings(ctx, action.UserId)
}

func resetStats(ctx context.Context, us user.Store, action *ms.ModAction) error {
	return us.ResetRatings(ctx, action.UserId)
}

func resetStatsAndRatings(ctx context.Context, us user.Store, action *ms.ModAction) error {
	err := us.ResetStats(ctx, action.UserId)
	if err != nil {
		return nil
	}
	return us.ResetRatings(ctx, action.UserId)
}