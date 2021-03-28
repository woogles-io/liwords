package mod

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/emailer"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

const ModActionEmailTemplate = `
Dear Woogles.io user,

The following action was taken against your account:

%s

If you think this was an error, please contact conduct@woogles.io.

The Woogles.io team
`

var DiscordTarget = "https://discord.com/api/webhooks/"

var ModActionEmailMap = map[ms.ModActionType]string{
	ms.ModActionType_MUTE:                    "Disable Chat",
	ms.ModActionType_SUSPEND_ACCOUNT:         "Account Suspension",
	ms.ModActionType_SUSPEND_RATED_GAMES:     "Disable Rated Games",
	ms.ModActionType_SUSPEND_GAMES:           "Disable Games",
	ms.ModActionType_RESET_RATINGS:           "Reset Ratings",
	ms.ModActionType_RESET_STATS:             "Reset Statistics",
	ms.ModActionType_RESET_STATS_AND_RATINGS: "Reset Ratings and Statistics",
}

var ModActionDispatching = map[string]func(context.Context, user.Store, user.ChatStore, *ms.ModAction) error{

	/*
		All types are listed here for clearness
		Types that are commented are not transient
		actions but are applied over a duration of time

		ms.ModActionType_MUTE,
		ms.ModActionType_SUSPEND_ACCOUNT,
		ms.ModActionType_SUSPEND_RATED_GAMES,
		ms.ModActionType_SUSPEND_GAMES,
	*/
	ms.ModActionType_RESET_RATINGS.String():           resetRatings,
	ms.ModActionType_RESET_STATS.String():             resetStats,
	ms.ModActionType_RESET_STATS_AND_RATINGS.String(): resetStatsAndRatings,
	ms.ModActionType_REMOVE_CHAT.String():             removeChat,
}

var ModActionTextMap = map[ms.ModActionType]string{
	ms.ModActionType_MUTE:                "chatting",
	ms.ModActionType_SUSPEND_ACCOUNT:     "logging in",
	ms.ModActionType_SUSPEND_RATED_GAMES: "playing rated games",
	ms.ModActionType_SUSPEND_GAMES:       "playing games",
}

func ActionExists(ctx context.Context, us user.Store, uuid string, forceInsistLogout bool, actionTypes []ms.ModActionType) error {
	currentActions, err := GetActions(ctx, us, uuid)
	if err != nil {
		return err
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
			golangEndTime, err := ptypes.Timestamp(action.EndTime)
			if err != nil {
				return err
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
			return fmt.Errorf("Action %s is unmapped. Please report this to the Woogles team immediately.", relevantActionType.String())
		}
		if forceInsistLogout || (numberOfActionsChecked > 1 && relevantActionType == ms.ModActionType_SUSPEND_ACCOUNT) {
			disabledError = errors.New("Whoops, something went wrong! Please log out and try logging in again.")
		} else if permaban {
			disabledError = fmt.Errorf("You are banned from %s. If you think this is an error, contact conduct@woogles.io.", actionText)
		} else if latestTime.After(now) {
			year, month, day := latestTime.Date()
			disabledError = fmt.Errorf("You are suspended from %s until %v %v, %v.", actionText, month, day, year)
		} else {
			return errors.New("Encountered an error while checking available user actions. Please report this to the Woogles team immediately.")
		}
	}
	return disabledError
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

func ApplyActions(ctx context.Context, us user.Store, cs user.ChatStore,
	mailgunKey string, discordToken string, actions []*ms.ModAction) error {
	applierUserId, err := sessionUserId(ctx, us)
	if err != nil {
		return err
	}
	for _, action := range actions {
		action.ApplierUserId = applierUserId
		err := applyAction(ctx, us, cs, mailgunKey, discordToken, action)
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

func applyAction(ctx context.Context, us user.Store, cs user.ChatStore,
	mailgunKey string, discordToken string, action *ms.ModAction) error {
	user, err := us.GetByUUID(ctx, action.UserId)
	if err != nil {
		return err
	}
	action.StartTime = ptypes.TimestampNow()
	modActionFunc, actionExists := ModActionDispatching[action.Type.String()]
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

	actionEmailText, ok := ModActionEmailMap[action.Type]
	if ok {
		if mailgunKey != "" {
			_, err := emailer.SendSimpleMessage(mailgunKey, user.Email, fmt.Sprintf("Account Taken Against %s", user.Username),
				fmt.Sprintf(ModActionEmailTemplate, actionEmailText))
			if err != nil {
				// Errors should not be fatal, just log them
				log.Err(err).Str("userID", user.UUID).Msg("mod-action-user-email")
			}
		}
		if discordToken != "" {
			modUser, err := us.GetByUUID(ctx, action.ApplierUserId)
			if err != nil {
				log.Err(err).Str("userID", user.UUID).Msg("mod-action-applier")
			} else {
				message := fmt.Sprintf("Action %s was applied to user %s by moderator %s.", actionEmailText, user.Username, modUser.Username)
				if !actionExists { // Action is non-transient
					if action.Duration == 0 {
						message += " This action is permanent."
					} else if action.EndTime == nil {
						log.Err(err).Str("userID", user.UUID).Msg("mod-action-endtime-nil")
					} else {
						golangActionEndTime, err := ptypes.Timestamp(action.EndTime)
						if err != nil {
							log.Err(err).Str("error", err.Error()).Msg("mod-action-endtime-conversion")
						} else {
							message += fmt.Sprintf(" This action will expire on %s.", golangActionEndTime.UTC().Format(time.UnixDate))
						}
					}
				}
				requestBody, err := json.Marshal(map[string]string{"content": message})

				// Errors should not be fatal, just log them
				if err != nil {
					log.Err(err).Str("error", err.Error()).Msg("mod-action-discord-notification-marshal")
				} else {
					resp, err := http.Post(DiscordTarget+discordToken, "application/json", bytes.NewBuffer(requestBody))
					// Errors should not be fatal, just log them
					if err != nil {
						log.Err(err).Str("error", err.Error()).Msg("mod-action-discord-notification-post-error")
					} else if resp.StatusCode != 204 { // No Content
						// We do not expect any other response
						log.Err(err).Str("status", resp.Status).Msg("mod-action-discord-notification-post-bad-response")
					}
				}
			}
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
