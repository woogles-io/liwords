// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: proto/ipc/errors.proto

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

type WooglesError int32

const (
	WooglesError_DEFAULT                                                WooglesError = 0
	WooglesError_TOURNAMENT_NEGATIVE_MAX_BYE_PLACEMENT                  WooglesError = 1001
	WooglesError_TOURNAMENT_NEGATIVE_MIN_PLACEMENT                      WooglesError = 1002
	WooglesError_TOURNAMENT_NEGATIVE_GIBSON_SPREAD                      WooglesError = 1003
	WooglesError_TOURNAMENT_EMPTY_ROUND_CONTROLS                        WooglesError = 1004
	WooglesError_TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START              WooglesError = 1005
	WooglesError_TOURNAMENT_ELIMINATION_PAIRINGS_MIX                    WooglesError = 1006
	WooglesError_TOURNAMENT_DISCONTINUOUS_INITIAL_FONTES                WooglesError = 1007
	WooglesError_TOURNAMENT_INVALID_INITIAL_FONTES_ROUNDS               WooglesError = 1008
	WooglesError_TOURNAMENT_INVALID_ELIMINATION_PLAYERS                 WooglesError = 1009
	WooglesError_TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE                   WooglesError = 1010
	WooglesError_TOURNAMENT_NONEXISTENT_PLAYER                          WooglesError = 1011
	WooglesError_TOURNAMENT_NONAMENDMENT_PAST_RESULT                    WooglesError = 1012
	WooglesError_TOURNAMENT_FUTURE_NONBYE_RESULT                        WooglesError = 1013
	WooglesError_TOURNAMENT_NIL_PLAYER_PAIRING                          WooglesError = 1014
	WooglesError_TOURNAMENT_NONOPPONENTS                                WooglesError = 1015
	WooglesError_TOURNAMENT_MIXED_VOID_AND_NONVOID_RESULTS              WooglesError = 1016
	WooglesError_TOURNAMENT_NONEXISTENT_PAIRING                         WooglesError = 1017
	WooglesError_TOURNAMENT_UNINITIALIZED_GAMES                         WooglesError = 1018
	WooglesError_TOURNAMENT_TIEBREAK_INVALID_GAME_INDEX                 WooglesError = 1019
	WooglesError_TOURNAMENT_GAME_INDEX_OUT_OF_RANGE                     WooglesError = 1020
	WooglesError_TOURNAMENT_RESULT_ALREADY_SUBMITTED                    WooglesError = 1021
	WooglesError_TOURNAMENT_NONEXISTENT_RESULT_AMENDMENT                WooglesError = 1022
	WooglesError_TOURNAMENT_GIBSON_CAN_CATCH                            WooglesError = 1023
	WooglesError_TOURNAMENT_CANNOT_ASSIGN_BYE                           WooglesError = 1024
	WooglesError_TOURNAMENT_INTERNAL_BYE_ASSIGNMENT                     WooglesError = 1025
	WooglesError_TOURNAMENT_INCORRECT_PAIRINGS_LENGTH                   WooglesError = 1026
	WooglesError_TOURNAMENT_PAIRINGS_ASSIGNED_BYE                       WooglesError = 1027
	WooglesError_TOURNAMENT_SUSPENDED_PLAYER_UNREMOVED                  WooglesError = 1028
	WooglesError_TOURNAMENT_PAIRING_INDEX_OUT_OF_RANGE                  WooglesError = 1029
	WooglesError_TOURNAMENT_SUSPENDED_PLAYER_PAIRED                     WooglesError = 1030
	WooglesError_TOURNAMENT_PLAYER_NOT_PAIRED                           WooglesError = 1031
	WooglesError_TOURNAMENT_PLAYER_ALREADY_EXISTS                       WooglesError = 1032
	WooglesError_TOURNAMENT_ADD_PLAYERS_LAST_ROUND                      WooglesError = 1033
	WooglesError_TOURNAMENT_PLAYER_INDEX_OUT_OF_RANGE                   WooglesError = 1034
	WooglesError_TOURNAMENT_PLAYER_ALREADY_REMOVED                      WooglesError = 1035
	WooglesError_TOURNAMENT_REMOVAL_CREATES_EMPTY_DIVISION              WooglesError = 1036
	WooglesError_TOURNAMENT_NEGATIVE_GIBSON_ROUND                       WooglesError = 1037
	WooglesError_TOURNAMENT_ROUND_NOT_COMPLETE                          WooglesError = 1038
	WooglesError_TOURNAMENT_FINISHED                                    WooglesError = 1039
	WooglesError_TOURNAMENT_NOT_STARTABLE                               WooglesError = 1040
	WooglesError_TOURNAMENT_ROUND_NOT_READY                             WooglesError = 1041
	WooglesError_TOURNAMENT_SET_GAME_ROUND_NUMBER                       WooglesError = 1042
	WooglesError_TOURNAMENT_ALREADY_READY                               WooglesError = 1043
	WooglesError_TOURNAMENT_SET_READY_MULTIPLE_IDS                      WooglesError = 1044
	WooglesError_TOURNAMENT_SET_READY_PLAYER_NOT_FOUND                  WooglesError = 1045
	WooglesError_TOURNAMENT_NO_LOSER                                    WooglesError = 1046
	WooglesError_TOURNAMENT_NO_WINNER                                   WooglesError = 1047
	WooglesError_TOURNAMENT_UNPAIRED_PLAYER                             WooglesError = 1048
	WooglesError_TOURNAMENT_INVALID_PAIRING                             WooglesError = 1049
	WooglesError_TOURNAMENT_INVALID_SWISS                               WooglesError = 1050
	WooglesError_TOURNAMENT_ZERO_GAMES_PER_ROUND                        WooglesError = 1051
	WooglesError_TOURNAMENT_EMPTY_NAME                                  WooglesError = 1052
	WooglesError_TOURNAMENT_NOT_STARTED                                 WooglesError = 1053
	WooglesError_TOURNAMENT_NONEXISTENT_DIVISION                        WooglesError = 1054
	WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER                        WooglesError = 1055
	WooglesError_TOURNAMENT_SET_NON_FUTURE_ROUND_CONTROLS               WooglesError = 1056
	WooglesError_TOURNAMENT_ADD_DIVISION_AFTER_START                    WooglesError = 1057
	WooglesError_TOURNAMENT_INVALID_DIVISION_NAME                       WooglesError = 1058
	WooglesError_TOURNAMENT_DIVISION_ALREADY_EXISTS                     WooglesError = 1059
	WooglesError_TOURNAMENT_DIVISION_REMOVAL_AFTER_START                WooglesError = 1060
	WooglesError_TOURNAMENT_DIVISION_REMOVAL_EXISTING_PLAYERS           WooglesError = 1061
	WooglesError_TOURNAMENT_PLAYER_ID_CONSTRUCTION                      WooglesError = 1062
	WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_EXISTS                   WooglesError = 1063
	WooglesError_TOURNAMENT_DIRECTOR_EXISTS                             WooglesError = 1064
	WooglesError_TOURNAMENT_NO_DIVISIONS                                WooglesError = 1065
	WooglesError_TOURNAMENT_GAME_CONTROLS_NOT_SET                       WooglesError = 1066
	WooglesError_TOURNAMENT_INCORRECT_START_ROUND                       WooglesError = 1067
	WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND                       WooglesError = 1068
	WooglesError_TOURNAMENT_DELETE_NON_FUTURE_ROUND                     WooglesError = 1069
	WooglesError_TOURNAMENT_DIVISION_NOT_FINISHED                       WooglesError = 1070
	WooglesError_TOURNAMENT_NOT_EXACTLY_ONE_EXECUTIVE_DIRECTOR          WooglesError = 1071
	WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL                  WooglesError = 1072
	WooglesError_TOURNAMENT_INVALID_FUTURE_RESULT                       WooglesError = 1073
	WooglesError_TOURNAMENT_SCHEDULED_START_AFTER_END                   WooglesError = 1090
	WooglesError_TOURNAMENT_NOT_FINISHED                                WooglesError = 1091
	WooglesError_TOURNAMENT_OPENCHECKINS_AFTER_START                    WooglesError = 1092
	WooglesError_TOURNAMENT_CHECKINS_CLOSED                             WooglesError = 1093
	WooglesError_TOURNAMENT_NOT_REGISTERED                              WooglesError = 1094
	WooglesError_TOURNAMENT_REGISTRATIONS_CLOSED                        WooglesError = 1095
	WooglesError_TOURNAMENT_ALREADY_STARTED                             WooglesError = 1096
	WooglesError_TOURNAMENT_OPENREGISTRATIONS_AFTER_START               WooglesError = 1097
	WooglesError_TOURNAMENT_CANNOT_START_CHECKINS_OR_REGISTRATIONS_OPEN WooglesError = 1098
	WooglesError_TOURNAMENT_CANNOT_REMOVE_UNCHECKED_IN_IF_CHECKINS_OPEN WooglesError = 1099
	WooglesError_PUZZLE_VOTE_INVALID                                    WooglesError = 1074
	WooglesError_PUZZLE_GET_RANDOM_PUZZLE_ID_NOT_FOUND                  WooglesError = 1075
	WooglesError_PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND                     WooglesError = 1076
	WooglesError_PUZZLE_GET_PUZZLE_UUID_NOT_FOUND                       WooglesError = 1077
	WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS                 WooglesError = 1078
	WooglesError_PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND           WooglesError = 1079
	WooglesError_PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND                WooglesError = 1080
	WooglesError_PUZZLE_SUBMIT_ANSWER_PUZZLE_ID_NOT_FOUND               WooglesError = 1081
	WooglesError_PUZZLE_SUBMIT_ANSWER_SET_CORRECT                       WooglesError = 1082
	WooglesError_PUZZLE_SUBMIT_ANSWER_SET_ATTEMPTS                      WooglesError = 1083
	WooglesError_PUZZLE_SET_PUZZLE_VOTE_ID_NOT_FOUND                    WooglesError = 1084
	WooglesError_PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND          WooglesError = 1085
	WooglesError_PUZZLE_GET_PUZZLE_UPDATE_ATTEMPT                       WooglesError = 1086
	WooglesError_PUZZLE_GET_ANSWER_NOT_YET_RATED                        WooglesError = 1087
	WooglesError_USER_UPDATE_NOT_FOUND                                  WooglesError = 1088
	WooglesError_GAME_NO_LONGER_AVAILABLE                               WooglesError = 1089
)

// Enum value maps for WooglesError.
var (
	WooglesError_name = map[int32]string{
		0:    "DEFAULT",
		1001: "TOURNAMENT_NEGATIVE_MAX_BYE_PLACEMENT",
		1002: "TOURNAMENT_NEGATIVE_MIN_PLACEMENT",
		1003: "TOURNAMENT_NEGATIVE_GIBSON_SPREAD",
		1004: "TOURNAMENT_EMPTY_ROUND_CONTROLS",
		1005: "TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START",
		1006: "TOURNAMENT_ELIMINATION_PAIRINGS_MIX",
		1007: "TOURNAMENT_DISCONTINUOUS_INITIAL_FONTES",
		1008: "TOURNAMENT_INVALID_INITIAL_FONTES_ROUNDS",
		1009: "TOURNAMENT_INVALID_ELIMINATION_PLAYERS",
		1010: "TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE",
		1011: "TOURNAMENT_NONEXISTENT_PLAYER",
		1012: "TOURNAMENT_NONAMENDMENT_PAST_RESULT",
		1013: "TOURNAMENT_FUTURE_NONBYE_RESULT",
		1014: "TOURNAMENT_NIL_PLAYER_PAIRING",
		1015: "TOURNAMENT_NONOPPONENTS",
		1016: "TOURNAMENT_MIXED_VOID_AND_NONVOID_RESULTS",
		1017: "TOURNAMENT_NONEXISTENT_PAIRING",
		1018: "TOURNAMENT_UNINITIALIZED_GAMES",
		1019: "TOURNAMENT_TIEBREAK_INVALID_GAME_INDEX",
		1020: "TOURNAMENT_GAME_INDEX_OUT_OF_RANGE",
		1021: "TOURNAMENT_RESULT_ALREADY_SUBMITTED",
		1022: "TOURNAMENT_NONEXISTENT_RESULT_AMENDMENT",
		1023: "TOURNAMENT_GIBSON_CAN_CATCH",
		1024: "TOURNAMENT_CANNOT_ASSIGN_BYE",
		1025: "TOURNAMENT_INTERNAL_BYE_ASSIGNMENT",
		1026: "TOURNAMENT_INCORRECT_PAIRINGS_LENGTH",
		1027: "TOURNAMENT_PAIRINGS_ASSIGNED_BYE",
		1028: "TOURNAMENT_SUSPENDED_PLAYER_UNREMOVED",
		1029: "TOURNAMENT_PAIRING_INDEX_OUT_OF_RANGE",
		1030: "TOURNAMENT_SUSPENDED_PLAYER_PAIRED",
		1031: "TOURNAMENT_PLAYER_NOT_PAIRED",
		1032: "TOURNAMENT_PLAYER_ALREADY_EXISTS",
		1033: "TOURNAMENT_ADD_PLAYERS_LAST_ROUND",
		1034: "TOURNAMENT_PLAYER_INDEX_OUT_OF_RANGE",
		1035: "TOURNAMENT_PLAYER_ALREADY_REMOVED",
		1036: "TOURNAMENT_REMOVAL_CREATES_EMPTY_DIVISION",
		1037: "TOURNAMENT_NEGATIVE_GIBSON_ROUND",
		1038: "TOURNAMENT_ROUND_NOT_COMPLETE",
		1039: "TOURNAMENT_FINISHED",
		1040: "TOURNAMENT_NOT_STARTABLE",
		1041: "TOURNAMENT_ROUND_NOT_READY",
		1042: "TOURNAMENT_SET_GAME_ROUND_NUMBER",
		1043: "TOURNAMENT_ALREADY_READY",
		1044: "TOURNAMENT_SET_READY_MULTIPLE_IDS",
		1045: "TOURNAMENT_SET_READY_PLAYER_NOT_FOUND",
		1046: "TOURNAMENT_NO_LOSER",
		1047: "TOURNAMENT_NO_WINNER",
		1048: "TOURNAMENT_UNPAIRED_PLAYER",
		1049: "TOURNAMENT_INVALID_PAIRING",
		1050: "TOURNAMENT_INVALID_SWISS",
		1051: "TOURNAMENT_ZERO_GAMES_PER_ROUND",
		1052: "TOURNAMENT_EMPTY_NAME",
		1053: "TOURNAMENT_NOT_STARTED",
		1054: "TOURNAMENT_NONEXISTENT_DIVISION",
		1055: "TOURNAMENT_NIL_DIVISION_MANAGER",
		1056: "TOURNAMENT_SET_NON_FUTURE_ROUND_CONTROLS",
		1057: "TOURNAMENT_ADD_DIVISION_AFTER_START",
		1058: "TOURNAMENT_INVALID_DIVISION_NAME",
		1059: "TOURNAMENT_DIVISION_ALREADY_EXISTS",
		1060: "TOURNAMENT_DIVISION_REMOVAL_AFTER_START",
		1061: "TOURNAMENT_DIVISION_REMOVAL_EXISTING_PLAYERS",
		1062: "TOURNAMENT_PLAYER_ID_CONSTRUCTION",
		1063: "TOURNAMENT_EXECUTIVE_DIRECTOR_EXISTS",
		1064: "TOURNAMENT_DIRECTOR_EXISTS",
		1065: "TOURNAMENT_NO_DIVISIONS",
		1066: "TOURNAMENT_GAME_CONTROLS_NOT_SET",
		1067: "TOURNAMENT_INCORRECT_START_ROUND",
		1068: "TOURNAMENT_PAIR_NON_FUTURE_ROUND",
		1069: "TOURNAMENT_DELETE_NON_FUTURE_ROUND",
		1070: "TOURNAMENT_DIVISION_NOT_FINISHED",
		1071: "TOURNAMENT_NOT_EXACTLY_ONE_EXECUTIVE_DIRECTOR",
		1072: "TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL",
		1073: "TOURNAMENT_INVALID_FUTURE_RESULT",
		1090: "TOURNAMENT_SCHEDULED_START_AFTER_END",
		1091: "TOURNAMENT_NOT_FINISHED",
		1092: "TOURNAMENT_OPENCHECKINS_AFTER_START",
		1093: "TOURNAMENT_CHECKINS_CLOSED",
		1094: "TOURNAMENT_NOT_REGISTERED",
		1095: "TOURNAMENT_REGISTRATIONS_CLOSED",
		1096: "TOURNAMENT_ALREADY_STARTED",
		1097: "TOURNAMENT_OPENREGISTRATIONS_AFTER_START",
		1098: "TOURNAMENT_CANNOT_START_CHECKINS_OR_REGISTRATIONS_OPEN",
		1099: "TOURNAMENT_CANNOT_REMOVE_UNCHECKED_IN_IF_CHECKINS_OPEN",
		1074: "PUZZLE_VOTE_INVALID",
		1075: "PUZZLE_GET_RANDOM_PUZZLE_ID_NOT_FOUND",
		1076: "PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND",
		1077: "PUZZLE_GET_PUZZLE_UUID_NOT_FOUND",
		1078: "PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS",
		1079: "PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND",
		1080: "PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND",
		1081: "PUZZLE_SUBMIT_ANSWER_PUZZLE_ID_NOT_FOUND",
		1082: "PUZZLE_SUBMIT_ANSWER_SET_CORRECT",
		1083: "PUZZLE_SUBMIT_ANSWER_SET_ATTEMPTS",
		1084: "PUZZLE_SET_PUZZLE_VOTE_ID_NOT_FOUND",
		1085: "PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND",
		1086: "PUZZLE_GET_PUZZLE_UPDATE_ATTEMPT",
		1087: "PUZZLE_GET_ANSWER_NOT_YET_RATED",
		1088: "USER_UPDATE_NOT_FOUND",
		1089: "GAME_NO_LONGER_AVAILABLE",
	}
	WooglesError_value = map[string]int32{
		"DEFAULT":                                                0,
		"TOURNAMENT_NEGATIVE_MAX_BYE_PLACEMENT":                  1001,
		"TOURNAMENT_NEGATIVE_MIN_PLACEMENT":                      1002,
		"TOURNAMENT_NEGATIVE_GIBSON_SPREAD":                      1003,
		"TOURNAMENT_EMPTY_ROUND_CONTROLS":                        1004,
		"TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START":              1005,
		"TOURNAMENT_ELIMINATION_PAIRINGS_MIX":                    1006,
		"TOURNAMENT_DISCONTINUOUS_INITIAL_FONTES":                1007,
		"TOURNAMENT_INVALID_INITIAL_FONTES_ROUNDS":               1008,
		"TOURNAMENT_INVALID_ELIMINATION_PLAYERS":                 1009,
		"TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE":                   1010,
		"TOURNAMENT_NONEXISTENT_PLAYER":                          1011,
		"TOURNAMENT_NONAMENDMENT_PAST_RESULT":                    1012,
		"TOURNAMENT_FUTURE_NONBYE_RESULT":                        1013,
		"TOURNAMENT_NIL_PLAYER_PAIRING":                          1014,
		"TOURNAMENT_NONOPPONENTS":                                1015,
		"TOURNAMENT_MIXED_VOID_AND_NONVOID_RESULTS":              1016,
		"TOURNAMENT_NONEXISTENT_PAIRING":                         1017,
		"TOURNAMENT_UNINITIALIZED_GAMES":                         1018,
		"TOURNAMENT_TIEBREAK_INVALID_GAME_INDEX":                 1019,
		"TOURNAMENT_GAME_INDEX_OUT_OF_RANGE":                     1020,
		"TOURNAMENT_RESULT_ALREADY_SUBMITTED":                    1021,
		"TOURNAMENT_NONEXISTENT_RESULT_AMENDMENT":                1022,
		"TOURNAMENT_GIBSON_CAN_CATCH":                            1023,
		"TOURNAMENT_CANNOT_ASSIGN_BYE":                           1024,
		"TOURNAMENT_INTERNAL_BYE_ASSIGNMENT":                     1025,
		"TOURNAMENT_INCORRECT_PAIRINGS_LENGTH":                   1026,
		"TOURNAMENT_PAIRINGS_ASSIGNED_BYE":                       1027,
		"TOURNAMENT_SUSPENDED_PLAYER_UNREMOVED":                  1028,
		"TOURNAMENT_PAIRING_INDEX_OUT_OF_RANGE":                  1029,
		"TOURNAMENT_SUSPENDED_PLAYER_PAIRED":                     1030,
		"TOURNAMENT_PLAYER_NOT_PAIRED":                           1031,
		"TOURNAMENT_PLAYER_ALREADY_EXISTS":                       1032,
		"TOURNAMENT_ADD_PLAYERS_LAST_ROUND":                      1033,
		"TOURNAMENT_PLAYER_INDEX_OUT_OF_RANGE":                   1034,
		"TOURNAMENT_PLAYER_ALREADY_REMOVED":                      1035,
		"TOURNAMENT_REMOVAL_CREATES_EMPTY_DIVISION":              1036,
		"TOURNAMENT_NEGATIVE_GIBSON_ROUND":                       1037,
		"TOURNAMENT_ROUND_NOT_COMPLETE":                          1038,
		"TOURNAMENT_FINISHED":                                    1039,
		"TOURNAMENT_NOT_STARTABLE":                               1040,
		"TOURNAMENT_ROUND_NOT_READY":                             1041,
		"TOURNAMENT_SET_GAME_ROUND_NUMBER":                       1042,
		"TOURNAMENT_ALREADY_READY":                               1043,
		"TOURNAMENT_SET_READY_MULTIPLE_IDS":                      1044,
		"TOURNAMENT_SET_READY_PLAYER_NOT_FOUND":                  1045,
		"TOURNAMENT_NO_LOSER":                                    1046,
		"TOURNAMENT_NO_WINNER":                                   1047,
		"TOURNAMENT_UNPAIRED_PLAYER":                             1048,
		"TOURNAMENT_INVALID_PAIRING":                             1049,
		"TOURNAMENT_INVALID_SWISS":                               1050,
		"TOURNAMENT_ZERO_GAMES_PER_ROUND":                        1051,
		"TOURNAMENT_EMPTY_NAME":                                  1052,
		"TOURNAMENT_NOT_STARTED":                                 1053,
		"TOURNAMENT_NONEXISTENT_DIVISION":                        1054,
		"TOURNAMENT_NIL_DIVISION_MANAGER":                        1055,
		"TOURNAMENT_SET_NON_FUTURE_ROUND_CONTROLS":               1056,
		"TOURNAMENT_ADD_DIVISION_AFTER_START":                    1057,
		"TOURNAMENT_INVALID_DIVISION_NAME":                       1058,
		"TOURNAMENT_DIVISION_ALREADY_EXISTS":                     1059,
		"TOURNAMENT_DIVISION_REMOVAL_AFTER_START":                1060,
		"TOURNAMENT_DIVISION_REMOVAL_EXISTING_PLAYERS":           1061,
		"TOURNAMENT_PLAYER_ID_CONSTRUCTION":                      1062,
		"TOURNAMENT_EXECUTIVE_DIRECTOR_EXISTS":                   1063,
		"TOURNAMENT_DIRECTOR_EXISTS":                             1064,
		"TOURNAMENT_NO_DIVISIONS":                                1065,
		"TOURNAMENT_GAME_CONTROLS_NOT_SET":                       1066,
		"TOURNAMENT_INCORRECT_START_ROUND":                       1067,
		"TOURNAMENT_PAIR_NON_FUTURE_ROUND":                       1068,
		"TOURNAMENT_DELETE_NON_FUTURE_ROUND":                     1069,
		"TOURNAMENT_DIVISION_NOT_FINISHED":                       1070,
		"TOURNAMENT_NOT_EXACTLY_ONE_EXECUTIVE_DIRECTOR":          1071,
		"TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL":                  1072,
		"TOURNAMENT_INVALID_FUTURE_RESULT":                       1073,
		"TOURNAMENT_SCHEDULED_START_AFTER_END":                   1090,
		"TOURNAMENT_NOT_FINISHED":                                1091,
		"TOURNAMENT_OPENCHECKINS_AFTER_START":                    1092,
		"TOURNAMENT_CHECKINS_CLOSED":                             1093,
		"TOURNAMENT_NOT_REGISTERED":                              1094,
		"TOURNAMENT_REGISTRATIONS_CLOSED":                        1095,
		"TOURNAMENT_ALREADY_STARTED":                             1096,
		"TOURNAMENT_OPENREGISTRATIONS_AFTER_START":               1097,
		"TOURNAMENT_CANNOT_START_CHECKINS_OR_REGISTRATIONS_OPEN": 1098,
		"TOURNAMENT_CANNOT_REMOVE_UNCHECKED_IN_IF_CHECKINS_OPEN": 1099,
		"PUZZLE_VOTE_INVALID":                                    1074,
		"PUZZLE_GET_RANDOM_PUZZLE_ID_NOT_FOUND":                  1075,
		"PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND":                     1076,
		"PUZZLE_GET_PUZZLE_UUID_NOT_FOUND":                       1077,
		"PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS":                 1078,
		"PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND":           1079,
		"PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND":                1080,
		"PUZZLE_SUBMIT_ANSWER_PUZZLE_ID_NOT_FOUND":               1081,
		"PUZZLE_SUBMIT_ANSWER_SET_CORRECT":                       1082,
		"PUZZLE_SUBMIT_ANSWER_SET_ATTEMPTS":                      1083,
		"PUZZLE_SET_PUZZLE_VOTE_ID_NOT_FOUND":                    1084,
		"PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND":          1085,
		"PUZZLE_GET_PUZZLE_UPDATE_ATTEMPT":                       1086,
		"PUZZLE_GET_ANSWER_NOT_YET_RATED":                        1087,
		"USER_UPDATE_NOT_FOUND":                                  1088,
		"GAME_NO_LONGER_AVAILABLE":                               1089,
	}
)

func (x WooglesError) Enum() *WooglesError {
	p := new(WooglesError)
	*p = x
	return p
}

func (x WooglesError) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (WooglesError) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_ipc_errors_proto_enumTypes[0].Descriptor()
}

func (WooglesError) Type() protoreflect.EnumType {
	return &file_proto_ipc_errors_proto_enumTypes[0]
}

func (x WooglesError) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use WooglesError.Descriptor instead.
func (WooglesError) EnumDescriptor() ([]byte, []int) {
	return file_proto_ipc_errors_proto_rawDescGZIP(), []int{0}
}

type ErrorMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ErrorMessage) Reset() {
	*x = ErrorMessage{}
	mi := &file_proto_ipc_errors_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ErrorMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ErrorMessage) ProtoMessage() {}

func (x *ErrorMessage) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ipc_errors_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ErrorMessage.ProtoReflect.Descriptor instead.
func (*ErrorMessage) Descriptor() ([]byte, []int) {
	return file_proto_ipc_errors_proto_rawDescGZIP(), []int{0}
}

func (x *ErrorMessage) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_proto_ipc_errors_proto protoreflect.FileDescriptor

const file_proto_ipc_errors_proto_rawDesc = "" +
	"\n" +
	"\x16proto/ipc/errors.proto\x12\x03ipc\"(\n" +
	"\fErrorMessage\x12\x18\n" +
	"\amessage\x18\x01 \x01(\tR\amessage*\xf4\x1e\n" +
	"\fWooglesError\x12\v\n" +
	"\aDEFAULT\x10\x00\x12*\n" +
	"%TOURNAMENT_NEGATIVE_MAX_BYE_PLACEMENT\x10\xe9\a\x12&\n" +
	"!TOURNAMENT_NEGATIVE_MIN_PLACEMENT\x10\xea\a\x12&\n" +
	"!TOURNAMENT_NEGATIVE_GIBSON_SPREAD\x10\xeb\a\x12$\n" +
	"\x1fTOURNAMENT_EMPTY_ROUND_CONTROLS\x10\xec\a\x12.\n" +
	")TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START\x10\xed\a\x12(\n" +
	"#TOURNAMENT_ELIMINATION_PAIRINGS_MIX\x10\xee\a\x12,\n" +
	"'TOURNAMENT_DISCONTINUOUS_INITIAL_FONTES\x10\xef\a\x12-\n" +
	"(TOURNAMENT_INVALID_INITIAL_FONTES_ROUNDS\x10\xf0\a\x12+\n" +
	"&TOURNAMENT_INVALID_ELIMINATION_PLAYERS\x10\xf1\a\x12)\n" +
	"$TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE\x10\xf2\a\x12\"\n" +
	"\x1dTOURNAMENT_NONEXISTENT_PLAYER\x10\xf3\a\x12(\n" +
	"#TOURNAMENT_NONAMENDMENT_PAST_RESULT\x10\xf4\a\x12$\n" +
	"\x1fTOURNAMENT_FUTURE_NONBYE_RESULT\x10\xf5\a\x12\"\n" +
	"\x1dTOURNAMENT_NIL_PLAYER_PAIRING\x10\xf6\a\x12\x1c\n" +
	"\x17TOURNAMENT_NONOPPONENTS\x10\xf7\a\x12.\n" +
	")TOURNAMENT_MIXED_VOID_AND_NONVOID_RESULTS\x10\xf8\a\x12#\n" +
	"\x1eTOURNAMENT_NONEXISTENT_PAIRING\x10\xf9\a\x12#\n" +
	"\x1eTOURNAMENT_UNINITIALIZED_GAMES\x10\xfa\a\x12+\n" +
	"&TOURNAMENT_TIEBREAK_INVALID_GAME_INDEX\x10\xfb\a\x12'\n" +
	"\"TOURNAMENT_GAME_INDEX_OUT_OF_RANGE\x10\xfc\a\x12(\n" +
	"#TOURNAMENT_RESULT_ALREADY_SUBMITTED\x10\xfd\a\x12,\n" +
	"'TOURNAMENT_NONEXISTENT_RESULT_AMENDMENT\x10\xfe\a\x12 \n" +
	"\x1bTOURNAMENT_GIBSON_CAN_CATCH\x10\xff\a\x12!\n" +
	"\x1cTOURNAMENT_CANNOT_ASSIGN_BYE\x10\x80\b\x12'\n" +
	"\"TOURNAMENT_INTERNAL_BYE_ASSIGNMENT\x10\x81\b\x12)\n" +
	"$TOURNAMENT_INCORRECT_PAIRINGS_LENGTH\x10\x82\b\x12%\n" +
	" TOURNAMENT_PAIRINGS_ASSIGNED_BYE\x10\x83\b\x12*\n" +
	"%TOURNAMENT_SUSPENDED_PLAYER_UNREMOVED\x10\x84\b\x12*\n" +
	"%TOURNAMENT_PAIRING_INDEX_OUT_OF_RANGE\x10\x85\b\x12'\n" +
	"\"TOURNAMENT_SUSPENDED_PLAYER_PAIRED\x10\x86\b\x12!\n" +
	"\x1cTOURNAMENT_PLAYER_NOT_PAIRED\x10\x87\b\x12%\n" +
	" TOURNAMENT_PLAYER_ALREADY_EXISTS\x10\x88\b\x12&\n" +
	"!TOURNAMENT_ADD_PLAYERS_LAST_ROUND\x10\x89\b\x12)\n" +
	"$TOURNAMENT_PLAYER_INDEX_OUT_OF_RANGE\x10\x8a\b\x12&\n" +
	"!TOURNAMENT_PLAYER_ALREADY_REMOVED\x10\x8b\b\x12.\n" +
	")TOURNAMENT_REMOVAL_CREATES_EMPTY_DIVISION\x10\x8c\b\x12%\n" +
	" TOURNAMENT_NEGATIVE_GIBSON_ROUND\x10\x8d\b\x12\"\n" +
	"\x1dTOURNAMENT_ROUND_NOT_COMPLETE\x10\x8e\b\x12\x18\n" +
	"\x13TOURNAMENT_FINISHED\x10\x8f\b\x12\x1d\n" +
	"\x18TOURNAMENT_NOT_STARTABLE\x10\x90\b\x12\x1f\n" +
	"\x1aTOURNAMENT_ROUND_NOT_READY\x10\x91\b\x12%\n" +
	" TOURNAMENT_SET_GAME_ROUND_NUMBER\x10\x92\b\x12\x1d\n" +
	"\x18TOURNAMENT_ALREADY_READY\x10\x93\b\x12&\n" +
	"!TOURNAMENT_SET_READY_MULTIPLE_IDS\x10\x94\b\x12*\n" +
	"%TOURNAMENT_SET_READY_PLAYER_NOT_FOUND\x10\x95\b\x12\x18\n" +
	"\x13TOURNAMENT_NO_LOSER\x10\x96\b\x12\x19\n" +
	"\x14TOURNAMENT_NO_WINNER\x10\x97\b\x12\x1f\n" +
	"\x1aTOURNAMENT_UNPAIRED_PLAYER\x10\x98\b\x12\x1f\n" +
	"\x1aTOURNAMENT_INVALID_PAIRING\x10\x99\b\x12\x1d\n" +
	"\x18TOURNAMENT_INVALID_SWISS\x10\x9a\b\x12$\n" +
	"\x1fTOURNAMENT_ZERO_GAMES_PER_ROUND\x10\x9b\b\x12\x1a\n" +
	"\x15TOURNAMENT_EMPTY_NAME\x10\x9c\b\x12\x1b\n" +
	"\x16TOURNAMENT_NOT_STARTED\x10\x9d\b\x12$\n" +
	"\x1fTOURNAMENT_NONEXISTENT_DIVISION\x10\x9e\b\x12$\n" +
	"\x1fTOURNAMENT_NIL_DIVISION_MANAGER\x10\x9f\b\x12-\n" +
	"(TOURNAMENT_SET_NON_FUTURE_ROUND_CONTROLS\x10\xa0\b\x12(\n" +
	"#TOURNAMENT_ADD_DIVISION_AFTER_START\x10\xa1\b\x12%\n" +
	" TOURNAMENT_INVALID_DIVISION_NAME\x10\xa2\b\x12'\n" +
	"\"TOURNAMENT_DIVISION_ALREADY_EXISTS\x10\xa3\b\x12,\n" +
	"'TOURNAMENT_DIVISION_REMOVAL_AFTER_START\x10\xa4\b\x121\n" +
	",TOURNAMENT_DIVISION_REMOVAL_EXISTING_PLAYERS\x10\xa5\b\x12&\n" +
	"!TOURNAMENT_PLAYER_ID_CONSTRUCTION\x10\xa6\b\x12)\n" +
	"$TOURNAMENT_EXECUTIVE_DIRECTOR_EXISTS\x10\xa7\b\x12\x1f\n" +
	"\x1aTOURNAMENT_DIRECTOR_EXISTS\x10\xa8\b\x12\x1c\n" +
	"\x17TOURNAMENT_NO_DIVISIONS\x10\xa9\b\x12%\n" +
	" TOURNAMENT_GAME_CONTROLS_NOT_SET\x10\xaa\b\x12%\n" +
	" TOURNAMENT_INCORRECT_START_ROUND\x10\xab\b\x12%\n" +
	" TOURNAMENT_PAIR_NON_FUTURE_ROUND\x10\xac\b\x12'\n" +
	"\"TOURNAMENT_DELETE_NON_FUTURE_ROUND\x10\xad\b\x12%\n" +
	" TOURNAMENT_DIVISION_NOT_FINISHED\x10\xae\b\x122\n" +
	"-TOURNAMENT_NOT_EXACTLY_ONE_EXECUTIVE_DIRECTOR\x10\xaf\b\x12*\n" +
	"%TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL\x10\xb0\b\x12%\n" +
	" TOURNAMENT_INVALID_FUTURE_RESULT\x10\xb1\b\x12)\n" +
	"$TOURNAMENT_SCHEDULED_START_AFTER_END\x10\xc2\b\x12\x1c\n" +
	"\x17TOURNAMENT_NOT_FINISHED\x10\xc3\b\x12(\n" +
	"#TOURNAMENT_OPENCHECKINS_AFTER_START\x10\xc4\b\x12\x1f\n" +
	"\x1aTOURNAMENT_CHECKINS_CLOSED\x10\xc5\b\x12\x1e\n" +
	"\x19TOURNAMENT_NOT_REGISTERED\x10\xc6\b\x12$\n" +
	"\x1fTOURNAMENT_REGISTRATIONS_CLOSED\x10\xc7\b\x12\x1f\n" +
	"\x1aTOURNAMENT_ALREADY_STARTED\x10\xc8\b\x12-\n" +
	"(TOURNAMENT_OPENREGISTRATIONS_AFTER_START\x10\xc9\b\x12;\n" +
	"6TOURNAMENT_CANNOT_START_CHECKINS_OR_REGISTRATIONS_OPEN\x10\xca\b\x12;\n" +
	"6TOURNAMENT_CANNOT_REMOVE_UNCHECKED_IN_IF_CHECKINS_OPEN\x10\xcb\b\x12\x18\n" +
	"\x13PUZZLE_VOTE_INVALID\x10\xb2\b\x12*\n" +
	"%PUZZLE_GET_RANDOM_PUZZLE_ID_NOT_FOUND\x10\xb3\b\x12'\n" +
	"\"PUZZLE_GET_RANDOM_PUZZLE_NOT_FOUND\x10\xb4\b\x12%\n" +
	" PUZZLE_GET_PUZZLE_UUID_NOT_FOUND\x10\xb5\b\x12+\n" +
	"&PUZZLE_GET_PREVIOUS_PUZZLE_NO_ATTEMPTS\x10\xb6\b\x121\n" +
	",PUZZLE_GET_PREVIOUS_PUZZLE_ATTEMPT_NOT_FOUND\x10\xb7\b\x12,\n" +
	"'PUZZLE_GET_ANSWER_PUZZLE_UUID_NOT_FOUND\x10\xb8\b\x12-\n" +
	"(PUZZLE_SUBMIT_ANSWER_PUZZLE_ID_NOT_FOUND\x10\xb9\b\x12%\n" +
	" PUZZLE_SUBMIT_ANSWER_SET_CORRECT\x10\xba\b\x12&\n" +
	"!PUZZLE_SUBMIT_ANSWER_SET_ATTEMPTS\x10\xbb\b\x12(\n" +
	"#PUZZLE_SET_PUZZLE_VOTE_ID_NOT_FOUND\x10\xbc\b\x122\n" +
	"-PUZZLE_SUBMIT_ANSWER_PUZZLE_ATTEMPT_NOT_FOUND\x10\xbd\b\x12%\n" +
	" PUZZLE_GET_PUZZLE_UPDATE_ATTEMPT\x10\xbe\b\x12$\n" +
	"\x1fPUZZLE_GET_ANSWER_NOT_YET_RATED\x10\xbf\b\x12\x1a\n" +
	"\x15USER_UPDATE_NOT_FOUND\x10\xc0\b\x12\x1d\n" +
	"\x18GAME_NO_LONGER_AVAILABLE\x10\xc1\bBs\n" +
	"\acom.ipcB\vErrorsProtoP\x01Z/github.com/woogles-io/liwords/rpc/api/proto/ipc\xa2\x02\x03IXX\xaa\x02\x03Ipc\xca\x02\x03Ipc\xe2\x02\x0fIpc\\GPBMetadata\xea\x02\x03Ipcb\x06proto3"

var (
	file_proto_ipc_errors_proto_rawDescOnce sync.Once
	file_proto_ipc_errors_proto_rawDescData []byte
)

func file_proto_ipc_errors_proto_rawDescGZIP() []byte {
	file_proto_ipc_errors_proto_rawDescOnce.Do(func() {
		file_proto_ipc_errors_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_ipc_errors_proto_rawDesc), len(file_proto_ipc_errors_proto_rawDesc)))
	})
	return file_proto_ipc_errors_proto_rawDescData
}

var file_proto_ipc_errors_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_ipc_errors_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proto_ipc_errors_proto_goTypes = []any{
	(WooglesError)(0),    // 0: ipc.WooglesError
	(*ErrorMessage)(nil), // 1: ipc.ErrorMessage
}
var file_proto_ipc_errors_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_ipc_errors_proto_init() }
func file_proto_ipc_errors_proto_init() {
	if File_proto_ipc_errors_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_ipc_errors_proto_rawDesc), len(file_proto_ipc_errors_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_ipc_errors_proto_goTypes,
		DependencyIndexes: file_proto_ipc_errors_proto_depIdxs,
		EnumInfos:         file_proto_ipc_errors_proto_enumTypes,
		MessageInfos:      file_proto_ipc_errors_proto_msgTypes,
	}.Build()
	File_proto_ipc_errors_proto = out.File
	file_proto_ipc_errors_proto_goTypes = nil
	file_proto_ipc_errors_proto_depIdxs = nil
}
