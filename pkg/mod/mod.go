package mod

import (
	"context"
	"fmt"
	"time"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
	*/
	ms.ModActionType_RESET_RATINGS:           resetRatings,
	ms.ModActionType_RESET_STATS:             resetStats,
	ms.ModActionType_RESET_STATS_AND_RATINGS: resetStatsAndRatings,
	ms.ModActionType_REMOVE_CHAT:             removeChat,
	ms.ModActionType_DELETE_ACCOUNT:          deleteAccount,
}

var ModActionTextMap = map[ms.ModActionType]string{
	ms.ModActionType_MUTE:                "chatting",
	ms.ModActionType_SUSPEND_ACCOUNT:     "logging in",
	ms.ModActionType_SUSPEND_RATED_GAMES: "playing rated games",
	ms.ModActionType_SUSPEND_GAMES:       "playing games",
}

var RemovalDuration = 60

type UserModeratedError struct {
	description string
}

func (u *UserModeratedError) Error() string {
	return u.description
}

// DB version of the ActionExists function above for testing purposes
// This can be deleted once the above function uses the DB to get the actions.
func ActionExists(ctx context.Context, userStore user.Store, uuid string, forceInsistLogout bool, actionTypes []ms.ModActionType) (bool, error) {
	currentActions, err := userStore.GetActions(ctx, uuid)
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
			if !actionExists {
				actionExists = true
			}
			if action.EndTime == nil {
				relevantActionType = actionType
				permaban = true
				break
			}
			golangEndTime := action.EndTime.AsTime()
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
			return false, &UserModeratedError{fmt.Sprintf("Action %s is unmapped. Please report this to the Woogles team immediately.", relevantActionType.String())}
		}
		if forceInsistLogout || (numberOfActionsChecked > 1 && relevantActionType == ms.ModActionType_SUSPEND_ACCOUNT) {
			disabledError = &UserModeratedError{"Whoops, something went wrong! Please log out and try logging in again."}
		} else if permaban {
			if relevantActionType == ms.ModActionType_SUSPEND_ACCOUNT {
				disabledError = &UserModeratedError{"This account has been deactivated. If you think this is an error, contact conduct@woogles.io."}
			} else {
				disabledError = &UserModeratedError{fmt.Sprintf("You are banned from %s. If you think this is an error, contact conduct@woogles.io.", actionText)}
			}
		} else if latestTime.After(now) {
			year, month, day := latestTime.Date()
			disabledError = &UserModeratedError{fmt.Sprintf("You are suspended from %s until %v %v, %v.", actionText, month, day, year)}
		} else {
			return false, &UserModeratedError{"Encountered an error while checking available user actions. Please report this to the Woogles team immediately."}
		}
	}
	return permaban, disabledError
}

func GetActions(ctx context.Context, us user.Store, uuid string) (map[string]*ms.ModAction, error) {
	currentActions, err := us.GetActions(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return currentActions, nil
}

func GetActionHistory(ctx context.Context, us user.Store, uuid string) ([]*ms.ModAction, error) {
	history, err := us.GetActionHistory(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func ApplyActions(ctx context.Context, us user.Store, cs user.ChatStore, applierUserId string, actions []*ms.ModAction) error {
	actionsToApply := []*ms.ModAction{}
	for _, action := range actions {
		if action.Type == ms.ModActionType_DELETE_ACCOUNT {
			// The DELETE_ACCOUNT action erases the profile,
			// but we still need to permanently ban
			suspendAccountAction := &ms.ModAction{
				UserId:        action.UserId,
				Type:          ms.ModActionType_SUSPEND_ACCOUNT,
				Duration:      0,
				ApplierUserId: applierUserId,
				EmailType:     ms.EmailType_DELETION,
				Note:          "AUTOGENERATED ACTION: " + action.Note}
			err := prepareAction(ctx, us, cs, suspendAccountAction)
			if err != nil {
				return err
			}
			actionsToApply = append(actionsToApply, suspendAccountAction)
		}
		action.ApplierUserId = applierUserId
		err := prepareAction(ctx, us, cs, action)
		if err != nil {
			return err
		}
		actionsToApply = append(actionsToApply, action)
	}

	return us.ApplyActions(ctx, actionsToApply)
}

func prepareAction(ctx context.Context, us user.Store, cs user.ChatStore, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}
	action.StartTime = timestamppb.Now()
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
	} else {
		if action.Duration < 0 {
			return fmt.Errorf("nontransient moderator action has a negative duration: %d", action.Duration)
		}
		// A Duration of 0 seconds for nontransient
		// actions is considered a permanent action
		if action.Duration == 0 {
			action.EndTime = nil
		} else {
			golangStartTime := action.StartTime.AsTime()
			golangEndTime := golangStartTime.Add(time.Second * time.Duration(action.Duration))
			protoEndTime := timestamppb.New(golangEndTime)
			action.EndTime = protoEndTime
		}
	}
	sendNotification(ctx, us, user, action, "")
	return nil
}

func RemoveActions(ctx context.Context, userStore user.Store, removerUserId string, actions []*ms.ModAction) error {
	return userStore.RemoveActions(ctx, actions)
}

func IsRemoval(action *ms.ModAction) bool {
	return action.Duration != 0 && action.Duration <= int32(RemovalDuration)
}

func IsCensorable(ctx context.Context, userStore user.Store, uuid string) bool {
	// Don't censor if already censored
	if uuid == utilities.CensoredUsername ||
		uuid == utilities.AnotherCensoredUsername {
		return false
	}
	permaban, _ := ActionExists(ctx, userStore, uuid, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT})
	return permaban
}

// IsCensorableFromCache checks if a user should be censored using pre-fetched action data
// This avoids N+1 query problems when checking multiple users
func IsCensorableFromCache(uuid string, userActions map[string]map[string]*ms.ModAction) bool {
	// Don't censor if already censored
	if uuid == utilities.CensoredUsername ||
		uuid == utilities.AnotherCensoredUsername {
		return false
	}

	// Check if user has permanent account suspension in cached actions
	actions, exists := userActions[uuid]
	if !exists {
		return false
	}

	_, permaban := actions[ms.ModActionType_SUSPEND_ACCOUNT.String()]
	return permaban
}

func censorPlayerInHistory(hist *macondopb.GameHistory, playerIndex int, bothCensorable bool) {
	censoredUsername := utilities.CensoredUsername
	if bothCensorable && playerIndex == 1 {
		censoredUsername = utilities.AnotherCensoredUsername
	}
	hist.Players[playerIndex].UserId = censoredUsername
	hist.Players[playerIndex].RealName = censoredUsername
	hist.Players[playerIndex].Nickname = censoredUsername
}

func CensorHistory(ctx context.Context, userStore user.Store, hist *macondopb.GameHistory) *macondopb.GameHistory {
	playerOne := hist.Players[0].UserId
	playerTwo := hist.Players[1].UserId

	playerOneCensorable := IsCensorable(ctx, userStore, playerOne)
	playerTwoCensorable := IsCensorable(ctx, userStore, playerTwo)
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
