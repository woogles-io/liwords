// package: ipc
// file: api/proto/ipc/omgwords.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

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
    RESIGN: 4;
  }

  export const EventType: EventTypeMap;
}

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

  getPlayerVsBot(): boolean;
  setPlayerVsBot(value: boolean): void;

  getOriginalRequestId(): string;
  setOriginalRequestId(value: string): void;

  getBotType(): macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap[keyof macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap];
  setBotType(value: macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap[keyof macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap]): void;

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
    playerVsBot: boolean,
    originalRequestId: string,
    botType: macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap[keyof macondo_api_proto_macondo_macondo_pb.BotRequest.BotCodeMap],
  }
}

export class GameMetaEvent extends jspb.Message {
  getOrigEventId(): string;
  setOrigEventId(value: string): void;

  hasTimestamp(): boolean;
  clearTimestamp(): void;
  getTimestamp(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setTimestamp(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getType(): GameMetaEvent.EventTypeMap[keyof GameMetaEvent.EventTypeMap];
  setType(value: GameMetaEvent.EventTypeMap[keyof GameMetaEvent.EventTypeMap]): void;

  getPlayerId(): string;
  setPlayerId(value: string): void;

  getGameId(): string;
  setGameId(value: string): void;

  getExpiry(): number;
  setExpiry(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameMetaEvent.AsObject;
  static toObject(includeInstance: boolean, msg: GameMetaEvent): GameMetaEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameMetaEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameMetaEvent;
  static deserializeBinaryFromReader(message: GameMetaEvent, reader: jspb.BinaryReader): GameMetaEvent;
}

export namespace GameMetaEvent {
  export type AsObject = {
    origEventId: string,
    timestamp?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    type: GameMetaEvent.EventTypeMap[keyof GameMetaEvent.EventTypeMap],
    playerId: string,
    gameId: string,
    expiry: number,
  }

  export interface EventTypeMap {
    REQUEST_ABORT: 0;
    REQUEST_ADJUDICATION: 1;
    REQUEST_UNDO: 2;
    REQUEST_ADJOURN: 3;
    ABORT_ACCEPTED: 4;
    ABORT_DENIED: 5;
    ADJUDICATION_ACCEPTED: 6;
    ADJUDICATION_DENIED: 7;
    UNDO_ACCEPTED: 8;
    UNDO_DENIED: 9;
    ADD_TIME: 10;
    TIMER_EXPIRED: 11;
  }

  export const EventType: EventTypeMap;
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

  getMaxOvertimeMinutes(): number;
  setMaxOvertimeMinutes(value: number): void;

  hasOutstandingEvent(): boolean;
  clearOutstandingEvent(): void;
  getOutstandingEvent(): GameMetaEvent | undefined;
  setOutstandingEvent(value?: GameMetaEvent): void;

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
    maxOvertimeMinutes: number,
    outstandingEvent?: GameMetaEvent.AsObject,
  }
}

export class TournamentDataForGame extends jspb.Message {
  getTid(): string;
  setTid(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDataForGame.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDataForGame): TournamentDataForGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDataForGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDataForGame;
  static deserializeBinaryFromReader(message: TournamentDataForGame, reader: jspb.BinaryReader): TournamentDataForGame;
}

export namespace TournamentDataForGame {
  export type AsObject = {
    tid: string,
    division: string,
    round: number,
    gameIndex: number,
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

  getGameEndReason(): GameEndReasonMap[keyof GameEndReasonMap];
  setGameEndReason(value: GameEndReasonMap[keyof GameEndReasonMap]): void;

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
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

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
    gameEndReason: GameEndReasonMap[keyof GameEndReasonMap],
    scoresList: Array<number>,
    winner: number,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    gameId: string,
    lastUpdate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    gameRequest?: GameRequest.AsObject,
    tournamentDivision: string,
    tournamentRound: number,
    tournamentGameIndex: number,
  }
}

export class GameInfoResponses extends jspb.Message {
  clearGameInfoList(): void;
  getGameInfoList(): Array<GameInfoResponse>;
  setGameInfoList(value: Array<GameInfoResponse>): void;
  addGameInfo(value?: GameInfoResponse, index?: number): GameInfoResponse;

  getCount(): number;
  setCount(value: number): void;

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
    count: number,
  }
}

export class InstantiateGame extends jspb.Message {
  clearUserIdsList(): void;
  getUserIdsList(): Array<string>;
  setUserIdsList(value: Array<string>): void;
  addUserIds(value: string, index?: number): string;

  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  getAssignedFirst(): number;
  setAssignedFirst(value: number): void;

  hasTournamentData(): boolean;
  clearTournamentData(): void;
  getTournamentData(): TournamentDataForGame | undefined;
  setTournamentData(value?: TournamentDataForGame): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InstantiateGame.AsObject;
  static toObject(includeInstance: boolean, msg: InstantiateGame): InstantiateGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: InstantiateGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InstantiateGame;
  static deserializeBinaryFromReader(message: InstantiateGame, reader: jspb.BinaryReader): InstantiateGame;
}

export namespace InstantiateGame {
  export type AsObject = {
    userIdsList: Array<string>,
    gameRequest?: GameRequest.AsObject,
    assignedFirst: number,
    tournamentData?: TournamentDataForGame.AsObject,
  }
}

export class GameDeletion extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameDeletion.AsObject;
  static toObject(includeInstance: boolean, msg: GameDeletion): GameDeletion.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameDeletion, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameDeletion;
  static deserializeBinaryFromReader(message: GameDeletion, reader: jspb.BinaryReader): GameDeletion;
}

export namespace GameDeletion {
  export type AsObject = {
    id: string,
  }
}

export class ActiveGamePlayer extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ActiveGamePlayer.AsObject;
  static toObject(includeInstance: boolean, msg: ActiveGamePlayer): ActiveGamePlayer.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ActiveGamePlayer, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ActiveGamePlayer;
  static deserializeBinaryFromReader(message: ActiveGamePlayer, reader: jspb.BinaryReader): ActiveGamePlayer;
}

export namespace ActiveGamePlayer {
  export type AsObject = {
    username: string,
    userId: string,
  }
}

export class ActiveGameEntry extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  clearPlayerList(): void;
  getPlayerList(): Array<ActiveGamePlayer>;
  setPlayerList(value: Array<ActiveGamePlayer>): void;
  addPlayer(value?: ActiveGamePlayer, index?: number): ActiveGamePlayer;

  getTtl(): number;
  setTtl(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ActiveGameEntry.AsObject;
  static toObject(includeInstance: boolean, msg: ActiveGameEntry): ActiveGameEntry.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ActiveGameEntry, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ActiveGameEntry;
  static deserializeBinaryFromReader(message: ActiveGameEntry, reader: jspb.BinaryReader): ActiveGameEntry;
}

export namespace ActiveGameEntry {
  export type AsObject = {
    id: string,
    playerList: Array<ActiveGamePlayer.AsObject>,
    ttl: number,
  }
}

export class ReadyForGame extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReadyForGame.AsObject;
  static toObject(includeInstance: boolean, msg: ReadyForGame): ReadyForGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReadyForGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReadyForGame;
  static deserializeBinaryFromReader(message: ReadyForGame, reader: jspb.BinaryReader): ReadyForGame;
}

export namespace ReadyForGame {
  export type AsObject = {
    gameId: string,
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

  getReturnedTiles(): string;
  setReturnedTiles(value: string): void;

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
    returnedTiles: string,
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

  getTime(): number;
  setTime(value: number): void;

  getRatingDeltasMap(): jspb.Map<string, number>;
  clearRatingDeltasMap(): void;
  hasHistory(): boolean;
  clearHistory(): void;
  getHistory(): macondo_api_proto_macondo_macondo_pb.GameHistory | undefined;
  setHistory(value?: macondo_api_proto_macondo_macondo_pb.GameHistory): void;

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
    time: number,
    ratingDeltasMap: Array<[string, number]>,
    history?: macondo_api_proto_macondo_macondo_pb.GameHistory.AsObject,
  }
}

export class RematchStartedEvent extends jspb.Message {
  getRematchGameId(): string;
  setRematchGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RematchStartedEvent.AsObject;
  static toObject(includeInstance: boolean, msg: RematchStartedEvent): RematchStartedEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RematchStartedEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RematchStartedEvent;
  static deserializeBinaryFromReader(message: RematchStartedEvent, reader: jspb.BinaryReader): RematchStartedEvent;
}

export namespace RematchStartedEvent {
  export type AsObject = {
    rematchGameId: string,
  }
}

export class NewGameEvent extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  getRequesterCid(): string;
  setRequesterCid(value: string): void;

  getAccepterCid(): string;
  setAccepterCid(value: string): void;

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
    requesterCid: string,
    accepterCid: string,
  }
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

export interface GameEndReasonMap {
  NONE: 0;
  TIME: 1;
  STANDARD: 2;
  CONSECUTIVE_ZEROES: 3;
  RESIGNED: 4;
  ABORTED: 5;
  TRIPLE_CHALLENGE: 6;
  CANCELLED: 7;
  FORCE_FORFEIT: 8;
}

export const GameEndReason: GameEndReasonMap;

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

