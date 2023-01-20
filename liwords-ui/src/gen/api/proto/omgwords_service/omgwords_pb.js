// @generated by protoc-gen-es v0.2.1
// @generated from file omgwords_service/omgwords.proto (package omgwords_service, syntax proto3)
/* eslint-disable */
/* @ts-nocheck */

import {proto3, Timestamp} from "@bufbuild/protobuf";
import {ChallengeRule, ClientGameplayEvent, GameDocument, GameRules, PlayerInfo} from "../ipc/omgwords_pb.js";

/**
 * GameEventResponse doesn't need to have any extra data. The GameEvent API
 * will still use sockets to broadcast game information.
 *
 * @generated from message omgwords_service.GameEventResponse
 */
export const GameEventResponse = proto3.makeMessageType(
  "omgwords_service.GameEventResponse",
  [],
);

/**
 * @generated from message omgwords_service.TimePenaltyEvent
 */
export const TimePenaltyEvent = proto3.makeMessageType(
  "omgwords_service.TimePenaltyEvent",
  () => [
    { no: 1, name: "points_lost", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
  ],
);

/**
 * @generated from message omgwords_service.ChallengeBonusPointsEvent
 */
export const ChallengeBonusPointsEvent = proto3.makeMessageType(
  "omgwords_service.ChallengeBonusPointsEvent",
  () => [
    { no: 1, name: "points_gained", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
  ],
);

/**
 * @generated from message omgwords_service.CreateBroadcastGameRequest
 */
export const CreateBroadcastGameRequest = proto3.makeMessageType(
  "omgwords_service.CreateBroadcastGameRequest",
  () => [
    { no: 1, name: "players_info", kind: "message", T: PlayerInfo, repeated: true },
    { no: 2, name: "lexicon", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "rules", kind: "message", T: GameRules },
    { no: 4, name: "challenge_rule", kind: "enum", T: proto3.getEnumType(ChallengeRule) },
    { no: 5, name: "public", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ],
);

/**
 * @generated from message omgwords_service.CreateBroadcastGameResponse
 */
export const CreateBroadcastGameResponse = proto3.makeMessageType(
  "omgwords_service.CreateBroadcastGameResponse",
  () => [
    { no: 1, name: "game_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ],
);

/**
 * @generated from message omgwords_service.BroadcastGamePrivacy
 */
export const BroadcastGamePrivacy = proto3.makeMessageType(
  "omgwords_service.BroadcastGamePrivacy",
  () => [
    { no: 1, name: "public", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ],
);

/**
 * @generated from message omgwords_service.GetGamesForEditorRequest
 */
export const GetGamesForEditorRequest = proto3.makeMessageType(
  "omgwords_service.GetGamesForEditorRequest",
  () => [
    { no: 1, name: "user_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "limit", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
    { no: 3, name: "offset", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
    { no: 4, name: "unfinished", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ],
);

/**
 * Assume we can never have so many unfinished games that we'd need limits and
 * offsets. Ideally we should only have one unfinished game per authed player at
 * a time.
 *
 * @generated from message omgwords_service.GetMyUnfinishedGamesRequest
 */
export const GetMyUnfinishedGamesRequest = proto3.makeMessageType(
  "omgwords_service.GetMyUnfinishedGamesRequest",
  [],
);

/**
 * @generated from message omgwords_service.BroadcastGamesResponse
 */
export const BroadcastGamesResponse = proto3.makeMessageType(
  "omgwords_service.BroadcastGamesResponse",
  () => [
    { no: 1, name: "games", kind: "message", T: BroadcastGamesResponse_BroadcastGame, repeated: true },
  ],
);

/**
 * @generated from message omgwords_service.BroadcastGamesResponse.BroadcastGame
 */
export const BroadcastGamesResponse_BroadcastGame = proto3.makeMessageType(
  "omgwords_service.BroadcastGamesResponse.BroadcastGame",
  () => [
    { no: 1, name: "game_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "creator_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "private", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
    { no: 4, name: "finished", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
    { no: 5, name: "players_info", kind: "message", T: PlayerInfo, repeated: true },
    { no: 6, name: "lexicon", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 7, name: "created_at", kind: "message", T: Timestamp },
  ],
  {localName: "BroadcastGamesResponse_BroadcastGame"},
);

/**
 * @generated from message omgwords_service.AnnotatedGameEvent
 */
export const AnnotatedGameEvent = proto3.makeMessageType(
  "omgwords_service.AnnotatedGameEvent",
  () => [
    { no: 1, name: "event", kind: "message", T: ClientGameplayEvent },
    { no: 2, name: "user_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "event_number", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
    { no: 4, name: "amendment", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ],
);

/**
 * @generated from message omgwords_service.GetGameDocumentRequest
 */
export const GetGameDocumentRequest = proto3.makeMessageType(
  "omgwords_service.GetGameDocumentRequest",
  () => [
    { no: 1, name: "game_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ],
);

/**
 * @generated from message omgwords_service.DeleteBroadcastGameRequest
 */
export const DeleteBroadcastGameRequest = proto3.makeMessageType(
  "omgwords_service.DeleteBroadcastGameRequest",
  () => [
    { no: 1, name: "game_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ],
);

/**
 * @generated from message omgwords_service.DeleteBroadcastGameResponse
 */
export const DeleteBroadcastGameResponse = proto3.makeMessageType(
  "omgwords_service.DeleteBroadcastGameResponse",
  [],
);

/**
 * @generated from message omgwords_service.ReplaceDocumentRequest
 */
export const ReplaceDocumentRequest = proto3.makeMessageType(
  "omgwords_service.ReplaceDocumentRequest",
  () => [
    { no: 1, name: "document", kind: "message", T: GameDocument },
  ],
);

/**
 * @generated from message omgwords_service.PatchDocumentRequest
 */
export const PatchDocumentRequest = proto3.makeMessageType(
  "omgwords_service.PatchDocumentRequest",
  () => [
    { no: 1, name: "document", kind: "message", T: GameDocument },
  ],
);

/**
 * SetRacksEvent is the event used for sending player racks.
 *
 * @generated from message omgwords_service.SetRacksEvent
 */
export const SetRacksEvent = proto3.makeMessageType(
  "omgwords_service.SetRacksEvent",
  () => [
    { no: 1, name: "game_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "racks", kind: "scalar", T: 12 /* ScalarType.BYTES */, repeated: true },
    { no: 3, name: "event_number", kind: "scalar", T: 13 /* ScalarType.UINT32 */ },
    { no: 4, name: "amendment", kind: "scalar", T: 8 /* ScalarType.BOOL */ },
  ],
);
