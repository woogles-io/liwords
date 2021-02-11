package mod

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
)

var ModActionDispatching = map[string]func(context.Context, user.Store, *ms.ModAction) error{

	/*	All types are listed here for clearness
		    Types that are commented are not transient
		    actions but are applied over a duration of time
		    ms.ModActionType_MUTE,
			ms.ModActionType_SUSPEND_ACCOUNT,
			ms.ModActionType_SUSPEND_RATED_GAMES,
			ms.ModActionType_SUSPEND_GAMES,*/
	ms.ModActionType_RESET_RATINGS.String():           resetRatings,
	ms.ModActionType_RESET_STATS.String():             resetStats,
	ms.ModActionType_RESET_STATS_AND_RATINGS.String(): resetStatsAndRatings,
}

func ActionExists(ctx context.Context, us user.Store, uuid string, actionType ms.ModActionType) error {
	currentActions, err := GetActions(ctx, us, uuid)
	if err != nil {
		return err
	}
	_, actionExists := currentActions[actionType.String()]
	if actionExists {
		return errors.New("this user is not permitted to perform this action")
	}
	return nil
}

func GetActions(ctx context.Context, us user.Store, uuid string) (map[string]*ms.ModAction, error) {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	// updateActions will initialize user.Actions.Current
	// so the return will not result in a nil pointer error
	updated, err := updateActions(user)
	if err != nil {
		return nil, err
	}

	if updated {
		err = us.Set(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user.Actions.Current, nil
}

func GetActionHistory(ctx context.Context, us user.Store, uuid string) ([]*ms.ModAction, error) {
	user, err := us.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}

	// updateActions will initialize user.Actions.History
	// so the return will not result in a nil pointer error
	updated, err := updateActions(user)
	if err != nil {
		return nil, err
	}

	if updated {
		err = us.Set(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user.Actions.History, nil
}

func ApplyActions(ctx context.Context, us user.Store, actions []*ms.ModAction) error {
	modUserId, err := sessionUserId(ctx, us)
	if err != nil {
		return err
	}
	for _, action := range actions {
		action.ModUserId = modUserId
		err := applyAction(ctx, us, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveActions(ctx context.Context, us user.Store, actions []*ms.ModAction) error {
	modUserId, err := sessionUserId(ctx, us)
	if err != nil {
		return err
	}
	for _, action := range actions {
		// This call will update the user actions
		// so that actions that have already expired
		// are not removed by a mod or admin
		_, err := GetActions(ctx, us, action.UserId)
		err = removeAction(ctx, us, action, modUserId)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateActions(user *entity.User) (bool, error) {

	instantiateActions(user)

	now := time.Now()
	updated := false
	for _, action := range user.Actions.Current {
		// This conversion will throw an error if action.EndTime
		// is nil. This means that the action is permanent
		// and should never be removed by this function.
		convertedEndTime, err := ptypes.Timestamp(action.EndTime)
		if err == nil && now.After(convertedEndTime) {
			removeCurrentAction(user, action.Type, true, "")
			updated = true
		}
	}

	return updated, nil
}

func removeAction(ctx context.Context, us user.Store, action *ms.ModAction, modUserId string) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}

	err = removeCurrentAction(user, action.Type, false, modUserId)
	if err != nil {
		return err
	}

	return us.Set(ctx, user)
}

func applyAction(ctx context.Context, us user.Store, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}
	action.StartTime = ptypes.TimestampNow()
	modActionFunc, actionExists := ModActionDispatching[action.Type.String()]
	if actionExists { // This ModAction is transient
		err := modActionFunc(ctx, us, action)
		if err != nil {
			return err
		}
		action.Duration = 0
		action.EndTime = action.StartTime
		action.RemovedTime = action.StartTime
		action.Expired = true
		err = addActionToHistory(user, action)
		if err != nil {
			return err
		}
	} else {
		if action.Duration < 0 {
			return fmt.Errorf("nontransient moderator action has a negative duration: %d", action.Duration)
		}
		// A Duration of 0 seconds for nontransient
		// actions is considered a permanent action
		if action.Duration == 0 {
			action.EndTime = nil
		} else {
			err = setEndTime(action)
			if err != nil {
				return err
			}
		}

		err = setCurrentAction(user, action)
		if err != nil {
			return err
		}
	}

	return us.Set(ctx, user)
}

func setEndTime(action *ms.ModAction) error {
	golangStartTime, err := ptypes.Timestamp(action.StartTime)
	if err != nil {
		return err
	}
	golangEndTime := golangStartTime.Add(time.Second * time.Duration(action.Duration))
	protoEndTime, err := ptypes.TimestampProto(golangEndTime)
	if err != nil {
		return err
	}
	action.EndTime = protoEndTime
	return nil
}

func addActionToHistory(user *entity.User, action *ms.ModAction) error {
	instantiateActions(user)
	user.Actions.History = append(user.Actions.History, action)
	return nil
}

func setCurrentAction(user *entity.User, action *ms.ModAction) error {
	instantiateActions(user)
	// Remove existing actions for this type
	_, actionExists := user.Actions.Current[action.Type.String()]
	if actionExists {
		err := removeCurrentAction(user, action.Type, false, action.ModUserId)
		if err != nil {
			return err
		}
	}
	user.Actions.Current[action.Type.String()] = action
	return nil
}

func removeCurrentAction(user *entity.User, actionType ms.ModActionType, expired bool, modUserId string) error {
	instantiateActions(user)
	existingCurrentAction, actionExists := user.Actions.Current[actionType.String()]
	if !actionExists {
		return fmt.Errorf("user does not have current action %s", actionType.String())
	}
	existingCurrentAction.Expired = expired
	if expired {
		existingCurrentAction.RemovedTime = existingCurrentAction.EndTime
	} else {
		currentTime, err := ptypes.TimestampProto(time.Now())
		if err != nil {
			return err
		}
		existingCurrentAction.RemovedTime = currentTime
	}
	existingCurrentAction.ModUserId = modUserId
	addActionToHistory(user, existingCurrentAction)
	delete(user.Actions.Current, actionType.String())
	return nil
}

func resetRatings(ctx context.Context, us user.Store, action *ms.ModAction) error {
	return us.ResetRatings(ctx, action.UserId)
}

func resetStats(ctx context.Context, us user.Store, action *ms.ModAction) error {
	return us.ResetStats(ctx, action.UserId)
}

func resetStatsAndRatings(ctx context.Context, us user.Store, action *ms.ModAction) error {
	err := us.ResetStats(ctx, action.UserId)
	if err != nil {
		return nil
	}
	return us.ResetRatings(ctx, action.UserId)
}

func instantiateActions(u *entity.User) {
	if u.Actions == nil {
		u.Actions = &entity.Actions{}
	}
	instantiateActionsCurrent(u)
	instantiateActionsHistory(u)
}

func instantiateActionsCurrent(u *entity.User) {
	if u.Actions.Current == nil {
		u.Actions.Current = make(map[string]*ms.ModAction)
	}
}

func instantiateActionsHistory(u *entity.User) {
	if u.Actions.History == nil {
		u.Actions.History = []*ms.ModAction{}
	}
}

func sessionUserId(ctx context.Context, us user.Store) (string, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return "", err
	}

	user, err := us.Get(ctx, sess.Username)
	if err != nil {
		return "", err
	}
	return user.UUID, nil
}
