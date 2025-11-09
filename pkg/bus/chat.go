package bus

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	userservices "github.com/woogles-io/liwords/pkg/user/services"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

// chat-related functionality should be here. Chat should be mostly ephemeral,
// but will use Redis to keep a short history of previous chats in every channel.

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

	userFriendlyChannelName := ""
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
		first, second := sendingUser.Username, recUser.Username
		if first > second {
			first, second = second, first
		}
		userFriendlyChannelName = "pm:" + first + ":" + second
	} else if strings.HasPrefix(evt.Channel, "chat.tournament.") {
		tid := strings.TrimPrefix(evt.Channel, "chat.tournament.")
		if len(tid) == 0 {
			return errors.New("nonexistent tournament")
		}
		t, err := b.stores.TournamentStore.Get(ctx, tid)
		if err != nil {
			return err
		}
		userFriendlyChannelName = "tournament:" + t.Name
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
	}

	chatMessage, err := b.stores.ChatStore.AddChat(ctx, sendingUser.Username, userID, evt.Message, evt.Channel, userFriendlyChannelName, regulateChat)
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
