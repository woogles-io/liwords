// package: liwords
// file: api/proto/realtime/realtime.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";

export class GameRules extends jspb.Message {
  getBoardLayoutName(): string;
  setBoardLayoutName(value: string): void;

  getLetterDistributionName(): string;
  setLetterDistributionName(value: string): void;

  getVariantName(): string;
  setVariantName(value: string): void;

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
    variantName: string,
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

  getChallengeRule(): macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap];
  setChallengeRule(value: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap]): void;

  getGameMode(): GameModeMap[keyof GameModeMap];
  setGameMode(value: GameModeMap[keyof GameModeMap]): void;

  getRatingMode(): RatingModeMap[keyof RatingModeMap];
  setRatingMode(value: RatingModeMap[keyof RatingModeMap]): void;

  getRequestId(): string;
  setRequestId(value: string): void;

  getMaxOvertimeMinutes(): number;
  setMaxOvertimeMinutes(value: number): void;

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
    challengeRule: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap],
    gameMode: GameModeMap[keyof GameModeMap],
    ratingMode: RatingModeMap[keyof RatingModeMap],
    requestId: string,
    maxOvertimeMinutes: number,
  }
}

export class RequestingUser extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  getRelevantRating(): string;
  setRelevantRating(value: string): void;

  getIsAnonymous(): boolean;
  setIsAnonymous(value: boolean): void;

  getDisplayName(): string;
  setDisplayName(value: string): void;

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
    userId: string,
    relevantRating: string,
    isAnonymous: boolean,
    displayName: string,
  }
}

export class GameMeta extends jspb.Message {
  clearUsersList(): void;
  getUsersList(): Array<GameMeta.UserMeta>;
  setUsersList(value: Array<GameMeta.UserMeta>): void;
  addUsers(value?: GameMeta.UserMeta, index?: number): GameMeta.UserMeta;

  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameMeta.AsObject;
  static toObject(includeInstance: boolean, msg: GameMeta): GameMeta.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameMeta, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameMeta;
  static deserializeBinaryFromReader(message: GameMeta, reader: jspb.BinaryReader): GameMeta;
}

export namespace GameMeta {
  export type AsObject = {
    usersList: Array<GameMeta.UserMeta.AsObject>,
    gameRequest?: GameRequest.AsObject,
  }

  export class UserMeta extends jspb.Message {
    getRelevantRating(): string;
    setRelevantRating(value: string): void;

    getDisplayName(): string;
    setDisplayName(value: string): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): UserMeta.AsObject;
    static toObject(includeInstance: boolean, msg: UserMeta): UserMeta.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: UserMeta, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): UserMeta;
    static deserializeBinaryFromReader(message: UserMeta, reader: jspb.BinaryReader): UserMeta;
  }

  export namespace UserMeta {
    export type AsObject = {
      relevantRating: string,
      displayName: string,
    }
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

export class SeekRequests extends jspb.Message {
  clearRequestsList(): void;
  getRequestsList(): Array<SeekRequest>;
  setRequestsList(value: Array<SeekRequest>): void;
  addRequests(value?: SeekRequest, index?: number): SeekRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SeekRequests.AsObject;
  static toObject(includeInstance: boolean, msg: SeekRequests): SeekRequests.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SeekRequests, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SeekRequests;
  static deserializeBinaryFromReader(message: SeekRequests, reader: jspb.BinaryReader): SeekRequests;
}

export namespace SeekRequests {
  export type AsObject = {
    requestsList: Array<SeekRequest.AsObject>,
  }
}

export class ActiveGames extends jspb.Message {
  clearGamesList(): void;
  getGamesList(): Array<GameMeta>;
  setGamesList(value: Array<GameMeta>): void;
  addGames(value?: GameMeta, index?: number): GameMeta;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ActiveGames.AsObject;
  static toObject(includeInstance: boolean, msg: ActiveGames): ActiveGames.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ActiveGames, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ActiveGames;
  static deserializeBinaryFromReader(message: ActiveGames, reader: jspb.BinaryReader): ActiveGames;
}

export namespace ActiveGames {
  export type AsObject = {
    gamesList: Array<GameMeta.AsObject>,
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

  getPlaying(): macondo_api_proto_macondo_macondo_pb.PlayStateMap[keyof macondo_api_proto_macondo_macondo_pb.PlayStateMap];
  setPlaying(value: macondo_api_proto_macondo_macondo_pb.PlayStateMap[keyof macondo_api_proto_macondo_macondo_pb.PlayStateMap]): void;

  getUserId(): string;
  setUserId(value: string): void;

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
    playing: macondo_api_proto_macondo_macondo_pb.PlayStateMap[keyof macondo_api_proto_macondo_macondo_pb.PlayStateMap],
    userId: string,
  }
}

export class ServerChallengeResultEvent extends jspb.Message {
  getValid(): boolean;
  setValid(value: boolean): void;

  getChallenger(): string;
  setChallenger(value: string): void;

  getChallengeRule(): macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap];
  setChallengeRule(value: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServerChallengeResultEvent.AsObject;
  static toObject(includeInstance: boolean, msg: ServerChallengeResultEvent): ServerChallengeResultEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServerChallengeResultEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServerChallengeResultEvent;
  static deserializeBinaryFromReader(message: ServerChallengeResultEvent, reader: jspb.BinaryReader): ServerChallengeResultEvent;
}

export namespace ServerChallengeResultEvent {
  export type AsObject = {
    valid: boolean,
    challenger: string,
    challengeRule: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap],
  }
}

export class GameEndedEvent extends jspb.Message {
  getScoresMap(): jspb.Map<string, number>;
  clearScoresMap(): void;
  getNewRatingsMap(): jspb.Map<string, number>;
  clearNewRatingsMap(): void;
  getEndReason(): GameEndReasonMap[keyof GameEndReasonMap];
  setEndReason(value: GameEndReasonMap[keyof GameEndReasonMap]): void;

  getWinner(): string;
  setWinner(value: string): void;

  getLoser(): string;
  setLoser(value: string): void;

  getTie(): boolean;
  setTie(value: boolean): void;

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
    scoresMap: Array<[string, number]>,
    newRatingsMap: Array<[string, number]>,
    endReason: GameEndReasonMap[keyof GameEndReasonMap],
    winner: string,
    loser: string,
    tie: boolean,
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

export class TimedOut extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TimedOut.AsObject;
  static toObject(includeInstance: boolean, msg: TimedOut): TimedOut.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TimedOut, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TimedOut;
  static deserializeBinaryFromReader(message: TimedOut, reader: jspb.BinaryReader): TimedOut;
}

export namespace TimedOut {
  export type AsObject = {
    gameId: string,
    userId: string,
  }
}

export class TokenSocketLogin extends jspb.Message {
  getToken(): string;
  setToken(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TokenSocketLogin.AsObject;
  static toObject(includeInstance: boolean, msg: TokenSocketLogin): TokenSocketLogin.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TokenSocketLogin, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TokenSocketLogin;
  static deserializeBinaryFromReader(message: TokenSocketLogin, reader: jspb.BinaryReader): TokenSocketLogin;
}

export namespace TokenSocketLogin {
  export type AsObject = {
    token: string,
  }
}

export class JoinPath extends jspb.Message {
  getPath(): string;
  setPath(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): JoinPath.AsObject;
  static toObject(includeInstance: boolean, msg: JoinPath): JoinPath.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: JoinPath, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): JoinPath;
  static deserializeBinaryFromReader(message: JoinPath, reader: jspb.BinaryReader): JoinPath;
}

export namespace JoinPath {
  export type AsObject = {
    path: string,
  }
}

export class UnjoinRealm extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnjoinRealm.AsObject;
  static toObject(includeInstance: boolean, msg: UnjoinRealm): UnjoinRealm.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UnjoinRealm, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnjoinRealm;
  static deserializeBinaryFromReader(message: UnjoinRealm, reader: jspb.BinaryReader): UnjoinRealm;
}

export namespace UnjoinRealm {
  export type AsObject = {
  }
}

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
  SERVER_CHALLENGE_RESULT_EVENT: 9;
  SEEK_REQUESTS: 10;
  JOIN_PATH: 11;
  UNJOIN_REALM: 12;
  TIMED_OUT: 13;
  TOKEN_SOCKET_LOGIN: 14;
}

export const MessageType: MessageTypeMap;

export interface GameEndReasonMap {
  NONE: 0;
  TIME: 1;
  STANDARD: 2;
  CONSECUTIVE_ZEROES: 3;
  RESIGNED: 4;
  ABANDONED: 5;
}

export const GameEndReason: GameEndReasonMap;

