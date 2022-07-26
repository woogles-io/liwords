package bus

import (
	"context"
	"errors"
	"strings"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"github.com/rs/zerolog/log"
)

var (
	errNoPrivateChatAllowed = errors.New("you are in silent mode, so can not chat")

	errReceiverNoPrivateChat = errors.New("this user does not receive chats")
)

// chat-related functionality should be here. Chat should be mostly ephemeral,
// but will use Redis to keep a short history of previous chats in every channel.

func (b *Bus) chat(ctx context.Context, userID string, evt *pb.ChatMessage) error {
	if len(evt.Message) > MaxMessageLength {
		return errors.New("message-too-long")
	}

	// All channels should be of the form:
	// chat.a[.b]
	// e.g. chat.gametv.abcdef, chat.pm.user1_user2, chat.lobby, chat.tournament.weto

	sendingUser, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = mod.ActionExists(ctx, b.userStore, userID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT, ms.ModActionType_MUTE})
	if err != nil {
		return err
	}
	// check if the sending user is a child or has silent mode on.
	isChild := sendingUser.IsChild() == pb.ChildStatus_UNKNOWN ||
		sendingUser.IsChild() == pb.ChildStatus_CHILD
	// Do not allow chat in a private channel

	disallowPrivate := isChild || sendingUser.Profile.SilentMode
	privateChannel := strings.HasPrefix(evt.Channel, "chat.pm.") ||
		strings.HasPrefix(evt.Channel, "chat.game.")

	// Regulate chat only if the user is not privileged and the
	// chat is not a private chat or a game chat
	regulateChat := !(sendingUser.IsAdmin ||
		sendingUser.IsMod ||
		sendingUser.IsDirector ||
		privateChannel)

	if privateChannel && disallowPrivate {
		return errNoPrivateChatAllowed
	}

	userFriendlyChannelName := ""
	if strings.HasPrefix(evt.Channel, "chat.pm.") {
		receiver, err := user.ChatChannelReceiver(userID, evt.Channel)
		if err != nil {
			return err
		}
		recUser, err := b.userStore.GetByUUID(ctx, receiver)
		if err != nil {
			return err
		}
		if recUser.IsChild() == pb.ChildStatus_UNKNOWN ||
			recUser.IsChild() == pb.ChildStatus_CHILD || recUser.Profile.SilentMode {
			return errReceiverNoPrivateChat
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
		t, err := b.tournamentStore.Get(ctx, tid)
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

	chatMessage, err := b.chatStore.AddChat(ctx, sendingUser.Username, userID, evt.Message, evt.Channel, userFriendlyChannelName, regulateChat)
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
