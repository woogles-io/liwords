package bus

import (
	"context"
	"errors"
	"strings"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"

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

	// XXX: temporary migration code. Remove once frontends stop sending
	// on the wrong channel.
	// All channels should be of the form:
	// chat.a[.b]
	// e.g. chat.gametv.abcdef, chat.pm.user1_user2, chat.lobby, chat.tournament.weto
	if !strings.HasPrefix(evt.Channel, "chat.") {
		evt.Channel = "chat." + evt.Channel
		// Remove the .chat from the end -- legacy channel name.
		if strings.HasSuffix(evt.Channel, ".chat") {
			evt.Channel = strings.TrimSuffix(evt.Channel, ".chat")
		}
	}

	sendingUser, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	err = mod.ActionExists(ctx, b.userStore, userID, []ms.ModActionType{ms.ModActionType_MUTE, ms.ModActionType_SUSPEND_ACCOUNT})
	if err != nil {
		return err
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

	ts, err := b.chatStore.AddChat(ctx, sendingUser.Username, userID, evt.Message, evt.Channel, userFriendlyChannelName)

	chatMessage := &pb.ChatMessage{
		Username:  sendingUser.Username,
		UserId:    userID,
		Channel:   evt.Channel, // this info might be redundant
		Message:   evt.Message,
		Timestamp: ts,
	}

	toSend := entity.WrapEvent(chatMessage, pb.MessageType_CHAT_MESSAGE)
	data, err := toSend.Serialize()

	if err != nil {
		return err
	}

	log.Debug().Interface("chat-message", chatMessage).Msg("publish-chat")
	return b.natsconn.Publish(evt.Channel, data)
}
