package entity

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

const (
	// MaxNameLength is the maximum length that a proto message can be.
	MaxNameLength = 64

	protobufSerializationProtocol = "proto"
)

type EventAudienceType string

const (
	AudGame   EventAudienceType = "game"
	AudGameTV                   = "gametv"
	AudUser                     = "user"
)

// An EventWrapper is a real-time update, whether it is a played move,
// a challenged move, or the game ending, a seek beginning, etc.
type EventWrapper struct {
	Type pb.MessageType
	// The actual event should therefore be a proto object
	Event proto.Message
	// The gameID is the game this event belongs to. This will not be
	// serialized.
	gameID string

	// Serialization protocol
	protocol string
	audience []string
}

// WrapEvent wraps a protobuf event.
func WrapEvent(event proto.Message, messageType pb.MessageType, gameID string) *EventWrapper {
	return &EventWrapper{
		Type:     messageType,
		Event:    event,
		protocol: protobufSerializationProtocol,
		gameID:   gameID,
	}
}

// SetSerializationProtocol sets the serialization protocol of the protobuf
// object.
func (e *EventWrapper) SetSerializationProtocol(protocol string) {
	e.protocol = protocol
}

// AddAudience sets the audience(s) for this event. It is in the form of a NATS
// channel name. This is not required to be set in order to deliver a message,
// but certain functions will use it in the gameplay/entity module.
func (e *EventWrapper) AddAudience(audType EventAudienceType, specific string) {
	if e.audience == nil {
		e.audience = []string{}
	}
	e.audience = append(e.audience, string(audType)+"."+specific)
}

// Audience gets the audience(s) for this event, in the form of NATS channel names.
func (e *EventWrapper) Audience() []string {
	return e.audience
}

// Serialize serializes the event to a byte array.
// Our encoding inserts a two byte big-endian number indicating the length
// of the coming bytes, then a byte representing the message type to the
// start of the event.
func (e *EventWrapper) Serialize() ([]byte, error) {
	var b bytes.Buffer

	var data []byte
	var err error
	if e.protocol == "proto" {
		data, err = proto.Marshal(e.Event)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = json.Marshal(e.Event)
		if err != nil {
			return nil, err
		}
	}
	binary.Write(&b, binary.BigEndian, int16(len(data)+1))
	binary.Write(&b, binary.BigEndian, int8(e.Type))
	b.Write(data)
	return b.Bytes(), nil
}

func (e *EventWrapper) GameID() string {
	return e.gameID
}

// EventFromByteArray takes in a serialized event and deserializes it.
func EventFromByteArray(arr []byte) (*EventWrapper, error) {
	b := bytes.NewReader(arr)
	var msgtypeint int8
	var msglen int16
	binary.Read(b, binary.BigEndian, &msglen)
	binary.Read(b, binary.BigEndian, &msgtypeint)
	var message proto.Message
	msgBytes := make([]byte, msglen-1)
	msgType := pb.MessageType(msgtypeint)
	switch msgType {
	case pb.MessageType_CLIENT_GAMEPLAY_EVENT:
		message = &pb.ClientGameplayEvent{}
	case pb.MessageType_GAME_ENDED_EVENT:
		message = &pb.GameEndedEvent{}
	case pb.MessageType_GAME_HISTORY_REFRESHER:
		message = &pb.GameHistoryRefresher{}
	case pb.MessageType_SEEK_REQUEST:
		message = &pb.SeekRequest{}
	case pb.MessageType_GAME_ACCEPTED_EVENT:
		message = &pb.GameAcceptedEvent{}
	case pb.MessageType_JOIN_PATH:
		message = &pb.JoinPath{}
	case pb.MessageType_UNJOIN_REALM:
		message = &pb.UnjoinRealm{}
	case pb.MessageType_TIMED_OUT:
		message = &pb.TimedOut{}
	case pb.MessageType_TOKEN_SOCKET_LOGIN:
		message = &pb.TokenSocketLogin{}
	default:
		return nil, fmt.Errorf("event of type %d not handled", msgType)
	}
	_, err := io.ReadFull(b, msgBytes)
	if err != nil {
		return nil, err
	}

	// Assume it's protobuf unless it's surrounded by two { } brackets.
	// This is kind of ghetto but fast.
	if msgBytes[0] == '{' && msgBytes[len(msgBytes)-1] == '}' {
		err = json.Unmarshal(msgBytes, message)
		if err != nil {
			return nil, err
		}
	} else {
		err = proto.Unmarshal(msgBytes, message)
		if err != nil {
			return nil, err
		}
	}
	// The game ID doesn't matter here. This function handles incoming events,
	// and the downstream handlers already know how to decode the gameID,
	// if any, from them.
	return WrapEvent(message, msgType, ""), nil
}
