// package: game_service
// file: api/proto/game_service/game_service.proto

import * as jspb from "google-protobuf";
import * as api_proto_realtime_realtime_pb from "../../../api/proto/realtime/realtime_pb";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";

export class GameInfoRequest extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameInfoRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GameInfoRequest): GameInfoRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameInfoRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameInfoRequest;
  static deserializeBinaryFromReader(message: GameInfoRequest, reader: jspb.BinaryReader): GameInfoRequest;
}

export namespace GameInfoRequest {
  export type AsObject = {
    gameId: string,
  }
}

export class PlayerInfo extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  getNickname(): string;
  setNickname(value: string): void;

  getFullName(): string;
  setFullName(value: string): void;

  getCountryCode(): string;
  setCountryCode(value: string): void;

  getRating(): string;
  setRating(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getAvatarUrl(): string;
  setAvatarUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PlayerInfo.AsObject;
  static toObject(includeInstance: boolean, msg: PlayerInfo): PlayerInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PlayerInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PlayerInfo;
  static deserializeBinaryFromReader(message: PlayerInfo, reader: jspb.BinaryReader): PlayerInfo;
}

export namespace PlayerInfo {
  export type AsObject = {
    userId: string,
    nickname: string,
    fullName: string,
    countryCode: string,
    rating: string,
    title: string,
    avatarUrl: string,
  }
}

export class GameInfoResponse extends jspb.Message {
  clearPlayersList(): void;
  getPlayersList(): Array<PlayerInfo>;
  setPlayersList(value: Array<PlayerInfo>): void;
  addPlayers(value?: PlayerInfo, index?: number): PlayerInfo;

  getLexicon(): string;
  setLexicon(value: string): void;

  getVariant(): string;
  setVariant(value: string): void;

  getTimeControlName(): string;
  setTimeControlName(value: string): void;

  getInitialTimeSeconds(): number;
  setInitialTimeSeconds(value: number): void;

  getTournamentName(): string;
  setTournamentName(value: string): void;

  getChallengeRule(): macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap];
  setChallengeRule(value: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap]): void;

  getRatingMode(): api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap];
  setRatingMode(value: api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap]): void;

  getDone(): boolean;
  setDone(value: boolean): void;

  getMaxOvertimeMinutes(): number;
  setMaxOvertimeMinutes(value: number): void;

  getGameEndReason(): api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap];
  setGameEndReason(value: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap]): void;

  getIncrementSeconds(): number;
  setIncrementSeconds(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GameInfoResponse): GameInfoResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameInfoResponse;
  static deserializeBinaryFromReader(message: GameInfoResponse, reader: jspb.BinaryReader): GameInfoResponse;
}

export namespace GameInfoResponse {
  export type AsObject = {
    playersList: Array<PlayerInfo.AsObject>,
    lexicon: string,
    variant: string,
    timeControlName: string,
    initialTimeSeconds: number,
    tournamentName: string,
    challengeRule: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap],
    ratingMode: api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap],
    done: boolean,
    maxOvertimeMinutes: number,
    gameEndReason: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap],
    incrementSeconds: number,
  }
}

export class GCGRequest extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GCGRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GCGRequest): GCGRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GCGRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GCGRequest;
  static deserializeBinaryFromReader(message: GCGRequest, reader: jspb.BinaryReader): GCGRequest;
}

export namespace GCGRequest {
  export type AsObject = {
    gameId: string,
  }
}

export class GCGResponse extends jspb.Message {
  getGcg(): string;
  setGcg(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GCGResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GCGResponse): GCGResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GCGResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GCGResponse;
  static deserializeBinaryFromReader(message: GCGResponse, reader: jspb.BinaryReader): GCGResponse;
}

export namespace GCGResponse {
  export type AsObject = {
    gcg: string,
  }
}

