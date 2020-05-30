// package: crosswords
// file: api/proto/game_service.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../macondo/api/proto/macondo/macondo_pb";

export class GameRules extends jspb.Message {
  getBoardLayoutName(): string;
  setBoardLayoutName(value: string): void;

  getLetterDistributionName(): string;
  setLetterDistributionName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameRules.AsObject;
  static toObject(includeInstance: boolean, msg: GameRules): GameRules.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameRules, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameRules;
  static deserializeBinaryFromReader(message: GameRules, reader: jspb.BinaryReader): GameRules;
}

export namespace GameRules {
  export type AsObject = {
    boardLayoutName: string,
    letterDistributionName: string,
  }
}

export class GameRequest extends jspb.Message {
  getLexicon(): string;
  setLexicon(value: string): void;

  hasRules(): boolean;
  clearRules(): void;
  getRules(): GameRules | undefined;
  setRules(value?: GameRules): void;

  getInitialTimeSeconds(): number;
  setInitialTimeSeconds(value: number): void;

  getIncrementSeconds(): number;
  setIncrementSeconds(value: number): void;

  getChallengeRule(): ChallengeRuleMap[keyof ChallengeRuleMap];
  setChallengeRule(value: ChallengeRuleMap[keyof ChallengeRuleMap]): void;

  getGameMode(): GameModeMap[keyof GameModeMap];
  setGameMode(value: GameModeMap[keyof GameModeMap]): void;

  getRatingMode(): RatingModeMap[keyof RatingModeMap];
  setRatingMode(value: RatingModeMap[keyof RatingModeMap]): void;

  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GameRequest): GameRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameRequest;
  static deserializeBinaryFromReader(message: GameRequest, reader: jspb.BinaryReader): GameRequest;
}

export namespace GameRequest {
  export type AsObject = {
    lexicon: string,
    rules?: GameRules.AsObject,
    initialTimeSeconds: number,
    incrementSeconds: number,
    challengeRule: ChallengeRuleMap[keyof ChallengeRuleMap],
    gameMode: GameModeMap[keyof GameModeMap],
    ratingMode: RatingModeMap[keyof RatingModeMap],
    requestId: string,
  }
}

export class RequestingUser extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getRelevantrating(): number;
  setRelevantrating(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RequestingUser.AsObject;
  static toObject(includeInstance: boolean, msg: RequestingUser): RequestingUser.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RequestingUser, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RequestingUser;
  static deserializeBinaryFromReader(message: RequestingUser, reader: jspb.BinaryReader): RequestingUser;
}

export namespace RequestingUser {
  export type AsObject = {
    username: string,
    relevantrating: number,
  }
}

export class SeekRequest extends jspb.Message {
  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): RequestingUser | undefined;
  setUser(value?: RequestingUser): void;

  getMinimumRating(): number;
  setMinimumRating(value: number): void;

  getMaximumRating(): number;
  setMaximumRating(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SeekRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SeekRequest): SeekRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SeekRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SeekRequest;
  static deserializeBinaryFromReader(message: SeekRequest, reader: jspb.BinaryReader): SeekRequest;
}

export namespace SeekRequest {
  export type AsObject = {
    gameRequest?: GameRequest.AsObject,
    user?: RequestingUser.AsObject,
    minimumRating: number,
    maximumRating: number,
  }
}

export class MatchRequest extends jspb.Message {
  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): RequestingUser | undefined;
  setUser(value?: RequestingUser): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MatchRequest.AsObject;
  static toObject(includeInstance: boolean, msg: MatchRequest): MatchRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MatchRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MatchRequest;
  static deserializeBinaryFromReader(message: MatchRequest, reader: jspb.BinaryReader): MatchRequest;
}

export namespace MatchRequest {
  export type AsObject = {
    gameRequest?: GameRequest.AsObject,
    user?: RequestingUser.AsObject,
  }
}

export class ServerGameplayEvent extends jspb.Message {
  hasEvent(): boolean;
  clearEvent(): void;
  getEvent(): macondo_api_proto_macondo_macondo_pb.GameEvent | undefined;
  setEvent(value?: macondo_api_proto_macondo_macondo_pb.GameEvent): void;

  getGameId(): string;
  setGameId(value: string): void;

  getNewRack(): string;
  setNewRack(value: string): void;

  getTimeRemaining(): number;
  setTimeRemaining(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServerGameplayEvent.AsObject;
  static toObject(includeInstance: boolean, msg: ServerGameplayEvent): ServerGameplayEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServerGameplayEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServerGameplayEvent;
  static deserializeBinaryFromReader(message: ServerGameplayEvent, reader: jspb.BinaryReader): ServerGameplayEvent;
}

export namespace ServerGameplayEvent {
  export type AsObject = {
    event?: macondo_api_proto_macondo_macondo_pb.GameEvent.AsObject,
    gameId: string,
    newRack: string,
    timeRemaining: number,
  }
}

export class GameEndedEvent extends jspb.Message {
  getReason(): GameEndReasonMap[keyof GameEndReasonMap];
  setReason(value: GameEndReasonMap[keyof GameEndReasonMap]): void;

  getAffectedPlayer(): string;
  setAffectedPlayer(value: string): void;

  getNewRatingsMap(): jspb.Map<string, number>;
  clearNewRatingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameEndedEvent.AsObject;
  static toObject(includeInstance: boolean, msg: GameEndedEvent): GameEndedEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameEndedEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameEndedEvent;
  static deserializeBinaryFromReader(message: GameEndedEvent, reader: jspb.BinaryReader): GameEndedEvent;
}

export namespace GameEndedEvent {
  export type AsObject = {
    reason: GameEndReasonMap[keyof GameEndReasonMap],
    affectedPlayer: string,
    newRatingsMap: Array<[string, number]>,
  }
}

export class GameHistoryRefresher extends jspb.Message {
  hasHistory(): boolean;
  clearHistory(): void;
  getHistory(): macondo_api_proto_macondo_macondo_pb.GameHistory | undefined;
  setHistory(value?: macondo_api_proto_macondo_macondo_pb.GameHistory): void;

  getTimePlayer1(): number;
  setTimePlayer1(value: number): void;

  getTimePlayer2(): number;
  setTimePlayer2(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameHistoryRefresher.AsObject;
  static toObject(includeInstance: boolean, msg: GameHistoryRefresher): GameHistoryRefresher.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameHistoryRefresher, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameHistoryRefresher;
  static deserializeBinaryFromReader(message: GameHistoryRefresher, reader: jspb.BinaryReader): GameHistoryRefresher;
}

export namespace GameHistoryRefresher {
  export type AsObject = {
    history?: macondo_api_proto_macondo_macondo_pb.GameHistory.AsObject,
    timePlayer1: number,
    timePlayer2: number,
  }
}

export class NewGameEvent extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NewGameEvent.AsObject;
  static toObject(includeInstance: boolean, msg: NewGameEvent): NewGameEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NewGameEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NewGameEvent;
  static deserializeBinaryFromReader(message: NewGameEvent, reader: jspb.BinaryReader): NewGameEvent;
}

export namespace NewGameEvent {
  export type AsObject = {
    gameId: string,
  }
}

export class ErrorMessage extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ErrorMessage.AsObject;
  static toObject(includeInstance: boolean, msg: ErrorMessage): ErrorMessage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ErrorMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ErrorMessage;
  static deserializeBinaryFromReader(message: ErrorMessage, reader: jspb.BinaryReader): ErrorMessage;
}

export namespace ErrorMessage {
  export type AsObject = {
    message: string,
  }
}

export class GameAcceptedEvent extends jspb.Message {
  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameAcceptedEvent.AsObject;
  static toObject(includeInstance: boolean, msg: GameAcceptedEvent): GameAcceptedEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameAcceptedEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameAcceptedEvent;
  static deserializeBinaryFromReader(message: GameAcceptedEvent, reader: jspb.BinaryReader): GameAcceptedEvent;
}

export namespace GameAcceptedEvent {
  export type AsObject = {
    requestId: string,
  }
}

export class ClientGameplayEvent extends jspb.Message {
  getType(): ClientGameplayEvent.EventTypeMap[keyof ClientGameplayEvent.EventTypeMap];
  setType(value: ClientGameplayEvent.EventTypeMap[keyof ClientGameplayEvent.EventTypeMap]): void;

  getGameId(): string;
  setGameId(value: string): void;

  getPositionCoords(): string;
  setPositionCoords(value: string): void;

  getTiles(): string;
  setTiles(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClientGameplayEvent.AsObject;
  static toObject(includeInstance: boolean, msg: ClientGameplayEvent): ClientGameplayEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClientGameplayEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClientGameplayEvent;
  static deserializeBinaryFromReader(message: ClientGameplayEvent, reader: jspb.BinaryReader): ClientGameplayEvent;
}

export namespace ClientGameplayEvent {
  export type AsObject = {
    type: ClientGameplayEvent.EventTypeMap[keyof ClientGameplayEvent.EventTypeMap],
    gameId: string,
    positionCoords: string,
    tiles: string,
  }

  export interface EventTypeMap {
    TILE_PLACEMENT: 0;
    PASS: 1;
    EXCHANGE: 2;
    CHALLENGE_PLAY: 3;
  }

  export const EventType: EventTypeMap;
}

export interface ChallengeRuleMap {
  VOID: 0;
  SINGLE: 1;
  DOUBLE: 2;
  FIVE_POINT: 3;
  TEN_POINT: 4;
}

export const ChallengeRule: ChallengeRuleMap;

export interface GameModeMap {
  REAL_TIME: 0;
  CORRESPONDENCE: 1;
}

export const GameMode: GameModeMap;

export interface RatingModeMap {
  RATED: 0;
  CASUAL: 1;
}

export const RatingMode: RatingModeMap;

export interface MessageTypeMap {
  SEEK_REQUEST: 0;
  MATCH_REQUEST: 1;
  GAME_ACCEPTED_EVENT: 2;
  CLIENT_GAMEPLAY_EVENT: 3;
  SERVER_GAMEPLAY_EVENT: 4;
  GAME_ENDED_EVENT: 5;
  GAME_HISTORY_REFRESHER: 6;
  ERROR_MESSAGE: 7;
  NEW_GAME_EVENT: 8;
}

export const MessageType: MessageTypeMap;

export interface GameEndReasonMap {
  TIME: 0;
  WENT_OUT: 1;
  CONSECUTIVE_ZEROES: 2;
}

export const GameEndReason: GameEndReasonMap;

