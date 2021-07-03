// package: liwords
// file: api/proto/realtime/realtime.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

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
  }
}

export class MatchUser extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  getRelevantRating(): string;
  setRelevantRating(value: string): void;

  getIsAnonymous(): boolean;
  setIsAnonymous(value: boolean): void;

  getDisplayName(): string;
  setDisplayName(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MatchUser.AsObject;
  static toObject(includeInstance: boolean, msg: MatchUser): MatchUser.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MatchUser, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MatchUser;
  static deserializeBinaryFromReader(message: MatchUser, reader: jspb.BinaryReader): MatchUser;
}

export namespace MatchUser {
  export type AsObject = {
    userId: string,
    relevantRating: string,
    isAnonymous: boolean,
    displayName: string,
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

export class LagMeasurement extends jspb.Message {
  getLagMs(): number;
  setLagMs(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): LagMeasurement.AsObject;
  static toObject(includeInstance: boolean, msg: LagMeasurement): LagMeasurement.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: LagMeasurement, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): LagMeasurement;
  static deserializeBinaryFromReader(message: LagMeasurement, reader: jspb.BinaryReader): LagMeasurement;
}

export namespace LagMeasurement {
  export type AsObject = {
    lagMs: number,
  }
}

export class ChatMessage extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getChannel(): string;
  setChannel(value: string): void;

  getMessage(): string;
  setMessage(value: string): void;

  getTimestamp(): number;
  setTimestamp(value: number): void;

  getUserId(): string;
  setUserId(value: string): void;

  getId(): string;
  setId(value: string): void;

  getCountryCode(): string;
  setCountryCode(value: string): void;

  getAvatarUrl(): string;
  setAvatarUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChatMessage.AsObject;
  static toObject(includeInstance: boolean, msg: ChatMessage): ChatMessage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ChatMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChatMessage;
  static deserializeBinaryFromReader(message: ChatMessage, reader: jspb.BinaryReader): ChatMessage;
}

export namespace ChatMessage {
  export type AsObject = {
    username: string,
    channel: string,
    message: string,
    timestamp: number,
    userId: string,
    id: string,
    countryCode: string,
    avatarUrl: string,
  }
}

export class ChatMessages extends jspb.Message {
  clearMessagesList(): void;
  getMessagesList(): Array<ChatMessage>;
  setMessagesList(value: Array<ChatMessage>): void;
  addMessages(value?: ChatMessage, index?: number): ChatMessage;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChatMessages.AsObject;
  static toObject(includeInstance: boolean, msg: ChatMessages): ChatMessages.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ChatMessages, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChatMessages;
  static deserializeBinaryFromReader(message: ChatMessages, reader: jspb.BinaryReader): ChatMessages;
}

export namespace ChatMessages {
  export type AsObject = {
    messagesList: Array<ChatMessage.AsObject>,
  }
}

export class UserPresence extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  getChannel(): string;
  setChannel(value: string): void;

  getIsAnonymous(): boolean;
  setIsAnonymous(value: boolean): void;

  getDeleting(): boolean;
  setDeleting(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserPresence.AsObject;
  static toObject(includeInstance: boolean, msg: UserPresence): UserPresence.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UserPresence, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UserPresence;
  static deserializeBinaryFromReader(message: UserPresence, reader: jspb.BinaryReader): UserPresence;
}

export namespace UserPresence {
  export type AsObject = {
    username: string,
    userId: string,
    channel: string,
    isAnonymous: boolean,
    deleting: boolean,
  }
}

export class UserPresences extends jspb.Message {
  clearPresencesList(): void;
  getPresencesList(): Array<UserPresence>;
  setPresencesList(value: Array<UserPresence>): void;
  addPresences(value?: UserPresence, index?: number): UserPresence;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UserPresences.AsObject;
  static toObject(includeInstance: boolean, msg: UserPresences): UserPresences.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UserPresences, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UserPresences;
  static deserializeBinaryFromReader(message: UserPresences, reader: jspb.BinaryReader): UserPresences;
}

export namespace UserPresences {
  export type AsObject = {
    presencesList: Array<UserPresence.AsObject>,
  }
}

export class PresenceEntry extends jspb.Message {
  getUsername(): string;
  setUsername(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  clearChannelList(): void;
  getChannelList(): Array<string>;
  setChannelList(value: Array<string>): void;
  addChannel(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PresenceEntry.AsObject;
  static toObject(includeInstance: boolean, msg: PresenceEntry): PresenceEntry.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PresenceEntry, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PresenceEntry;
  static deserializeBinaryFromReader(message: PresenceEntry, reader: jspb.BinaryReader): PresenceEntry;
}

export namespace PresenceEntry {
  export type AsObject = {
    username: string,
    userId: string,
    channelList: Array<string>,
  }
}

export class SeekRequest extends jspb.Message {
  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): MatchUser | undefined;
  setUser(value?: MatchUser): void;

  getMinimumRating(): number;
  setMinimumRating(value: number): void;

  getMaximumRating(): number;
  setMaximumRating(value: number): void;

  getConnectionId(): string;
  setConnectionId(value: string): void;

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
    user?: MatchUser.AsObject,
    minimumRating: number,
    maximumRating: number,
    connectionId: string,
  }
}

export class MatchRequest extends jspb.Message {
  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): MatchUser | undefined;
  setUser(value?: MatchUser): void;

  hasReceivingUser(): boolean;
  clearReceivingUser(): void;
  getReceivingUser(): MatchUser | undefined;
  setReceivingUser(value?: MatchUser): void;

  getRematchFor(): string;
  setRematchFor(value: string): void;

  getConnectionId(): string;
  setConnectionId(value: string): void;

  getTournamentId(): string;
  setTournamentId(value: string): void;

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
    user?: MatchUser.AsObject,
    receivingUser?: MatchUser.AsObject,
    rematchFor: string,
    connectionId: string,
    tournamentId: string,
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

export class SoughtGameProcessEvent extends jspb.Message {
  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SoughtGameProcessEvent.AsObject;
  static toObject(includeInstance: boolean, msg: SoughtGameProcessEvent): SoughtGameProcessEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SoughtGameProcessEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SoughtGameProcessEvent;
  static deserializeBinaryFromReader(message: SoughtGameProcessEvent, reader: jspb.BinaryReader): SoughtGameProcessEvent;
}

export namespace SoughtGameProcessEvent {
  export type AsObject = {
    requestId: string,
  }
}

export class MatchRequestCancellation extends jspb.Message {
  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MatchRequestCancellation.AsObject;
  static toObject(includeInstance: boolean, msg: MatchRequestCancellation): MatchRequestCancellation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MatchRequestCancellation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MatchRequestCancellation;
  static deserializeBinaryFromReader(message: MatchRequestCancellation, reader: jspb.BinaryReader): MatchRequestCancellation;
}

export namespace MatchRequestCancellation {
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

export class MatchRequests extends jspb.Message {
  clearRequestsList(): void;
  getRequestsList(): Array<MatchRequest>;
  setRequestsList(value: Array<MatchRequest>): void;
  addRequests(value?: MatchRequest, index?: number): MatchRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): MatchRequests.AsObject;
  static toObject(includeInstance: boolean, msg: MatchRequests): MatchRequests.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: MatchRequests, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): MatchRequests;
  static deserializeBinaryFromReader(message: MatchRequests, reader: jspb.BinaryReader): MatchRequests;
}

export namespace MatchRequests {
  export type AsObject = {
    requestsList: Array<MatchRequest.AsObject>,
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

export class TournamentGameEndedEvent extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  clearPlayersList(): void;
  getPlayersList(): Array<TournamentGameEndedEvent.Player>;
  setPlayersList(value: Array<TournamentGameEndedEvent.Player>): void;
  addPlayers(value?: TournamentGameEndedEvent.Player, index?: number): TournamentGameEndedEvent.Player;

  getEndReason(): GameEndReasonMap[keyof GameEndReasonMap];
  setEndReason(value: GameEndReasonMap[keyof GameEndReasonMap]): void;

  getTime(): number;
  setTime(value: number): void;

  getRound(): number;
  setRound(value: number): void;

  getDivision(): string;
  setDivision(value: string): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentGameEndedEvent.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentGameEndedEvent): TournamentGameEndedEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentGameEndedEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentGameEndedEvent;
  static deserializeBinaryFromReader(message: TournamentGameEndedEvent, reader: jspb.BinaryReader): TournamentGameEndedEvent;
}

export namespace TournamentGameEndedEvent {
  export type AsObject = {
    gameId: string,
    playersList: Array<TournamentGameEndedEvent.Player.AsObject>,
    endReason: GameEndReasonMap[keyof GameEndReasonMap],
    time: number,
    round: number,
    division: string,
    gameIndex: number,
  }

  export class Player extends jspb.Message {
    getUsername(): string;
    setUsername(value: string): void;

    getScore(): number;
    setScore(value: number): void;

    getResult(): TournamentGameResultMap[keyof TournamentGameResultMap];
    setResult(value: TournamentGameResultMap[keyof TournamentGameResultMap]): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Player.AsObject;
    static toObject(includeInstance: boolean, msg: Player): Player.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Player, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Player;
    static deserializeBinaryFromReader(message: Player, reader: jspb.BinaryReader): Player;
  }

  export namespace Player {
    export type AsObject = {
      username: string,
      score: number,
      result: TournamentGameResultMap[keyof TournamentGameResultMap],
    }
  }
}

export class TournamentRoundStarted extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  hasDeadline(): boolean;
  clearDeadline(): void;
  getDeadline(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDeadline(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentRoundStarted.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentRoundStarted): TournamentRoundStarted.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentRoundStarted, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentRoundStarted;
  static deserializeBinaryFromReader(message: TournamentRoundStarted, reader: jspb.BinaryReader): TournamentRoundStarted;
}

export namespace TournamentRoundStarted {
  export type AsObject = {
    tournamentId: string,
    division: string,
    round: number,
    gameIndex: number,
    deadline?: google_protobuf_timestamp_pb.Timestamp.AsObject,
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

export class ServerMessage extends jspb.Message {
  getMessage(): string;
  setMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ServerMessage.AsObject;
  static toObject(includeInstance: boolean, msg: ServerMessage): ServerMessage.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ServerMessage, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ServerMessage;
  static deserializeBinaryFromReader(message: ServerMessage, reader: jspb.BinaryReader): ServerMessage;
}

export namespace ServerMessage {
  export type AsObject = {
    message: string,
  }
}

export class ChatMessageDeleted extends jspb.Message {
  getChannel(): string;
  setChannel(value: string): void;

  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ChatMessageDeleted.AsObject;
  static toObject(includeInstance: boolean, msg: ChatMessageDeleted): ChatMessageDeleted.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ChatMessageDeleted, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ChatMessageDeleted;
  static deserializeBinaryFromReader(message: ChatMessageDeleted, reader: jspb.BinaryReader): ChatMessageDeleted;
}

export namespace ChatMessageDeleted {
  export type AsObject = {
    channel: string,
    id: string,
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
    RESIGN: 4;
  }

  export const EventType: EventTypeMap;
}

export class ReadyForTournamentGame extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getPlayerId(): string;
  setPlayerId(value: string): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  getUnready(): boolean;
  setUnready(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReadyForTournamentGame.AsObject;
  static toObject(includeInstance: boolean, msg: ReadyForTournamentGame): ReadyForTournamentGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReadyForTournamentGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReadyForTournamentGame;
  static deserializeBinaryFromReader(message: ReadyForTournamentGame, reader: jspb.BinaryReader): ReadyForTournamentGame;
}

export namespace ReadyForTournamentGame {
  export type AsObject = {
    tournamentId: string,
    division: string,
    round: number,
    playerId: string,
    gameIndex: number,
    unready: boolean,
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

export class DeclineMatchRequest extends jspb.Message {
  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeclineMatchRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeclineMatchRequest): DeclineMatchRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeclineMatchRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeclineMatchRequest;
  static deserializeBinaryFromReader(message: DeclineMatchRequest, reader: jspb.BinaryReader): DeclineMatchRequest;
}

export namespace DeclineMatchRequest {
  export type AsObject = {
    requestId: string,
  }
}

export class TournamentPerson extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getRating(): number;
  setRating(value: number): void;

  getSuspended(): boolean;
  setSuspended(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPerson.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPerson): TournamentPerson.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPerson, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPerson;
  static deserializeBinaryFromReader(message: TournamentPerson, reader: jspb.BinaryReader): TournamentPerson;
}

export namespace TournamentPerson {
  export type AsObject = {
    id: string,
    rating: number,
    suspended: boolean,
  }
}

export class TournamentPersons extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearPersonsList(): void;
  getPersonsList(): Array<TournamentPerson>;
  setPersonsList(value: Array<TournamentPerson>): void;
  addPersons(value?: TournamentPerson, index?: number): TournamentPerson;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPersons.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPersons): TournamentPersons.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPersons, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPersons;
  static deserializeBinaryFromReader(message: TournamentPersons, reader: jspb.BinaryReader): TournamentPersons;
}

export namespace TournamentPersons {
  export type AsObject = {
    id: string,
    division: string,
    personsList: Array<TournamentPerson.AsObject>,
  }
}

export class RoundControl extends jspb.Message {
  getPairingMethod(): PairingMethodMap[keyof PairingMethodMap];
  setPairingMethod(value: PairingMethodMap[keyof PairingMethodMap]): void;

  getFirstMethod(): FirstMethodMap[keyof FirstMethodMap];
  setFirstMethod(value: FirstMethodMap[keyof FirstMethodMap]): void;

  getGamesPerRound(): number;
  setGamesPerRound(value: number): void;

  getRound(): number;
  setRound(value: number): void;

  getFactor(): number;
  setFactor(value: number): void;

  getInitialFontes(): number;
  setInitialFontes(value: number): void;

  getMaxRepeats(): number;
  setMaxRepeats(value: number): void;

  getAllowOverMaxRepeats(): boolean;
  setAllowOverMaxRepeats(value: boolean): void;

  getRepeatRelativeWeight(): number;
  setRepeatRelativeWeight(value: number): void;

  getWinDifferenceRelativeWeight(): number;
  setWinDifferenceRelativeWeight(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RoundControl.AsObject;
  static toObject(includeInstance: boolean, msg: RoundControl): RoundControl.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RoundControl, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RoundControl;
  static deserializeBinaryFromReader(message: RoundControl, reader: jspb.BinaryReader): RoundControl;
}

export namespace RoundControl {
  export type AsObject = {
    pairingMethod: PairingMethodMap[keyof PairingMethodMap],
    firstMethod: FirstMethodMap[keyof FirstMethodMap],
    gamesPerRound: number,
    round: number,
    factor: number,
    initialFontes: number,
    maxRepeats: number,
    allowOverMaxRepeats: boolean,
    repeatRelativeWeight: number,
    winDifferenceRelativeWeight: number,
  }
}

export class DivisionControls extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): GameRequest | undefined;
  setGameRequest(value?: GameRequest): void;

  getSuspendedResult(): TournamentGameResultMap[keyof TournamentGameResultMap];
  setSuspendedResult(value: TournamentGameResultMap[keyof TournamentGameResultMap]): void;

  getSuspendedSpread(): number;
  setSuspendedSpread(value: number): void;

  getAutoStart(): boolean;
  setAutoStart(value: boolean): void;

  getSpreadCap(): number;
  setSpreadCap(value: number): void;

  getGibsonize(): boolean;
  setGibsonize(value: boolean): void;

  getGibsonSpread(): number;
  setGibsonSpread(value: number): void;

  getMinimumPlacement(): number;
  setMinimumPlacement(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionControls.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionControls): DivisionControls.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionControls, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionControls;
  static deserializeBinaryFromReader(message: DivisionControls, reader: jspb.BinaryReader): DivisionControls;
}

export namespace DivisionControls {
  export type AsObject = {
    id: string,
    division: string,
    gameRequest?: GameRequest.AsObject,
    suspendedResult: TournamentGameResultMap[keyof TournamentGameResultMap],
    suspendedSpread: number,
    autoStart: boolean,
    spreadCap: number,
    gibsonize: boolean,
    gibsonSpread: number,
    minimumPlacement: number,
  }
}

export class TournamentGame extends jspb.Message {
  clearScoresList(): void;
  getScoresList(): Array<number>;
  setScoresList(value: Array<number>): void;
  addScores(value: number, index?: number): number;

  clearResultsList(): void;
  getResultsList(): Array<TournamentGameResultMap[keyof TournamentGameResultMap]>;
  setResultsList(value: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>): void;
  addResults(value: TournamentGameResultMap[keyof TournamentGameResultMap], index?: number): TournamentGameResultMap[keyof TournamentGameResultMap];

  getGameEndReason(): GameEndReasonMap[keyof GameEndReasonMap];
  setGameEndReason(value: GameEndReasonMap[keyof GameEndReasonMap]): void;

  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentGame.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentGame): TournamentGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentGame;
  static deserializeBinaryFromReader(message: TournamentGame, reader: jspb.BinaryReader): TournamentGame;
}

export namespace TournamentGame {
  export type AsObject = {
    scoresList: Array<number>,
    resultsList: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>,
    gameEndReason: GameEndReasonMap[keyof GameEndReasonMap],
    id: string,
  }
}

export class Pairing extends jspb.Message {
  clearPlayersList(): void;
  getPlayersList(): Array<number>;
  setPlayersList(value: Array<number>): void;
  addPlayers(value: number, index?: number): number;

  getRound(): number;
  setRound(value: number): void;

  clearGamesList(): void;
  getGamesList(): Array<TournamentGame>;
  setGamesList(value: Array<TournamentGame>): void;
  addGames(value?: TournamentGame, index?: number): TournamentGame;

  clearOutcomesList(): void;
  getOutcomesList(): Array<TournamentGameResultMap[keyof TournamentGameResultMap]>;
  setOutcomesList(value: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>): void;
  addOutcomes(value: TournamentGameResultMap[keyof TournamentGameResultMap], index?: number): TournamentGameResultMap[keyof TournamentGameResultMap];

  clearReadyStatesList(): void;
  getReadyStatesList(): Array<string>;
  setReadyStatesList(value: Array<string>): void;
  addReadyStates(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Pairing.AsObject;
  static toObject(includeInstance: boolean, msg: Pairing): Pairing.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Pairing, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Pairing;
  static deserializeBinaryFromReader(message: Pairing, reader: jspb.BinaryReader): Pairing;
}

export namespace Pairing {
  export type AsObject = {
    playersList: Array<number>,
    round: number,
    gamesList: Array<TournamentGame.AsObject>,
    outcomesList: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>,
    readyStatesList: Array<string>,
  }
}

export class PlayerStanding extends jspb.Message {
  getPlayerId(): string;
  setPlayerId(value: string): void;

  getWins(): number;
  setWins(value: number): void;

  getLosses(): number;
  setLosses(value: number): void;

  getDraws(): number;
  setDraws(value: number): void;

  getSpread(): number;
  setSpread(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PlayerStanding.AsObject;
  static toObject(includeInstance: boolean, msg: PlayerStanding): PlayerStanding.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PlayerStanding, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PlayerStanding;
  static deserializeBinaryFromReader(message: PlayerStanding, reader: jspb.BinaryReader): PlayerStanding;
}

export namespace PlayerStanding {
  export type AsObject = {
    playerId: string,
    wins: number,
    losses: number,
    draws: number,
    spread: number,
  }
}

export class RoundStandings extends jspb.Message {
  clearStandingsList(): void;
  getStandingsList(): Array<PlayerStanding>;
  setStandingsList(value: Array<PlayerStanding>): void;
  addStandings(value?: PlayerStanding, index?: number): PlayerStanding;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RoundStandings.AsObject;
  static toObject(includeInstance: boolean, msg: RoundStandings): RoundStandings.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RoundStandings, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RoundStandings;
  static deserializeBinaryFromReader(message: RoundStandings, reader: jspb.BinaryReader): RoundStandings;
}

export namespace RoundStandings {
  export type AsObject = {
    standingsList: Array<PlayerStanding.AsObject>,
  }
}

export class DivisionPairingsResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  getGibsonizedPlayersMap(): jspb.Map<string, number>;
  clearGibsonizedPlayersMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionPairingsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionPairingsResponse): DivisionPairingsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionPairingsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionPairingsResponse;
  static deserializeBinaryFromReader(message: DivisionPairingsResponse, reader: jspb.BinaryReader): DivisionPairingsResponse;
}

export namespace DivisionPairingsResponse {
  export type AsObject = {
    id: string,
    division: string,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
    gibsonizedPlayersMap: Array<[string, number]>,
  }
}

export class DivisionPairingsDeletedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionPairingsDeletedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionPairingsDeletedResponse): DivisionPairingsDeletedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionPairingsDeletedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionPairingsDeletedResponse;
  static deserializeBinaryFromReader(message: DivisionPairingsDeletedResponse, reader: jspb.BinaryReader): DivisionPairingsDeletedResponse;
}

export namespace DivisionPairingsDeletedResponse {
  export type AsObject = {
    id: string,
    division: string,
    round: number,
  }
}

export class PlayersAddedOrRemovedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasPlayers(): boolean;
  clearPlayers(): void;
  getPlayers(): TournamentPersons | undefined;
  setPlayers(value?: TournamentPersons): void;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PlayersAddedOrRemovedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PlayersAddedOrRemovedResponse): PlayersAddedOrRemovedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PlayersAddedOrRemovedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PlayersAddedOrRemovedResponse;
  static deserializeBinaryFromReader(message: PlayersAddedOrRemovedResponse, reader: jspb.BinaryReader): PlayersAddedOrRemovedResponse;
}

export namespace PlayersAddedOrRemovedResponse {
  export type AsObject = {
    id: string,
    division: string,
    players?: TournamentPersons.AsObject,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class DivisionRoundControls extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearRoundControlsList(): void;
  getRoundControlsList(): Array<RoundControl>;
  setRoundControlsList(value: Array<RoundControl>): void;
  addRoundControls(value?: RoundControl, index?: number): RoundControl;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionRoundControls.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionRoundControls): DivisionRoundControls.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionRoundControls, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionRoundControls;
  static deserializeBinaryFromReader(message: DivisionRoundControls, reader: jspb.BinaryReader): DivisionRoundControls;
}

export namespace DivisionRoundControls {
  export type AsObject = {
    id: string,
    division: string,
    roundControlsList: Array<RoundControl.AsObject>,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class DivisionControlsResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasDivisionControls(): boolean;
  clearDivisionControls(): void;
  getDivisionControls(): DivisionControls | undefined;
  setDivisionControls(value?: DivisionControls): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionControlsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionControlsResponse): DivisionControlsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionControlsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionControlsResponse;
  static deserializeBinaryFromReader(message: DivisionControlsResponse, reader: jspb.BinaryReader): DivisionControlsResponse;
}

export namespace DivisionControlsResponse {
  export type AsObject = {
    id: string,
    division: string,
    divisionControls?: DivisionControls.AsObject,
  }
}

export class TournamentDivisionDataResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasPlayers(): boolean;
  clearPlayers(): void;
  getPlayers(): TournamentPersons | undefined;
  setPlayers(value?: TournamentPersons): void;

  getStandingsMap(): jspb.Map<number, RoundStandings>;
  clearStandingsMap(): void;
  getPairingMapMap(): jspb.Map<string, Pairing>;
  clearPairingMapMap(): void;
  hasControls(): boolean;
  clearControls(): void;
  getControls(): DivisionControls | undefined;
  setControls(value?: DivisionControls): void;

  clearRoundControlsList(): void;
  getRoundControlsList(): Array<RoundControl>;
  setRoundControlsList(value: Array<RoundControl>): void;
  addRoundControls(value?: RoundControl, index?: number): RoundControl;

  getCurrentRound(): number;
  setCurrentRound(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDivisionDataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDivisionDataResponse): TournamentDivisionDataResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDivisionDataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDivisionDataResponse;
  static deserializeBinaryFromReader(message: TournamentDivisionDataResponse, reader: jspb.BinaryReader): TournamentDivisionDataResponse;
}

export namespace TournamentDivisionDataResponse {
  export type AsObject = {
    id: string,
    division: string,
    players?: TournamentPersons.AsObject,
    standingsMap: Array<[number, RoundStandings.AsObject]>,
    pairingMapMap: Array<[string, Pairing.AsObject]>,
    controls?: DivisionControls.AsObject,
    roundControlsList: Array<RoundControl.AsObject>,
    currentRound: number,
  }
}

export class FullTournamentDivisions extends jspb.Message {
  getDivisionsMap(): jspb.Map<string, TournamentDivisionDataResponse>;
  clearDivisionsMap(): void;
  getStarted(): boolean;
  setStarted(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FullTournamentDivisions.AsObject;
  static toObject(includeInstance: boolean, msg: FullTournamentDivisions): FullTournamentDivisions.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FullTournamentDivisions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FullTournamentDivisions;
  static deserializeBinaryFromReader(message: FullTournamentDivisions, reader: jspb.BinaryReader): FullTournamentDivisions;
}

export namespace FullTournamentDivisions {
  export type AsObject = {
    divisionsMap: Array<[string, TournamentDivisionDataResponse.AsObject]>,
    started: boolean,
  }
}

export class TournamentFinishedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentFinishedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentFinishedResponse): TournamentFinishedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentFinishedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentFinishedResponse;
  static deserializeBinaryFromReader(message: TournamentFinishedResponse, reader: jspb.BinaryReader): TournamentFinishedResponse;
}

export namespace TournamentFinishedResponse {
  export type AsObject = {
    id: string,
  }
}

export class TournamentDataResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getExecutiveDirector(): string;
  setExecutiveDirector(value: string): void;

  hasDirectors(): boolean;
  clearDirectors(): void;
  getDirectors(): TournamentPersons | undefined;
  setDirectors(value?: TournamentPersons): void;

  getIsStarted(): boolean;
  setIsStarted(value: boolean): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDataResponse): TournamentDataResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDataResponse;
  static deserializeBinaryFromReader(message: TournamentDataResponse, reader: jspb.BinaryReader): TournamentDataResponse;
}

export namespace TournamentDataResponse {
  export type AsObject = {
    id: string,
    name: string,
    description: string,
    executiveDirector: string,
    directors?: TournamentPersons.AsObject,
    isStarted: boolean,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class TournamentDivisionDeletedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDivisionDeletedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDivisionDeletedResponse): TournamentDivisionDeletedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDivisionDeletedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDivisionDeletedResponse;
  static deserializeBinaryFromReader(message: TournamentDivisionDeletedResponse, reader: jspb.BinaryReader): TournamentDivisionDeletedResponse;
}

export namespace TournamentDivisionDeletedResponse {
  export type AsObject = {
    id: string,
    division: string,
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

export interface ChildStatusMap {
  CHILD: 0;
  NOT_CHILD: 1;
  UNKNOWN: 2;
}

export const ChildStatus: ChildStatusMap;

export interface MessageTypeMap {
  SEEK_REQUEST: 0;
  MATCH_REQUEST: 1;
  SOUGHT_GAME_PROCESS_EVENT: 2;
  CLIENT_GAMEPLAY_EVENT: 3;
  SERVER_GAMEPLAY_EVENT: 4;
  GAME_ENDED_EVENT: 5;
  GAME_HISTORY_REFRESHER: 6;
  ERROR_MESSAGE: 7;
  NEW_GAME_EVENT: 8;
  SERVER_CHALLENGE_RESULT_EVENT: 9;
  SEEK_REQUESTS: 10;
  MATCH_REQUEST_CANCELLATION: 11;
  ONGOING_GAME_EVENT: 12;
  TIMED_OUT: 13;
  ONGOING_GAMES: 14;
  READY_FOR_TOURNAMENT_GAME: 15;
  TOURNAMENT_ROUND_STARTED: 16;
  GAME_DELETION: 17;
  MATCH_REQUESTS: 18;
  DECLINE_MATCH_REQUEST: 19;
  CHAT_MESSAGE: 20;
  CHAT_MESSAGE_DELETED: 21;
  USER_PRESENCE: 22;
  USER_PRESENCES: 23;
  SERVER_MESSAGE: 24;
  READY_FOR_GAME: 25;
  LAG_MEASUREMENT: 26;
  TOURNAMENT_GAME_ENDED_EVENT: 27;
  TOURNAMENT_MESSAGE: 28;
  REMATCH_STARTED: 29;
  TOURNAMENT_DIVISION_MESSAGE: 30;
  TOURNAMENT_DIVISION_DELETED_MESSAGE: 31;
  TOURNAMENT_FULL_DIVISIONS_MESSAGE: 32;
  TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE: 34;
  TOURNAMENT_DIVISION_PAIRINGS_MESSAGE: 35;
  TOURNAMENT_DIVISION_CONTROLS_MESSAGE: 36;
  TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE: 37;
  TOURNAMENT_FINISHED_MESSAGE: 38;
  TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE: 39;
  PRESENCE_ENTRY: 40;
  ACTIVE_GAME_ENTRY: 41;
  GAME_META_EVENT: 42;
}

export const MessageType: MessageTypeMap;

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

export interface TournamentGameResultMap {
  NO_RESULT: 0;
  WIN: 1;
  LOSS: 2;
  DRAW: 3;
  BYE: 4;
  FORFEIT_WIN: 5;
  FORFEIT_LOSS: 6;
  ELIMINATED: 7;
}

export const TournamentGameResult: TournamentGameResultMap;

export interface PairingMethodMap {
  RANDOM: 0;
  ROUND_ROBIN: 1;
  KING_OF_THE_HILL: 2;
  ELIMINATION: 3;
  FACTOR: 4;
  INITIAL_FONTES: 5;
  SWISS: 6;
  QUICKPAIR: 7;
  MANUAL: 8;
  TEAM_ROUND_ROBIN: 9;
}

export const PairingMethod: PairingMethodMap;

export interface FirstMethodMap {
  MANUAL_FIRST: 0;
  RANDOM_FIRST: 1;
  AUTOMATIC_FIRST: 2;
}

export const FirstMethod: FirstMethodMap;

