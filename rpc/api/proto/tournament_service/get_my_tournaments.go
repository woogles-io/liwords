package tournament_service

import (
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

// GetMyTournamentsRequest is a request to get tournaments owned/directed by the current user.
type GetMyTournamentsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetMyTournamentsRequest) Reset()         { *x = GetMyTournamentsRequest{} }
func (x *GetMyTournamentsRequest) String() string { return protoimpl.X.MessageStringOf(x) }
func (*GetMyTournamentsRequest) ProtoMessage()    {}

// GetMyTournamentsResponse contains tournaments owned/directed by the current user.
type GetMyTournamentsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tournaments   []*TournamentMetadata  `protobuf:"bytes,1,rep,name=tournaments,proto3" json:"tournaments,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetMyTournamentsResponse) Reset()         { *x = GetMyTournamentsResponse{} }
func (x *GetMyTournamentsResponse) String() string { return protoimpl.X.MessageStringOf(x) }
func (*GetMyTournamentsResponse) ProtoMessage()    {}

func (x *GetMyTournamentsResponse) GetTournaments() []*TournamentMetadata {
	if x != nil {
		return x.Tournaments
	}
	return nil
}
