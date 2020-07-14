// package: macondo
// file: macondo/api/proto/macondo/macondo.proto

import * as jspb from "google-protobuf";

export class GameHistory extends jspb.Message {
  clearEventsList(): void;
  getEventsList(): Array<GameEvent>;
  setEventsList(value: Array<GameEvent>): void;
  addEvents(value?: GameEvent, index?: number): GameEvent;

  clearPlayersList(): void;
  getPlayersList(): Array<PlayerInfo>;
  setPlayersList(value: Array<PlayerInfo>): void;
  addPlayers(value?: PlayerInfo, index?: number): PlayerInfo;

  getVersion(): number;
  setVersion(value: number): void;

  getOriginalGcg(): string;
  setOriginalGcg(value: string): void;

  getLexicon(): string;
  setLexicon(value: string): void;

  getIdAuth(): string;
  setIdAuth(value: string): void;

  getUid(): string;
  setUid(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  clearLastKnownRacksList(): void;
  getLastKnownRacksList(): Array<string>;
  setLastKnownRacksList(value: Array<string>): void;
  addLastKnownRacks(value: string, index?: number): string;

  getSecondWentFirst(): boolean;
  setSecondWentFirst(value: boolean): void;

  getChallengeRule(): ChallengeRuleMap[keyof ChallengeRuleMap];
  setChallengeRule(value: ChallengeRuleMap[keyof ChallengeRuleMap]): void;

  getPlayState(): PlayStateMap[keyof PlayStateMap];
  setPlayState(value: PlayStateMap[keyof PlayStateMap]): void;

  clearFinalScoresList(): void;
  getFinalScoresList(): Array<number>;
  setFinalScoresList(value: Array<number>): void;
  addFinalScores(value: number, index?: number): number;

  getVariant(): string;
  setVariant(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameHistory.AsObject;
  static toObject(includeInstance: boolean, msg: GameHistory): GameHistory.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameHistory, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameHistory;
  static deserializeBinaryFromReader(message: GameHistory, reader: jspb.BinaryReader): GameHistory;
}

export namespace GameHistory {
  export type AsObject = {
    eventsList: Array<GameEvent.AsObject>,
    playersList: Array<PlayerInfo.AsObject>,
    version: number,
    originalGcg: string,
    lexicon: string,
    idAuth: string,
    uid: string,
    title: string,
    description: string,
    lastKnownRacksList: Array<string>,
    secondWentFirst: boolean,
    challengeRule: ChallengeRuleMap[keyof ChallengeRuleMap],
    playState: PlayStateMap[keyof PlayStateMap],
    finalScoresList: Array<number>,
    variant: string,
  }
}

export class GameEvent extends jspb.Message {
  getNickname(): string;
  setNickname(value: string): void;

  getNote(): string;
  setNote(value: string): void;

  getRack(): string;
  setRack(value: string): void;

  getType(): GameEvent.TypeMap[keyof GameEvent.TypeMap];
  setType(value: GameEvent.TypeMap[keyof GameEvent.TypeMap]): void;

  getCumulative(): number;
  setCumulative(value: number): void;

  getRow(): number;
  setRow(value: number): void;

  getColumn(): number;
  setColumn(value: number): void;

  getDirection(): GameEvent.DirectionMap[keyof GameEvent.DirectionMap];
  setDirection(value: GameEvent.DirectionMap[keyof GameEvent.DirectionMap]): void;

  getPosition(): string;
  setPosition(value: string): void;

  getPlayedTiles(): string;
  setPlayedTiles(value: string): void;

  getExchanged(): string;
  setExchanged(value: string): void;

  getScore(): number;
  setScore(value: number): void;

  getBonus(): number;
  setBonus(value: number): void;

  getEndRackPoints(): number;
  setEndRackPoints(value: number): void;

  getLostScore(): number;
  setLostScore(value: number): void;

  getIsBingo(): boolean;
  setIsBingo(value: boolean): void;

  clearWordsFormedList(): void;
  getWordsFormedList(): Array<string>;
  setWordsFormedList(value: Array<string>): void;
  addWordsFormed(value: string, index?: number): string;

  getMillisRemaining(): number;
  setMillisRemaining(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameEvent.AsObject;
  static toObject(includeInstance: boolean, msg: GameEvent): GameEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameEvent;
  static deserializeBinaryFromReader(message: GameEvent, reader: jspb.BinaryReader): GameEvent;
}

export namespace GameEvent {
  export type AsObject = {
    nickname: string,
    note: string,
    rack: string,
    type: GameEvent.TypeMap[keyof GameEvent.TypeMap],
    cumulative: number,
    row: number,
    column: number,
    direction: GameEvent.DirectionMap[keyof GameEvent.DirectionMap],
    position: string,
    playedTiles: string,
    exchanged: string,
    score: number,
    bonus: number,
    endRackPoints: number,
    lostScore: number,
    isBingo: boolean,
    wordsFormedList: Array<string>,
    millisRemaining: number,
  }

  export interface TypeMap {
    TILE_PLACEMENT_MOVE: 0;
    PHONY_TILES_RETURNED: 1;
    PASS: 2;
    CHALLENGE_BONUS: 3;
    EXCHANGE: 4;
    END_RACK_PTS: 5;
    TIME_PENALTY: 6;
    END_RACK_PENALTY: 7;
    UNSUCCESSFUL_CHALLENGE_TURN_LOSS: 8;
  }

  export const Type: TypeMap;

  export interface DirectionMap {
    HORIZONTAL: 0;
    VERTICAL: 1;
  }

  export const Direction: DirectionMap;
}

export class PlayerInfo extends jspb.Message {
  getNickname(): string;
  setNickname(value: string): void;

  getRealName(): string;
  setRealName(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

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
    nickname: string,
    realName: string,
    userId: string,
  }
}

export interface PlayStateMap {
  PLAYING: 0;
  WAITING_FOR_FINAL_PASS: 1;
  GAME_OVER: 2;
}

export const PlayState: PlayStateMap;

export interface ChallengeRuleMap {
  VOID: 0;
  SINGLE: 1;
  DOUBLE: 2;
  FIVE_POINT: 3;
  TEN_POINT: 4;
}

export const ChallengeRule: ChallengeRuleMap;

