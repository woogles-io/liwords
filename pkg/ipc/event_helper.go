package ipc

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog/log"
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
	tp EventAudienceType
	// receiver is the actual audience for this. for example,
	// tp can be of type AudUser, and receiver would be the user's ID.
	receiver string
}

func (a Audience) RecipientTopic(mt pb.MessageType) string {
	// Create a NATS topic for this audience.
	// It looks something like:
	// user.pb.17.abcdefgh
	// where 17 is the type of the message that is being sent.

	topic := fmt.Sprintf("%s.%s.%d", string(a.tp), EventTransport, int(mt))

	if a.receiver != "" {
		topic = fmt.Sprintf("%s.%s", topic, a.receiver)
	}
	return topic
}

func (a Audience) Receiver() string {
	return a.receiver
}

func (a Audience) Type() EventAudienceType {
	return a.tp
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

func (e *EventWrapper) Audience() []Audience {
	return e.audience
}

// AddAudience sets the audience(s) for this event.
func (e *EventWrapper) AddAudience(audType EventAudienceType, receiver string) {
	if e.audience == nil {
		e.audience = []Audience{}
	}
	e.audience = append(e.audience, Audience{tp: audType, receiver: receiver})

}

// // SetAudience sets a single audience in string format.
// func (e *EventWrapper) SetAudience(audType Eve) {
// 	e.audience = []string{a}
// }

// Serialize just writes the event out as binary.
func (e *EventWrapper) Serialize() ([]byte, error) {
	return proto.Marshal(e.Event)
}

func (e *EventWrapper) Publish(p Publisher) error {
	data, err := e.Serialize()
	if err != nil {
		return err
	}
	for _, r := range e.audience {
		// Generate topic name from recipient.
		topic := r.RecipientTopic(e.Type)
		log.Info().Str("topic", topic).Msg("ewrapper-publish-to-topic")
		err = p.PublishToTopic(topic, data)
		if err != nil {
			log.Err(err).Str("topic", r.RecipientTopic(e.Type)).Msg("publish-error")
			continue
		}
	}
	return nil
}
