package ipc

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/nats-io/nats.go"
)

type EventAudienceType string

const EventTransport = "pb"

const (
	AudGame       EventAudienceType = "game"
	AudGameTV                       = "gametv"
	AudUser                         = "user"
	AudLobby                        = "lobby"
	AudTournament                   = "tournament"
	// AudChannel is used for a general channel.
	AudChannel = "channel"
)

type Audience struct {
	tp     EventAudienceType
	suffix string
}

func (a Audience) recipient(mt pb.MessageType) string {
	// Create a NATS topic for this audience.
	// It looks something like:
	// user.pb.17.abcdefgh
	// where 17 is the type of the message that is being sent.
	var topic strings.Builder
	topic.WriteString(string(a.tp))
	topic.WriteString(".")
	topic.WriteString(EventTransport)
	topic.WriteString(".")
	topic.WriteString(strconv.Itoa(int(mt)))
	if a.suffix != "" {
		topic.WriteString(".")
		topic.WriteString(a.suffix)
	}
	return topic.String()
}

// EventWrapper wraps some useful things around messages.
type EventWrapper struct {
	Type  pb.MessageType
	Event proto.Message

	audience []Audience
}

func WrapEvent(event proto.Message, messageType pb.MessageType) *EventWrapper {
	return &EventWrapper{
		Type:  messageType,
		Event: event,
	}
}

// AddAudience sets the audience(s) for this event. It is in the form of a NATS
// channel name. This is not required to be set in order to deliver a message,
// but certain functions will use it in the gameplay/entity module.
func (e *EventWrapper) AddAudience(audType EventAudienceType, suffix string) {
	if e.audience == nil {
		e.audience = []Audience{}
	}
	e.audience = append(e.audience, Audience{tp: audType, suffix: suffix})

}

// // SetAudience sets a single audience in string format.
// func (e *EventWrapper) SetAudience(audType Eve) {
// 	e.audience = []string{a}
// }

// Serialize just writes the event out as binary.
func (e *EventWrapper) Serialize() ([]byte, error) {
	return proto.Marshal(e.Event)
}

func (e *EventWrapper) PublishToNATS(natsconn *nats.Conn) error {
	data, err := e.Serialize()
	if err != nil {
		return err
	}
	for _, r := range e.audience {
		// Generate topic name from recipient.
		natsconn.Publish(r.recipient(e.Type), data)
	}
	return nil
}
