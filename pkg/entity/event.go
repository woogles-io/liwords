package entity

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"

	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/crosswords/rpc/api/proto"
)

const (
	// MaxNameLength is the maximum length that a proto message can be.
	MaxNameLength = 64

	protobufSerializationProtocol = "proto"
)

// An EventWrapper is a real-time update, whether it is a played move,
// a challenged move, or the game ending, a seek beginning, etc.
type EventWrapper struct {
	// The event name will correspond to a protobuf name.
	Name string
	// The actual event should therefore be a proto object
	Event proto.Message

	// Serialization protocol
	protocol string
}

// WrapEvent wraps a protobuf event.
func WrapEvent(event proto.Message) *EventWrapper {
	return &EventWrapper{
		Name:     string(event.ProtoReflect().Descriptor().FullName()),
		Event:    event,
		protocol: protobufSerializationProtocol,
	}
}

// SetSerializationProtocol sets the serialization protocol of the protobuf
// object.
func (e *EventWrapper) SetSerializationProtocol(protocol string) {
	e.protocol = protocol
}

// Serialize serializes the event to a byte array. Our encoding
// looks like the following:
// L1L2 -  Big endian 16-bit length of the entire data packet.
// LN (single byte for the length of the Name)
// Name (encoded to ASCII -- all our event names better be ASCII)
// Protobuf representation of the event.
func (e *EventWrapper) Serialize() ([]byte, error) {
	var b bytes.Buffer
	if len(e.Name) > MaxNameLength {
		return nil, errors.New("event name too long")
	}

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
	binary.Write(&b, binary.BigEndian, int16(len(e.Name)+len(data)))
	binary.Write(&b, binary.BigEndian, byte(len(e.Name)))
	b.Write([]byte(e.Name))
	b.Write(data)
	return b.Bytes(), nil
}

// EventFromByteArray takes in a serialized event and deserializes it.
func EventFromByteArray(arr []byte) (*EventWrapper, error) {
	b := bytes.NewReader(arr)
	var totalLength int16
	var namelength byte

	binary.Read(b, binary.BigEndian, &totalLength)
	binary.Read(b, binary.BigEndian, &namelength)

	nameholder := make([]byte, namelength)
	_, err := io.ReadFull(b, nameholder)
	if err != nil {
		return nil, err
	}

	name := string(nameholder)
	var message proto.Message
	msgLength := totalLength - int16(namelength)

	msgBytes := make([]byte, msgLength)

	switch name {
	// Add all relevant events here!
	// SeekRequest and MatchRequest will be sent from the user via regular
	// API and not necessarily the socket, so we won't need to process them here.
	case "crosswords.UserGameplayEvent":
		message = &pb.UserGameplayEvent{}
	case "crosswords.GameEndedEvent":
		message = &pb.GameEndedEvent{}
	case "crosswords.GameHistoryRefresher":
		message = &pb.GameHistoryRefresher{}
	default:
		return nil, errors.New("event of type " + name + " not handled")
	}
	_, err = io.ReadFull(b, msgBytes)
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
	return WrapEvent(message), nil
}
