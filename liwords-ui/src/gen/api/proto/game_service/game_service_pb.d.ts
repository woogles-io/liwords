// package: game_service
// file: api/proto/game_service/game_service.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";
import * as api_proto_realtime_realtime_pb from "../../../api/proto/realtime/realtime_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

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
    isBot: boolean,
    first: boolean,
  }
}

export class GameInfoResponse extends jspb.Message {
  clearPlayersList(): void;
  getPlayersList(): Array<PlayerInfo>;
  setPlayersList(value: Array<PlayerInfo>): void;
  addPlayers(value?: PlayerInfo, index?: number): PlayerInfo;

  getTimeControlName(): string;
  setTimeControlName(value: string): void;

  getTournamentId(): string;
  setTournamentId(value: string): void;

  getGameEndReason(): api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap];
  setGameEndReason(value: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap]): void;

  clearScoresList(): void;
  getScoresList(): Array<number>;
  setScoresList(value: Array<number>): void;
  addScores(value: number, index?: number): number;

  getWinner(): number;
  setWinner(value: number): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getGameId(): string;
  setGameId(value: string): void;

  hasLastUpdate(): boolean;
  clearLastUpdate(): void;
  getLastUpdate(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastUpdate(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): api_proto_realtime_realtime_pb.GameRequest | undefined;
  setGameRequest(value?: api_proto_realtime_realtime_pb.GameRequest): void;

  getTournamentDivision(): string;
  setTournamentDivision(value: string): void;

  getTournamentRound(): number;
  setTournamentRound(value: number): void;

  getTournamentGameIndex(): number;
  setTournamentGameIndex(value: number): void;

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
    timeControlName: string,
    tournamentId: string,
    gameEndReason: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap],
    scoresList: Array<number>,
    winner: number,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    gameId: string,
    lastUpdate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    gameRequest?: api_proto_realtime_realtime_pb.GameRequest.AsObject,
    tournamentDivision: string,
    tournamentRound: number,
    tournamentGameIndex: number,
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

export class GameHistoryRequest extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameHistoryRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GameHistoryRequest): GameHistoryRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameHistoryRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameHistoryRequest;
  static deserializeBinaryFromReader(message: GameHistoryRequest, reader: jspb.BinaryReader): GameHistoryRequest;
}

export namespace GameHistoryRequest {
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

export class GameHistoryResponse extends jspb.Message {
  hasHistory(): boolean;
  clearHistory(): void;
  getHistory(): macondo_api_proto_macondo_macondo_pb.GameHistory | undefined;
  setHistory(value?: macondo_api_proto_macondo_macondo_pb.GameHistory): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameHistoryResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GameHistoryResponse): GameHistoryResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameHistoryResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameHistoryResponse;
  static deserializeBinaryFromReader(message: GameHistoryResponse, reader: jspb.BinaryReader): GameHistoryResponse;
}

export namespace GameHistoryResponse {
  export type AsObject = {
    history?: macondo_api_proto_macondo_macondo_pb.GameHistory.AsObject,
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

export class StreakInfoResponse extends jspb.Message {
  clearStreakList(): void;
  getStreakList(): Array<StreakInfoResponse.SingleGameInfo>;
  setStreakList(value: Array<StreakInfoResponse.SingleGameInfo>): void;
  addStreak(value?: StreakInfoResponse.SingleGameInfo, index?: number): StreakInfoResponse.SingleGameInfo;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StreakInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StreakInfoResponse): StreakInfoResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StreakInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StreakInfoResponse;
  static deserializeBinaryFromReader(message: StreakInfoResponse, reader: jspb.BinaryReader): StreakInfoResponse;
}

export namespace StreakInfoResponse {
  export type AsObject = {
    streakList: Array<StreakInfoResponse.SingleGameInfo.AsObject>,
  }

  export class SingleGameInfo extends jspb.Message {
    getGameId(): string;
    setGameId(value: string): void;

    clearPlayersList(): void;
    getPlayersList(): Array<string>;
    setPlayersList(value: Array<string>): void;
    addPlayers(value: string, index?: number): string;

    getWinner(): number;
    setWinner(value: number): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SingleGameInfo.AsObject;
    static toObject(includeInstance: boolean, msg: SingleGameInfo): SingleGameInfo.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: SingleGameInfo, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SingleGameInfo;
    static deserializeBinaryFromReader(message: SingleGameInfo, reader: jspb.BinaryReader): SingleGameInfo;
  }

  export namespace SingleGameInfo {
    export type AsObject = {
      gameId: string,
      playersList: Array<string>,
      winner: number,
    }
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

