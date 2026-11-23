package bus

import (
	"errors"
	"fmt"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestErrorMessageGivesARefusedChatBack(t *testing.T) {
	is := is.New(t)
	chat := &pb.ChatMessage{Channel: "chat.game.g1", Message: "gg"}
	em := errorMessage(&refusedChat{err: errors.New("you cannot send messages that quickly"), chat: chat})
	is.Equal(em.Message, "you cannot send messages that quickly")
	is.Equal(em.RefusedChat, chat)
	// still when wrapped on the way
	em = errorMessage(fmt.Errorf("handling: %w", &refusedChat{err: errors.New("muted"), chat: chat}))
	is.Equal(em.RefusedChat, chat)
	// any other error gives nothing back
	is.True(errorMessage(errors.New("game not found")).RefusedChat == nil)
}
