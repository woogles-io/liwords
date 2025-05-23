// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: proto/comments_service/comments_service.proto

package comments_service

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

type GameComment struct {
	state       protoimpl.MessageState `protogen:"open.v1"`
	CommentId   string                 `protobuf:"bytes,1,opt,name=comment_id,json=commentId,proto3" json:"comment_id,omitempty"`
	GameId      string                 `protobuf:"bytes,2,opt,name=game_id,json=gameId,proto3" json:"game_id,omitempty"`
	UserId      string                 `protobuf:"bytes,3,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Username    string                 `protobuf:"bytes,4,opt,name=username,proto3" json:"username,omitempty"`
	EventNumber uint32                 `protobuf:"varint,5,opt,name=event_number,json=eventNumber,proto3" json:"event_number,omitempty"`
	Comment     string                 `protobuf:"bytes,6,opt,name=comment,proto3" json:"comment,omitempty"`
	LastEdited  *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=last_edited,json=lastEdited,proto3" json:"last_edited,omitempty"`
	// game_meta is for optional display of game metadata.
	GameMeta      map[string]string `protobuf:"bytes,8,rep,name=game_meta,json=gameMeta,proto3" json:"game_meta,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GameComment) Reset() {
	*x = GameComment{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GameComment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GameComment) ProtoMessage() {}

func (x *GameComment) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GameComment.ProtoReflect.Descriptor instead.
func (*GameComment) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{0}
}

func (x *GameComment) GetCommentId() string {
	if x != nil {
		return x.CommentId
	}
	return ""
}

func (x *GameComment) GetGameId() string {
	if x != nil {
		return x.GameId
	}
	return ""
}

func (x *GameComment) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *GameComment) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *GameComment) GetEventNumber() uint32 {
	if x != nil {
		return x.EventNumber
	}
	return 0
}

func (x *GameComment) GetComment() string {
	if x != nil {
		return x.Comment
	}
	return ""
}

func (x *GameComment) GetLastEdited() *timestamppb.Timestamp {
	if x != nil {
		return x.LastEdited
	}
	return nil
}

func (x *GameComment) GetGameMeta() map[string]string {
	if x != nil {
		return x.GameMeta
	}
	return nil
}

type AddCommentRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GameId        string                 `protobuf:"bytes,1,opt,name=game_id,json=gameId,proto3" json:"game_id,omitempty"`
	EventNumber   uint32                 `protobuf:"varint,2,opt,name=event_number,json=eventNumber,proto3" json:"event_number,omitempty"`
	Comment       string                 `protobuf:"bytes,3,opt,name=comment,proto3" json:"comment,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddCommentRequest) Reset() {
	*x = AddCommentRequest{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddCommentRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCommentRequest) ProtoMessage() {}

func (x *AddCommentRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCommentRequest.ProtoReflect.Descriptor instead.
func (*AddCommentRequest) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{1}
}

func (x *AddCommentRequest) GetGameId() string {
	if x != nil {
		return x.GameId
	}
	return ""
}

func (x *AddCommentRequest) GetEventNumber() uint32 {
	if x != nil {
		return x.EventNumber
	}
	return 0
}

func (x *AddCommentRequest) GetComment() string {
	if x != nil {
		return x.Comment
	}
	return ""
}

type AddCommentResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommentId     string                 `protobuf:"bytes,1,opt,name=comment_id,json=commentId,proto3" json:"comment_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddCommentResponse) Reset() {
	*x = AddCommentResponse{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddCommentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddCommentResponse) ProtoMessage() {}

func (x *AddCommentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddCommentResponse.ProtoReflect.Descriptor instead.
func (*AddCommentResponse) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{2}
}

func (x *AddCommentResponse) GetCommentId() string {
	if x != nil {
		return x.CommentId
	}
	return ""
}

type GetCommentsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GameId        string                 `protobuf:"bytes,1,opt,name=game_id,json=gameId,proto3" json:"game_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCommentsRequest) Reset() {
	*x = GetCommentsRequest{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCommentsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCommentsRequest) ProtoMessage() {}

func (x *GetCommentsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCommentsRequest.ProtoReflect.Descriptor instead.
func (*GetCommentsRequest) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{3}
}

func (x *GetCommentsRequest) GetGameId() string {
	if x != nil {
		return x.GameId
	}
	return ""
}

type GetCommentsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Comments      []*GameComment         `protobuf:"bytes,1,rep,name=comments,proto3" json:"comments,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCommentsResponse) Reset() {
	*x = GetCommentsResponse{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCommentsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCommentsResponse) ProtoMessage() {}

func (x *GetCommentsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCommentsResponse.ProtoReflect.Descriptor instead.
func (*GetCommentsResponse) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{4}
}

func (x *GetCommentsResponse) GetComments() []*GameComment {
	if x != nil {
		return x.Comments
	}
	return nil
}

type EditCommentRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommentId     string                 `protobuf:"bytes,1,opt,name=comment_id,json=commentId,proto3" json:"comment_id,omitempty"`
	Comment       string                 `protobuf:"bytes,2,opt,name=comment,proto3" json:"comment,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EditCommentRequest) Reset() {
	*x = EditCommentRequest{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EditCommentRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EditCommentRequest) ProtoMessage() {}

func (x *EditCommentRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EditCommentRequest.ProtoReflect.Descriptor instead.
func (*EditCommentRequest) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{5}
}

func (x *EditCommentRequest) GetCommentId() string {
	if x != nil {
		return x.CommentId
	}
	return ""
}

func (x *EditCommentRequest) GetComment() string {
	if x != nil {
		return x.Comment
	}
	return ""
}

type EditCommentResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EditCommentResponse) Reset() {
	*x = EditCommentResponse{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EditCommentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EditCommentResponse) ProtoMessage() {}

func (x *EditCommentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EditCommentResponse.ProtoReflect.Descriptor instead.
func (*EditCommentResponse) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{6}
}

type DeleteCommentRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CommentId     string                 `protobuf:"bytes,1,opt,name=comment_id,json=commentId,proto3" json:"comment_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteCommentRequest) Reset() {
	*x = DeleteCommentRequest{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteCommentRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCommentRequest) ProtoMessage() {}

func (x *DeleteCommentRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteCommentRequest.ProtoReflect.Descriptor instead.
func (*DeleteCommentRequest) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{7}
}

func (x *DeleteCommentRequest) GetCommentId() string {
	if x != nil {
		return x.CommentId
	}
	return ""
}

type DeleteCommentResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteCommentResponse) Reset() {
	*x = DeleteCommentResponse{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteCommentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCommentResponse) ProtoMessage() {}

func (x *DeleteCommentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteCommentResponse.ProtoReflect.Descriptor instead.
func (*DeleteCommentResponse) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{8}
}

type GetCommentsAllGamesRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Limit         uint32                 `protobuf:"varint,1,opt,name=limit,proto3" json:"limit,omitempty"`
	Offset        uint32                 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetCommentsAllGamesRequest) Reset() {
	*x = GetCommentsAllGamesRequest{}
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetCommentsAllGamesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCommentsAllGamesRequest) ProtoMessage() {}

func (x *GetCommentsAllGamesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_comments_service_comments_service_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCommentsAllGamesRequest.ProtoReflect.Descriptor instead.
func (*GetCommentsAllGamesRequest) Descriptor() ([]byte, []int) {
	return file_proto_comments_service_comments_service_proto_rawDescGZIP(), []int{9}
}

func (x *GetCommentsAllGamesRequest) GetLimit() uint32 {
	if x != nil {
		return x.Limit
	}
	return 0
}

func (x *GetCommentsAllGamesRequest) GetOffset() uint32 {
	if x != nil {
		return x.Offset
	}
	return 0
}

var File_proto_comments_service_comments_service_proto protoreflect.FileDescriptor

const file_proto_comments_service_comments_service_proto_rawDesc = "" +
	"\n" +
	"-proto/comments_service/comments_service.proto\x12\x10comments_service\x1a\x1fgoogle/protobuf/timestamp.proto\"\xfb\x02\n" +
	"\vGameComment\x12\x1d\n" +
	"\n" +
	"comment_id\x18\x01 \x01(\tR\tcommentId\x12\x17\n" +
	"\agame_id\x18\x02 \x01(\tR\x06gameId\x12\x17\n" +
	"\auser_id\x18\x03 \x01(\tR\x06userId\x12\x1a\n" +
	"\busername\x18\x04 \x01(\tR\busername\x12!\n" +
	"\fevent_number\x18\x05 \x01(\rR\veventNumber\x12\x18\n" +
	"\acomment\x18\x06 \x01(\tR\acomment\x12;\n" +
	"\vlast_edited\x18\a \x01(\v2\x1a.google.protobuf.TimestampR\n" +
	"lastEdited\x12H\n" +
	"\tgame_meta\x18\b \x03(\v2+.comments_service.GameComment.GameMetaEntryR\bgameMeta\x1a;\n" +
	"\rGameMetaEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"i\n" +
	"\x11AddCommentRequest\x12\x17\n" +
	"\agame_id\x18\x01 \x01(\tR\x06gameId\x12!\n" +
	"\fevent_number\x18\x02 \x01(\rR\veventNumber\x12\x18\n" +
	"\acomment\x18\x03 \x01(\tR\acomment\"3\n" +
	"\x12AddCommentResponse\x12\x1d\n" +
	"\n" +
	"comment_id\x18\x01 \x01(\tR\tcommentId\"-\n" +
	"\x12GetCommentsRequest\x12\x17\n" +
	"\agame_id\x18\x01 \x01(\tR\x06gameId\"P\n" +
	"\x13GetCommentsResponse\x129\n" +
	"\bcomments\x18\x01 \x03(\v2\x1d.comments_service.GameCommentR\bcomments\"M\n" +
	"\x12EditCommentRequest\x12\x1d\n" +
	"\n" +
	"comment_id\x18\x01 \x01(\tR\tcommentId\x12\x18\n" +
	"\acomment\x18\x02 \x01(\tR\acomment\"\x15\n" +
	"\x13EditCommentResponse\"5\n" +
	"\x14DeleteCommentRequest\x12\x1d\n" +
	"\n" +
	"comment_id\x18\x01 \x01(\tR\tcommentId\"\x17\n" +
	"\x15DeleteCommentResponse\"J\n" +
	"\x1aGetCommentsAllGamesRequest\x12\x14\n" +
	"\x05limit\x18\x01 \x01(\rR\x05limit\x12\x16\n" +
	"\x06offset\x18\x02 \x01(\rR\x06offset2\x86\x04\n" +
	"\x12GameCommentService\x12[\n" +
	"\x0eAddGameComment\x12#.comments_service.AddCommentRequest\x1a$.comments_service.AddCommentResponse\x12^\n" +
	"\x0fGetGameComments\x12$.comments_service.GetCommentsRequest\x1a%.comments_service.GetCommentsResponse\x12^\n" +
	"\x0fEditGameComment\x12$.comments_service.EditCommentRequest\x1a%.comments_service.EditCommentResponse\x12d\n" +
	"\x11DeleteGameComment\x12&.comments_service.DeleteCommentRequest\x1a'.comments_service.DeleteCommentResponse\x12m\n" +
	"\x16GetCommentsForAllGames\x12,.comments_service.GetCommentsAllGamesRequest\x1a%.comments_service.GetCommentsResponseB\xc6\x01\n" +
	"\x14com.comments_serviceB\x14CommentsServiceProtoP\x01Z<github.com/woogles-io/liwords/rpc/api/proto/comments_service\xa2\x02\x03CXX\xaa\x02\x0fCommentsService\xca\x02\x0fCommentsService\xe2\x02\x1bCommentsService\\GPBMetadata\xea\x02\x0fCommentsServiceb\x06proto3"

var (
	file_proto_comments_service_comments_service_proto_rawDescOnce sync.Once
	file_proto_comments_service_comments_service_proto_rawDescData []byte
)

func file_proto_comments_service_comments_service_proto_rawDescGZIP() []byte {
	file_proto_comments_service_comments_service_proto_rawDescOnce.Do(func() {
		file_proto_comments_service_comments_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_comments_service_comments_service_proto_rawDesc), len(file_proto_comments_service_comments_service_proto_rawDesc)))
	})
	return file_proto_comments_service_comments_service_proto_rawDescData
}

var file_proto_comments_service_comments_service_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_proto_comments_service_comments_service_proto_goTypes = []any{
	(*GameComment)(nil),                // 0: comments_service.GameComment
	(*AddCommentRequest)(nil),          // 1: comments_service.AddCommentRequest
	(*AddCommentResponse)(nil),         // 2: comments_service.AddCommentResponse
	(*GetCommentsRequest)(nil),         // 3: comments_service.GetCommentsRequest
	(*GetCommentsResponse)(nil),        // 4: comments_service.GetCommentsResponse
	(*EditCommentRequest)(nil),         // 5: comments_service.EditCommentRequest
	(*EditCommentResponse)(nil),        // 6: comments_service.EditCommentResponse
	(*DeleteCommentRequest)(nil),       // 7: comments_service.DeleteCommentRequest
	(*DeleteCommentResponse)(nil),      // 8: comments_service.DeleteCommentResponse
	(*GetCommentsAllGamesRequest)(nil), // 9: comments_service.GetCommentsAllGamesRequest
	nil,                                // 10: comments_service.GameComment.GameMetaEntry
	(*timestamppb.Timestamp)(nil),      // 11: google.protobuf.Timestamp
}
var file_proto_comments_service_comments_service_proto_depIdxs = []int32{
	11, // 0: comments_service.GameComment.last_edited:type_name -> google.protobuf.Timestamp
	10, // 1: comments_service.GameComment.game_meta:type_name -> comments_service.GameComment.GameMetaEntry
	0,  // 2: comments_service.GetCommentsResponse.comments:type_name -> comments_service.GameComment
	1,  // 3: comments_service.GameCommentService.AddGameComment:input_type -> comments_service.AddCommentRequest
	3,  // 4: comments_service.GameCommentService.GetGameComments:input_type -> comments_service.GetCommentsRequest
	5,  // 5: comments_service.GameCommentService.EditGameComment:input_type -> comments_service.EditCommentRequest
	7,  // 6: comments_service.GameCommentService.DeleteGameComment:input_type -> comments_service.DeleteCommentRequest
	9,  // 7: comments_service.GameCommentService.GetCommentsForAllGames:input_type -> comments_service.GetCommentsAllGamesRequest
	2,  // 8: comments_service.GameCommentService.AddGameComment:output_type -> comments_service.AddCommentResponse
	4,  // 9: comments_service.GameCommentService.GetGameComments:output_type -> comments_service.GetCommentsResponse
	6,  // 10: comments_service.GameCommentService.EditGameComment:output_type -> comments_service.EditCommentResponse
	8,  // 11: comments_service.GameCommentService.DeleteGameComment:output_type -> comments_service.DeleteCommentResponse
	4,  // 12: comments_service.GameCommentService.GetCommentsForAllGames:output_type -> comments_service.GetCommentsResponse
	8,  // [8:13] is the sub-list for method output_type
	3,  // [3:8] is the sub-list for method input_type
	3,  // [3:3] is the sub-list for extension type_name
	3,  // [3:3] is the sub-list for extension extendee
	0,  // [0:3] is the sub-list for field type_name
}

func init() { file_proto_comments_service_comments_service_proto_init() }
func file_proto_comments_service_comments_service_proto_init() {
	if File_proto_comments_service_comments_service_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_comments_service_comments_service_proto_rawDesc), len(file_proto_comments_service_comments_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_comments_service_comments_service_proto_goTypes,
		DependencyIndexes: file_proto_comments_service_comments_service_proto_depIdxs,
		MessageInfos:      file_proto_comments_service_comments_service_proto_msgTypes,
	}.Build()
	File_proto_comments_service_comments_service_proto = out.File
	file_proto_comments_service_comments_service_proto_goTypes = nil
	file_proto_comments_service_comments_service_proto_depIdxs = nil
}
