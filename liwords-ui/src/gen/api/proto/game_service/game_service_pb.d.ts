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

  getIsBot(): boolean;
  setIsBot(value: boolean): void;

  getFirst(): boolean;
  setFirst(value: boolean): void;

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
    isBot: boolean,
    first: boolean,
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

  getMaxOvertimeMinutes(): number;
  setMaxOvertimeMinutes(value: number): void;

  getGameEndReason(): api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap];
  setGameEndReason(value: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap]): void;

  getIncrementSeconds(): number;
  setIncrementSeconds(value: number): void;

  clearScoresList(): void;
  getScoresList(): Array<number>;
  setScoresList(value: Array<number>): void;
  addScores(value: number, index?: number): number;

  getWinner(): number;
  setWinner(value: number): void;

  getUpdatedAt(): number;
  setUpdatedAt(value: number): void;

  getGameId(): string;
  setGameId(value: string): void;

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
    maxOvertimeMinutes: number,
    gameEndReason: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap],
    incrementSeconds: number,
    scoresList: Array<number>,
    winner: number,
    updatedAt: number,
    gameId: string,
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

export class GameInfoResponses extends jspb.Message {
  clearGameInfoList(): void;
  getGameInfoList(): Array<GameInfoResponse>;
  setGameInfoList(value: Array<GameInfoResponse>): void;
  addGameInfo(value?: GameInfoResponse, index?: number): GameInfoResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameInfoResponses.AsObject;
  static toObject(includeInstance: boolean, msg: GameInfoResponses): GameInfoResponses.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameInfoResponses, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameInfoResponses;
  static deserializeBinaryFromReader(message: GameInfoResponses, reader: jspb.BinaryReader): GameInfoResponses;
}

export namespace GameInfoResponses {
  export type AsObject = {
    gameInfoList: Array<GameInfoResponse.AsObject>,
  }
}

export class RecentGamesRequest extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getNumGames(): number;
  setNumGames(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RecentGamesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RecentGamesRequest): RecentGamesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RecentGamesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RecentGamesRequest;
  static deserializeBinaryFromReader(message: RecentGamesRequest, reader: jspb.BinaryReader): RecentGamesRequest;
}

export namespace RecentGamesRequest {
  export type AsObject = {
    username: string,
    numGames: number,
    offset: number,
  }
}

export class RematchStreakRequest extends jspb.Message {
  getOriginalRequestId(): string;
  setOriginalRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RematchStreakRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RematchStreakRequest): RematchStreakRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RematchStreakRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RematchStreakRequest;
  static deserializeBinaryFromReader(message: RematchStreakRequest, reader: jspb.BinaryReader): RematchStreakRequest;
}

export namespace RematchStreakRequest {
  export type AsObject = {
    originalRequestId: string,
  }
}

