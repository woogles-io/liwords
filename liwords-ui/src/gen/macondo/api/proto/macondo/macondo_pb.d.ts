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

  getWinner(): number;
  setWinner(value: number): void;

  getBoardLayout(): string;
  setBoardLayout(value: string): void;

  getLetterDistribution(): string;
  setLetterDistribution(value: string): void;

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
    winner: number,
    boardLayout: string,
    letterDistribution: string,
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
    CHALLENGE: 9;
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

export class BotRequest extends jspb.Message {
  hasGameHistory(): boolean;
  clearGameHistory(): void;
  getGameHistory(): GameHistory | undefined;
  setGameHistory(value?: GameHistory): void;

  hasEvaluationRequest(): boolean;
  clearEvaluationRequest(): void;
  getEvaluationRequest(): EvaluationRequest | undefined;
  setEvaluationRequest(value?: EvaluationRequest): void;

  getBotType(): BotRequest.BotCodeMap[keyof BotRequest.BotCodeMap];
  setBotType(value: BotRequest.BotCodeMap[keyof BotRequest.BotCodeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BotRequest.AsObject;
  static toObject(includeInstance: boolean, msg: BotRequest): BotRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BotRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BotRequest;
  static deserializeBinaryFromReader(message: BotRequest, reader: jspb.BinaryReader): BotRequest;
}

export namespace BotRequest {
  export type AsObject = {
    gameHistory?: GameHistory.AsObject,
    evaluationRequest?: EvaluationRequest.AsObject,
    botType: BotRequest.BotCodeMap[keyof BotRequest.BotCodeMap],
  }

  export interface BotCodeMap {
    HASTY_BOT: 0;
    LEVEL1_CEL_BOT: 1;
    LEVEL2_CEL_BOT: 2;
    LEVEL3_CEL_BOT: 3;
    LEVEL4_CEL_BOT: 4;
    LEVEL1_PROBABILISTIC: 5;
    LEVEL2_PROBABILISTIC: 6;
    LEVEL3_PROBABILISTIC: 7;
    LEVEL4_PROBABILISTIC: 8;
    LEVEL5_PROBABILISTIC: 9;
    SIMMING_BOT: 10;
  }

  export const BotCode: BotCodeMap;
}

export class EvaluationRequest extends jspb.Message {
  getUser(): string;
  setUser(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): EvaluationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: EvaluationRequest): EvaluationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: EvaluationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): EvaluationRequest;
  static deserializeBinaryFromReader(message: EvaluationRequest, reader: jspb.BinaryReader): EvaluationRequest;
}

export namespace EvaluationRequest {
  export type AsObject = {
    user: string,
  }
}

export class Evaluation extends jspb.Message {
  clearPlayEvalList(): void;
  getPlayEvalList(): Array<SingleEvaluation>;
  setPlayEvalList(value: Array<SingleEvaluation>): void;
  addPlayEval(value?: SingleEvaluation, index?: number): SingleEvaluation;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Evaluation.AsObject;
  static toObject(includeInstance: boolean, msg: Evaluation): Evaluation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Evaluation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Evaluation;
  static deserializeBinaryFromReader(message: Evaluation, reader: jspb.BinaryReader): Evaluation;
}

export namespace Evaluation {
  export type AsObject = {
    playEvalList: Array<SingleEvaluation.AsObject>,
  }
}

export class SingleEvaluation extends jspb.Message {
  getEquityLoss(): number;
  setEquityLoss(value: number): void;

  getWinPctLoss(): number;
  setWinPctLoss(value: number): void;

  getMissedBingo(): boolean;
  setMissedBingo(value: boolean): void;

  getPossibleStarPlay(): boolean;
  setPossibleStarPlay(value: boolean): void;

  getMissedStarPlay(): boolean;
  setMissedStarPlay(value: boolean): void;

  getTopIsBingo(): boolean;
  setTopIsBingo(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SingleEvaluation.AsObject;
  static toObject(includeInstance: boolean, msg: SingleEvaluation): SingleEvaluation.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SingleEvaluation, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SingleEvaluation;
  static deserializeBinaryFromReader(message: SingleEvaluation, reader: jspb.BinaryReader): SingleEvaluation;
}

export namespace SingleEvaluation {
  export type AsObject = {
    equityLoss: number,
    winPctLoss: number,
    missedBingo: boolean,
    possibleStarPlay: boolean,
    missedStarPlay: boolean,
    topIsBingo: boolean,
  }
}

export class BotResponse extends jspb.Message {
  hasMove(): boolean;
  clearMove(): void;
  getMove(): GameEvent | undefined;
  setMove(value?: GameEvent): void;

  hasError(): boolean;
  clearError(): void;
  getError(): string;
  setError(value: string): void;

  hasEval(): boolean;
  clearEval(): void;
  getEval(): Evaluation | undefined;
  setEval(value?: Evaluation): void;

  getResponseCase(): BotResponse.ResponseCase;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BotResponse.AsObject;
  static toObject(includeInstance: boolean, msg: BotResponse): BotResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BotResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BotResponse;
  static deserializeBinaryFromReader(message: BotResponse, reader: jspb.BinaryReader): BotResponse;
}

export namespace BotResponse {
  export type AsObject = {
    move?: GameEvent.AsObject,
    error: string,
    eval?: Evaluation.AsObject,
  }

  export enum ResponseCase {
    RESPONSE_NOT_SET = 0,
    MOVE = 1,
    ERROR = 2,
  }
}

export class PuzzleCreationResponse extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  getTurnNumber(): number;
  setTurnNumber(value: number): void;

  hasAnswer(): boolean;
  clearAnswer(): void;
  getAnswer(): GameEvent | undefined;
  setAnswer(value?: GameEvent): void;

  clearTagsList(): void;
  getTagsList(): Array<PuzzleTagMap[keyof PuzzleTagMap]>;
  setTagsList(value: Array<PuzzleTagMap[keyof PuzzleTagMap]>): void;
  addTags(value: PuzzleTagMap[keyof PuzzleTagMap], index?: number): PuzzleTagMap[keyof PuzzleTagMap];

  getBucketIndex(): number;
  setBucketIndex(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleCreationResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleCreationResponse): PuzzleCreationResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleCreationResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleCreationResponse;
  static deserializeBinaryFromReader(message: PuzzleCreationResponse, reader: jspb.BinaryReader): PuzzleCreationResponse;
}

export namespace PuzzleCreationResponse {
  export type AsObject = {
    gameId: string,
    turnNumber: number,
    answer?: GameEvent.AsObject,
    tagsList: Array<PuzzleTagMap[keyof PuzzleTagMap]>,
    bucketIndex: number,
  }
}

export class PuzzleBucket extends jspb.Message {
  getIndex(): number;
  setIndex(value: number): void;

  getSize(): number;
  setSize(value: number): void;

  clearIncludesList(): void;
  getIncludesList(): Array<PuzzleTagMap[keyof PuzzleTagMap]>;
  setIncludesList(value: Array<PuzzleTagMap[keyof PuzzleTagMap]>): void;
  addIncludes(value: PuzzleTagMap[keyof PuzzleTagMap], index?: number): PuzzleTagMap[keyof PuzzleTagMap];

  clearExcludesList(): void;
  getExcludesList(): Array<PuzzleTagMap[keyof PuzzleTagMap]>;
  setExcludesList(value: Array<PuzzleTagMap[keyof PuzzleTagMap]>): void;
  addExcludes(value: PuzzleTagMap[keyof PuzzleTagMap], index?: number): PuzzleTagMap[keyof PuzzleTagMap];

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleBucket.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleBucket): PuzzleBucket.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleBucket, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleBucket;
  static deserializeBinaryFromReader(message: PuzzleBucket, reader: jspb.BinaryReader): PuzzleBucket;
}

export namespace PuzzleBucket {
  export type AsObject = {
    index: number,
    size: number,
    includesList: Array<PuzzleTagMap[keyof PuzzleTagMap]>,
    excludesList: Array<PuzzleTagMap[keyof PuzzleTagMap]>,
  }
}

export class PuzzleGenerationRequest extends jspb.Message {
  clearBucketsList(): void;
  getBucketsList(): Array<PuzzleBucket>;
  setBucketsList(value: Array<PuzzleBucket>): void;
  addBuckets(value?: PuzzleBucket, index?: number): PuzzleBucket;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleGenerationRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleGenerationRequest): PuzzleGenerationRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleGenerationRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleGenerationRequest;
  static deserializeBinaryFromReader(message: PuzzleGenerationRequest, reader: jspb.BinaryReader): PuzzleGenerationRequest;
}

export namespace PuzzleGenerationRequest {
  export type AsObject = {
    bucketsList: Array<PuzzleBucket.AsObject>,
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
  TRIPLE: 5;
}

export const ChallengeRule: ChallengeRuleMap;

export interface PuzzleTagMap {
  EQUITY: 0;
  BINGO: 1;
  ONLY_BINGO: 2;
  BLANK_BINGO: 3;
  NON_BINGO: 4;
  POWER_TILE: 5;
  BINGO_NINE_OR_ABOVE: 6;
  CEL_ONLY: 7;
}

export const PuzzleTag: PuzzleTagMap;

