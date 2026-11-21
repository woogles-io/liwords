package bus

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	userservices "github.com/woogles-io/liwords/pkg/user/services"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// chat-related functionality should be here. Messages are stored by the chat
// store (Postgres) and published on NATS for the socket server to deliver.

// refusedChat is a chat message that was not sent, with why. Its sender is
// told why, and given the message back, to send again.
type refusedChat struct {
	err  error
	chat *pb.ChatMessage
}

func (r *refusedChat) Error() string { return r.err.Error() }
func (r *refusedChat) Unwrap() error { return r.err }

// errorMessage is what the sender of a request that failed is told: why, and
// for a chat message refused, the message as it was sent.
func errorMessage(err error) *pb.ErrorMessage {
	em := &pb.ErrorMessage{Message: err.Error()}
	var refused *refusedChat
	if errors.As(err, &refused) {
		em.RefusedChat = refused.chat
	}
	return em
}

func (b *Bus) chat(ctx context.Context, userID string, evt *pb.ChatMessage) error {
	if utf8.RuneCountInString(evt.Message) > MaxMessageLength {
		return errors.New("message-too-long")
	}

	// All channels should be of the form:
	// chat.a[.b]
	// e.g. chat.gametv.abcdef, chat.pm.user1_user2, chat.lobby, chat.tournament.weto

	sendingUser, err := b.stores.UserStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = mod.ActionExists(ctx, b.stores.UserStore, userID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT, ms.ModActionType_MUTE})
	if err != nil {
		return err
	}
	privilegedUser, err := b.stores.Queries.HasPermission(ctx, models.HasPermissionParams{
		UserID:     int32(sendingUser.ID),
		Permission: string(rbac.CanModerateUsers),
	})
	if err != nil {
		return err
	}
	// Regulate chat only if the user is not privileged and the
	// chat is not a private chat or a game chat
	regulateChat := !(privilegedUser ||
		strings.HasPrefix(evt.Channel, "chat.pm.") ||
		strings.HasPrefix(evt.Channel, "chat.game."))

	if strings.HasPrefix(evt.Channel, "chat.pm.") {
		receiver, err := userservices.ChatChannelReceiver(userID, evt.Channel)
		if err != nil {
			return err
		}
		recUser, err := b.stores.UserStore.GetByUUID(ctx, receiver)
		if err != nil {
			return err
		}
		block, err := b.blockExists(ctx, recUser, sendingUser)
		if err != nil {
			return err
		}
		if block == 0 {
			// receiver is blocking sender
			return errors.New("your message could not be delivered to " + recUser.Username)
		} else if block == 1 {
			return errors.New("you cannot send messages to people you are blocking")
		}
	} else if strings.HasPrefix(evt.Channel, "chat.tournament.") {
		tid := strings.TrimPrefix(evt.Channel, "chat.tournament.")
		if len(tid) == 0 {
			return errors.New("nonexistent tournament")
		}
		t, err := b.stores.TournamentStore.GetHeader(ctx, tid)
		if err != nil {
			return err
		}
		if regulateChat {
			for _, director := range t.Directors.Persons {
				uuid := director.Id
				colon := strings.IndexByte(uuid, ':')
				if colon >= 0 {
					uuid = uuid[:colon]
				}
				if userID == uuid {
					regulateChat = false
					break
				}
			}
		}
	} else if strings.HasPrefix(evt.Channel, "chat.league.") {
		leagueID := strings.TrimPrefix(evt.Channel, "chat.league.")
		if len(leagueID) == 0 {
			return errors.New("nonexistent league")
		}
		leagueUUID, err := uuid.Parse(leagueID)
		if err != nil {
			return err
		}
		if _, err := b.stores.LeagueStore.GetLeagueByUUID(ctx, leagueUUID); err != nil {
			return err
		}
		// TODO: League managers could bypass chat regulation similar to tournament directors
		// This would require adding a GetLeagueManagers query or including managers in the league model
	}

	chatMessage, err := b.stores.ChatStore.AddChat(ctx, sendingUser.Username, userID, evt.Message, evt.Channel, regulateChat)
	if err != nil {
		return err
	}
	if sendingUser.Profile != nil {
		chatMessage.CountryCode = sendingUser.Profile.CountryCode
		chatMessage.AvatarUrl = sendingUser.AvatarUrl()
	} else {
		// not sure if it can be nil, but we don't want to crash if that happens
		log.Warn().Interface("chat-message", chatMessage).Msg("chat-no-profile")
	}
	toSend := entity.WrapEvent(chatMessage, pb.MessageType_CHAT_MESSAGE)
	data, err := toSend.Serialize()

	if err != nil {
		return err
	}

	log.Debug().Interface("chat-message", chatMessage).Msg("publish-chat")
	return b.natsconn.Publish(evt.Channel, data)
}
