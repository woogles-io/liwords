package mod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	"github.com/domino14/liwords/pkg/utilities"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var ModActionDispatching = map[ms.ModActionType]func(context.Context, user.Store, user.ChatStore, *ms.ModAction) error{

	/*
		All types are listed here for clearness
		Types that are commented are not transient
		actions but are applied over a duration of time

		ms.ModActionType_MUTE,
		ms.ModActionType_SUSPEND_ACCOUNT,
		ms.ModActionType_SUSPEND_RATED_GAMES,
		ms.ModActionType_SUSPEND_GAMES,
		ms.ModActionType_SUSPEND_TOURNAMENT_ROOM,
	*/
	ms.ModActionType_RESET_RATINGS:           resetRatings,
	ms.ModActionType_RESET_STATS:             resetStats,
	ms.ModActionType_RESET_STATS_AND_RATINGS: resetStatsAndRatings,
	ms.ModActionType_REMOVE_CHAT:             removeChat,
	ms.ModActionType_DELETE_ACCOUNT:          deleteAccount,
}

var ModActionTextMap = map[ms.ModActionType]string{
	ms.ModActionType_MUTE:                    "chatting",
	ms.ModActionType_MUTE_IN_CHANNEL:         "chatting in this channel",
	ms.ModActionType_SUSPEND_ACCOUNT:         "logging in",
	ms.ModActionType_SUSPEND_RATED_GAMES:     "playing rated games",
	ms.ModActionType_SUSPEND_GAMES:           "playing games",
	ms.ModActionType_SUSPEND_TOURNAMENT_ROOM: "entering this room",
}

var RemovalDuration = 60

func ActionExists(ctx context.Context, us user.Store, uuid string, forceInsistLogout bool, actionTypes []ms.ModActionType, actionParams ...string) (bool, error) {
	currentActions, err := GetActions(ctx, us, uuid)
	if err != nil {
		return false, err
	}

	// We want to show the user longest ban out of all the actions,
	// so we want the time furthest in the future. Initialize the latestTime
	// to be the unix epoch. Any valid times that come from
	// actions will be later than this time.
	now := time.Now()
	latestTime := time.Unix(0, 0)
	permaban := false
	actionExists := false
	secondTimeIsLater := false
	var relevantActionType ms.ModActionType

	for _, actionType := range actionTypes {
		action, thisActionExists := currentActions[actionType.String()]
		if thisActionExists {
			if action.Type == ms.ModActionType_MUTE_IN_CHANNEL {
				// check extra parameters
				if len(actionParams) > 0 {
					actionParams
				}
			}

			if !actionExists {
				actionExists = true
			}
			if action.EndTime == nil {
				relevantActionType = actionType
				permaban = true
				break
			}
			golangEndTime, err := ptypes.Timestamp(action.EndTime)
			if err != nil {
				return false, err
			}
			latestTime, secondTimeIsLater = getLaterTime(latestTime, golangEndTime)
			if secondTimeIsLater {
				relevantActionType = actionType
			}
		}
	}

	var disabledError error = nil

	if actionExists {
		numberOfActionsChecked := len(actionTypes)
		actionText, ok := ModActionTextMap[relevantActionType]
		if !ok {
			return false, fmt.Errorf("Action %s is unmapped. Please report this to the Woogles team immediately.", relevantActionType.String())
		}
		if forceInsistLogout || (numberOfActionsChecked > 1 && relevantActionType == ms.ModActionType_SUSPEND_ACCOUNT) {
			disabledError = errors.New("Whoops, something went wrong! Please log out and try logging in again.")
		} else if permaban {
			if relevantActionType == ms.ModActionType_SUSPEND_ACCOUNT {
				disabledError = errors.New("This account has been deactivated. If you think this is an error, contact conduct@woogles.io.")
			} else {
				disabledError = fmt.Errorf("You are banned from %s. If you think this is an error, contact conduct@woogles.io.", actionText)
			}
		} else if latestTime.After(now) {
			year, month, day := latestTime.Date()
			disabledError = fmt.Errorf("You are suspended from %s until %v %v, %v.", actionText, month, day, year)
		} else {
			return false, errors.New("Encountered an error while checking available user actions. Please report this to the Woogles team immediately.")
		}
	}
	return permaban, disabledError
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

func ApplyActions(ctx context.Context, us user.Store, cs user.ChatStore, actions []*ms.ModAction) error {
	applierUserId, err := sessionUserId(ctx, us)
	if err != nil {
		return err
	}
	for _, action := range actions {
		if action.Type == ms.ModActionType_DELETE_ACCOUNT {
			// The DELETE_ACCOUNT action erases the profile,
			// but we still need to permanently ban
			err := applyAction(ctx, us, cs, &ms.ModAction{
				UserId:        action.UserId,
				Type:          ms.ModActionType_SUSPEND_ACCOUNT,
				Duration:      0,
				ApplierUserId: applierUserId,
				EmailType:     ms.EmailType_DELETION,
				Note:          "AUTOGENERATED ACTION: " + action.Note})
			if err != nil {
				return err
			}
		}
		action.ApplierUserId = applierUserId
		err := applyAction(ctx, us, cs, action)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveActions(ctx context.Context, us user.Store, actions []*ms.ModAction) error {
	removerUserId, err := sessionUserId(ctx, us)
	if err != nil {
		return err
	}
	for _, action := range actions {
		// This call will update the user actions
		// so that actions that have already expired
		// are not removed by a mod or admin
		_, err := GetActions(ctx, us, action.UserId)
		if err != nil {
			return err
		}
		err = removeAction(ctx, us, action, removerUserId)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsRemoval(action *ms.ModAction) bool {
	return action.Duration != 0 && action.Duration <= int32(RemovalDuration)
}

func IsCensorable(ctx context.Context, us user.Store, uuid string) bool {
	// Don't censor if already censored
	if uuid == utilities.CensoredUsername ||
		uuid == utilities.AnotherCensoredUsername {
		return false
	}
	permaban, _ := ActionExists(ctx, us, uuid, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	return permaban
}

func censorPlayerInHistory(hist *macondopb.GameHistory, playerIndex int, bothCensorable bool) {
	uncensoredNickname := hist.Players[playerIndex].Nickname
	censoredUsername := utilities.CensoredUsername
	if bothCensorable && playerIndex == 1 {
		censoredUsername = utilities.AnotherCensoredUsername
	}
	hist.Players[playerIndex].UserId = censoredUsername
	hist.Players[playerIndex].RealName = censoredUsername
	hist.Players[playerIndex].Nickname = censoredUsername
	for idx, _ := range hist.Events {
		if hist.Events[idx].Nickname == uncensoredNickname {
			hist.Events[idx].Nickname = censoredUsername
		}
	}
}

func CensorHistory(ctx context.Context, us user.Store, hist *macondopb.GameHistory) *macondopb.GameHistory {
	playerOne := hist.Players[0].UserId
	playerTwo := hist.Players[1].UserId

	playerOneCensorable := IsCensorable(ctx, us, playerOne)
	playerTwoCensorable := IsCensorable(ctx, us, playerTwo)
	bothCensorable := playerOneCensorable && playerTwoCensorable

	if !playerOneCensorable && !playerTwoCensorable {
		return hist
	}

	censoredHistory := proto.Clone(hist).(*macondopb.GameHistory)

	if playerOneCensorable {
		censorPlayerInHistory(censoredHistory, 0, bothCensorable)
	}

	if playerTwoCensorable {
		censorPlayerInHistory(censoredHistory, 1, bothCensorable)
	}
	return censoredHistory
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
			removeCurrentAction(user, action.Type, "")
			updated = true
		}
	}

	return updated, nil
}

func removeAction(ctx context.Context, us user.Store, action *ms.ModAction, removerUserId string) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}

	err = removeCurrentAction(user, action.Type, removerUserId)
	if err != nil {
		return err
	}

	return us.Set(ctx, user)
}

func applyAction(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}
	action.StartTime = ptypes.TimestampNow()
	modActionFunc, actionExists := ModActionDispatching[action.Type]
	if actionExists { // This ModAction is transient
		err := modActionFunc(ctx, us, cs, action)
		if err != nil {
			return err
		}
		action.Duration = 0
		action.EndTime = action.StartTime
		action.RemovedTime = action.StartTime
		action.RemoverUserId = ""
		err = addActionToHistory(user, action)
		if err != nil {
			return err
		}
	} else {
		err = setCurrentAction(user, action)
		if err != nil {
			return err
		}
	}

	err = us.Set(ctx, user)
	if err != nil {
		return err
	}
	notify(ctx, us, user, action, "")
	return nil
}

func addActionToHistory(user *entity.User, action *ms.ModAction) error {
	instantiateActions(user)
	user.Actions.History = append(user.Actions.History, action)
	return nil
}

func setCurrentAction(user *entity.User, action *ms.ModAction) error {
	if action.Duration < 0 {
		return fmt.Errorf("nontransient moderator action has a negative duration: %d", action.Duration)
	}
	// A Duration of 0 seconds for nontransient
	// actions is considered a permanent action
	if action.Duration == 0 {
		action.EndTime = nil
	} else {
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
	}

	instantiateActions(user)
	// Remove existing actions for this type
	_, actionExists := user.Actions.Current[action.Type.String()]
	if actionExists {
		err := removeCurrentAction(user, action.Type, action.ApplierUserId)
		if err != nil {
			return err
		}
	}
	user.Actions.Current[action.Type.String()] = action
	return nil
}

func removeCurrentAction(user *entity.User, actionType ms.ModActionType, removerUserId string) error {
	instantiateActions(user)

	existingCurrentAction, actionExists := user.Actions.Current[actionType.String()]
	if !actionExists {
		return fmt.Errorf("user does not have current action %s", actionType.String())
	}

	existingCurrentAction.RemoverUserId = removerUserId

	// If this action has expired, the removed time is the same
	// as the end time. An expired action in this function is
	// indicated by an empty string for removerUserId
	if removerUserId == "" {
		existingCurrentAction.RemovedTime = existingCurrentAction.EndTime
	} else {
		currentTime, err := ptypes.TimestampProto(time.Now())
		if err != nil {
			return err
		}
		existingCurrentAction.RemovedTime = currentTime
	}

	addActionToHistory(user, existingCurrentAction)
	delete(user.Actions.Current, actionType.String())
	return nil
}

func getLaterTime(t1 time.Time, t2 time.Time) (time.Time, bool) {
	laterTime := t1
	secondTimeIsLater := false
	if t2.After(t1) {
		laterTime = t2
		secondTimeIsLater = true
	}
	return laterTime, secondTimeIsLater
}

func resetRatings(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	return us.ResetRatings(ctx, action.UserId)
}

func resetStats(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	return us.ResetStats(ctx, action.UserId)
}

func resetStatsAndRatings(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	err := us.ResetStats(ctx, action.UserId)
	if err != nil {
		return nil
	}
	return us.ResetRatings(ctx, action.UserId)
}

func removeChat(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	chat, err := cs.GetChat(ctx, action.Channel, action.MessageId)
	if err != nil {
		return err
	}

	err = cs.DeleteChat(ctx, action.Channel, action.MessageId)
	if err != nil {
		return err
	}
	action.ChatText = chat.Message

	// Send a message via pubsub
	evtChan := cs.EventChan()
	if evtChan != nil {
		evt := &pb.ChatMessageDeleted{
			Channel: action.Channel,
			Id:      action.MessageId,
		}
		wrapped := entity.WrapEvent(evt, pb.MessageType_CHAT_MESSAGE_DELETED)
		wrapped.SetAudience(action.Channel)
		evtChan <- wrapped
	}

	return nil
}

func deleteAccount(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	return us.ResetProfile(ctx, action.UserId)
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
