// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: proto/ipc/chat.proto

// Package ipc describes all the messages used for inter-process
// communication between the different microservices in liwords
// (so far, just the API and the socket server).
// Many of these messages end up being transmitted to the front-end.

package ipc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ChatMessage struct {
	state    protoimpl.MessageState `protogen:"open.v1"`
	Username string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Channel  string                 `protobuf:"bytes,2,opt,name=channel,proto3" json:"channel,omitempty"`
	Message  string                 `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	// timestamp is in milliseconds!
	Timestamp int64  `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	UserId    string `protobuf:"bytes,5,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Id        string `protobuf:"bytes,6,opt,name=id,proto3" json:"id,omitempty"`
	// these are only loaded by specific endpoints.
	CountryCode   string `protobuf:"bytes,7,opt,name=country_code,json=countryCode,proto3" json:"country_code,omitempty"`
	AvatarUrl     string `protobuf:"bytes,8,opt,name=avatar_url,json=avatarUrl,proto3" json:"avatar_url,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChatMessage) Reset() {
	*x = ChatMessage{}
	mi := &file_proto_ipc_chat_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChatMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChatMessage) ProtoMessage() {}

func (x *ChatMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ipc_chat_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChatMessage.ProtoReflect.Descriptor instead.
func (*ChatMessage) Descriptor() ([]byte, []int) {
	return file_proto_ipc_chat_proto_rawDescGZIP(), []int{0}
}

func (x *ChatMessage) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *ChatMessage) GetChannel() string {
	if x != nil {
		return x.Channel
	}
	return ""
}

func (x *ChatMessage) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *ChatMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *ChatMessage) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *ChatMessage) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ChatMessage) GetCountryCode() string {
	if x != nil {
		return x.CountryCode
	}
	return ""
}

func (x *ChatMessage) GetAvatarUrl() string {
	if x != nil {
		return x.AvatarUrl
	}
	return ""
}

type ChatMessages struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Messages      []*ChatMessage         `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChatMessages) Reset() {
	*x = ChatMessages{}
	mi := &file_proto_ipc_chat_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChatMessages) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChatMessages) ProtoMessage() {}

func (x *ChatMessages) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ipc_chat_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChatMessages.ProtoReflect.Descriptor instead.
func (*ChatMessages) Descriptor() ([]byte, []int) {
	return file_proto_ipc_chat_proto_rawDescGZIP(), []int{1}
}

func (x *ChatMessages) GetMessages() []*ChatMessage {
	if x != nil {
		return x.Messages
	}
	return nil
}

type ChatMessageDeleted struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Channel       string                 `protobuf:"bytes,1,opt,name=channel,proto3" json:"channel,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChatMessageDeleted) Reset() {
	*x = ChatMessageDeleted{}
	mi := &file_proto_ipc_chat_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChatMessageDeleted) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChatMessageDeleted) ProtoMessage() {}

func (x *ChatMessageDeleted) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ipc_chat_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChatMessageDeleted.ProtoReflect.Descriptor instead.
func (*ChatMessageDeleted) Descriptor() ([]byte, []int) {
	return file_proto_ipc_chat_proto_rawDescGZIP(), []int{2}
}

func (x *ChatMessageDeleted) GetChannel() string {
	if x != nil {
		return x.Channel
	}
	return ""
}

func (x *ChatMessageDeleted) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

var File_proto_ipc_chat_proto protoreflect.FileDescriptor

const file_proto_ipc_chat_proto_rawDesc = "" +
	"\n" +
	"\x14proto/ipc/chat.proto\x12\x03ipc\"\xe6\x01\n" +
	"\vChatMessage\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\x12\x18\n" +
	"\achannel\x18\x02 \x01(\tR\achannel\x12\x18\n" +
	"\amessage\x18\x03 \x01(\tR\amessage\x12\x1c\n" +
	"\ttimestamp\x18\x04 \x01(\x03R\ttimestamp\x12\x17\n" +
	"\auser_id\x18\x05 \x01(\tR\x06userId\x12\x0e\n" +
	"\x02id\x18\x06 \x01(\tR\x02id\x12!\n" +
	"\fcountry_code\x18\a \x01(\tR\vcountryCode\x12\x1d\n" +
	"\n" +
	"avatar_url\x18\b \x01(\tR\tavatarUrl\"<\n" +
	"\fChatMessages\x12,\n" +
	"\bmessages\x18\x01 \x03(\v2\x10.ipc.ChatMessageR\bmessages\">\n" +
	"\x12ChatMessageDeleted\x12\x18\n" +
	"\achannel\x18\x01 \x01(\tR\achannel\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02idBq\n" +
	"\acom.ipcB\tChatProtoP\x01Z/github.com/woogles-io/liwords/rpc/api/proto/ipc\xa2\x02\x03IXX\xaa\x02\x03Ipc\xca\x02\x03Ipc\xe2\x02\x0fIpc\\GPBMetadata\xea\x02\x03Ipcb\x06proto3"

var (
	file_proto_ipc_chat_proto_rawDescOnce sync.Once
	file_proto_ipc_chat_proto_rawDescData []byte
)

func file_proto_ipc_chat_proto_rawDescGZIP() []byte {
	file_proto_ipc_chat_proto_rawDescOnce.Do(func() {
		file_proto_ipc_chat_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_ipc_chat_proto_rawDesc), len(file_proto_ipc_chat_proto_rawDesc)))
	})
	return file_proto_ipc_chat_proto_rawDescData
}

var file_proto_ipc_chat_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proto_ipc_chat_proto_goTypes = []any{
	(*ChatMessage)(nil),        // 0: ipc.ChatMessage
	(*ChatMessages)(nil),       // 1: ipc.ChatMessages
	(*ChatMessageDeleted)(nil), // 2: ipc.ChatMessageDeleted
}
var file_proto_ipc_chat_proto_depIdxs = []int32{
	0, // 0: ipc.ChatMessages.messages:type_name -> ipc.ChatMessage
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proto_ipc_chat_proto_init() }
func file_proto_ipc_chat_proto_init() {
	if File_proto_ipc_chat_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_ipc_chat_proto_rawDesc), len(file_proto_ipc_chat_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_ipc_chat_proto_goTypes,
		DependencyIndexes: file_proto_ipc_chat_proto_depIdxs,
		MessageInfos:      file_proto_ipc_chat_proto_msgTypes,
	}.Build()
	File_proto_ipc_chat_proto = out.File
	file_proto_ipc_chat_proto_goTypes = nil
	file_proto_ipc_chat_proto_depIdxs = nil
}
