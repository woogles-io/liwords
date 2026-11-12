package bus

import (
	"testing"

	"github.com/matryer/is"
)

// The event-stream topics reach every server; the rest are handled once.
func TestQueueGroups(t *testing.T) {
	is := is.New(t)
	is.Equal(queueGroup("user.>"), "")
	is.Equal(queueGroup("game.>"), "")
	is.Equal(queueGroup("ipc.pb.>"), "pbworkers")
	is.Equal(queueGroup("bot.publish_event.>"), "pbworkers")
	is.Equal(queueGroup("ipc.request.>"), "requestworkers")
}
