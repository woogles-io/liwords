package ipc

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// RequestProto calls the Request function but with a protobuf object. It marshals
// it before passing it across the wire, and then unmarshals the response into
// the passed in `resp` variable, which must be an instantiation of another protobuf
// message.
func RequestProto(subject string, b Publisher, msg proto.Message, resp protoreflect.ProtoMessage,
	opts ...Option) error {

	toSend, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	r, err := b.Request(subject, toSend, opts...)
	if err != nil {
		return err
	}

	err = proto.Unmarshal(r, resp)
	return err
}
