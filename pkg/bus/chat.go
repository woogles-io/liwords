package bus

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"
	"github.com/golang/protobuf/ptypes"

	"github.com/domino14/liwords/pkg/entity"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/rs/zerolog/log"
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

	err = mod.ActionExists(ctx, b.userStore, userID, false, []ms.ModActionType{ms.ModActionType_SUSPEND_ACCOUNT, ms.ModActionType_MUTE})
	if err != nil {
		return err
	}

	// Regulate chat only if the user is not privileged and the
	// chat is not a game chat
	if !sendingUser.IsAdmin && !sendingUser.IsMod && !sendingUser.IsDirector && !strings.HasPrefix(evt.Channel, "chat.game.") {
		if sendingUser.ChatInfo == nil || sendingUser.ChatInfo.LastMessageTime == nil {
			// Instantiate chat info
			sendingUser.ChatInfo = &entity.ChatInfo{LastMessage: ""}
			initialLastMessageTime, err := ptypes.TimestampProto(time.Unix(0, 0))
			if err != nil {
				return err
			}
			sendingUser.ChatInfo.LastMessageTime = initialLastMessageTime
		}
		// Check if the message is identical to the last one
		if evt.Message == sendingUser.ChatInfo.LastMessage {
			return errors.New("you cannot send a message identical to the previous one")
		}

		// Check if the cooldown is over
		lastMessageTime, err := ptypes.Timestamp(sendingUser.ChatInfo.LastMessageTime)
		if err != nil {
			return err
		}
		cooldownFinishedTime := lastMessageTime.Add(time.Second * time.Duration(MessageCooldownTime))
		now := time.Now()
		if cooldownFinishedTime.After(now) {
			return errors.New("you cannot send messages that quickly")
		}

		newLastMessageTime, err := ptypes.TimestampProto(now)
		if err != nil {
			return err
		}

		sendingUser.ChatInfo.LastMessageTime = newLastMessageTime
		sendingUser.ChatInfo.LastMessage = evt.Message
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
	}

	chatMessage, err := b.chatStore.AddChat(ctx, sendingUser.Username, userID, evt.Message, evt.Channel, userFriendlyChannelName)
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
