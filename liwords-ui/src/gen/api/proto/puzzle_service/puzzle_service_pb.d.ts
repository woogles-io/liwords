// package: puzzle_service
// file: api/proto/puzzle_service/puzzle_service.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as api_proto_ipc_omgwords_pb from "../../../api/proto/ipc/omgwords_pb";

export class StartPuzzleIdRequest extends jspb.Message {
  getLexicon(): string;
  setLexicon(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartPuzzleIdRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartPuzzleIdRequest): StartPuzzleIdRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartPuzzleIdRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartPuzzleIdRequest;
  static deserializeBinaryFromReader(message: StartPuzzleIdRequest, reader: jspb.BinaryReader): StartPuzzleIdRequest;
}

export namespace StartPuzzleIdRequest {
  export type AsObject = {
    lexicon: string,
  }
}

export class StartPuzzleIdResponse extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  getQueryResult(): PuzzleQueryResultMap[keyof PuzzleQueryResultMap];
  setQueryResult(value: PuzzleQueryResultMap[keyof PuzzleQueryResultMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartPuzzleIdResponse.AsObject;
  static toObject(includeInstance: boolean, msg: StartPuzzleIdResponse): StartPuzzleIdResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartPuzzleIdResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartPuzzleIdResponse;
  static deserializeBinaryFromReader(message: StartPuzzleIdResponse, reader: jspb.BinaryReader): StartPuzzleIdResponse;
}

export namespace StartPuzzleIdResponse {
  export type AsObject = {
    puzzleId: string,
    queryResult: PuzzleQueryResultMap[keyof PuzzleQueryResultMap],
  }
}

export class NextPuzzleIdRequest extends jspb.Message {
  getLexicon(): string;
  setLexicon(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NextPuzzleIdRequest.AsObject;
  static toObject(includeInstance: boolean, msg: NextPuzzleIdRequest): NextPuzzleIdRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NextPuzzleIdRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NextPuzzleIdRequest;
  static deserializeBinaryFromReader(message: NextPuzzleIdRequest, reader: jspb.BinaryReader): NextPuzzleIdRequest;
}

export namespace NextPuzzleIdRequest {
  export type AsObject = {
    lexicon: string,
  }
}

export class NextPuzzleIdResponse extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  getQueryResult(): PuzzleQueryResultMap[keyof PuzzleQueryResultMap];
  setQueryResult(value: PuzzleQueryResultMap[keyof PuzzleQueryResultMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NextPuzzleIdResponse.AsObject;
  static toObject(includeInstance: boolean, msg: NextPuzzleIdResponse): NextPuzzleIdResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NextPuzzleIdResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NextPuzzleIdResponse;
  static deserializeBinaryFromReader(message: NextPuzzleIdResponse, reader: jspb.BinaryReader): NextPuzzleIdResponse;
}

export namespace NextPuzzleIdResponse {
  export type AsObject = {
    puzzleId: string,
    queryResult: PuzzleQueryResultMap[keyof PuzzleQueryResultMap],
  }
}

export class NextClosestRatingPuzzleIdRequest extends jspb.Message {
  getLexicon(): string;
  setLexicon(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NextClosestRatingPuzzleIdRequest.AsObject;
  static toObject(includeInstance: boolean, msg: NextClosestRatingPuzzleIdRequest): NextClosestRatingPuzzleIdRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NextClosestRatingPuzzleIdRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NextClosestRatingPuzzleIdRequest;
  static deserializeBinaryFromReader(message: NextClosestRatingPuzzleIdRequest, reader: jspb.BinaryReader): NextClosestRatingPuzzleIdRequest;
}

export namespace NextClosestRatingPuzzleIdRequest {
  export type AsObject = {
    lexicon: string,
  }
}

export class NextClosestRatingPuzzleIdResponse extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  getQueryResult(): PuzzleQueryResultMap[keyof PuzzleQueryResultMap];
  setQueryResult(value: PuzzleQueryResultMap[keyof PuzzleQueryResultMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NextClosestRatingPuzzleIdResponse.AsObject;
  static toObject(includeInstance: boolean, msg: NextClosestRatingPuzzleIdResponse): NextClosestRatingPuzzleIdResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NextClosestRatingPuzzleIdResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NextClosestRatingPuzzleIdResponse;
  static deserializeBinaryFromReader(message: NextClosestRatingPuzzleIdResponse, reader: jspb.BinaryReader): NextClosestRatingPuzzleIdResponse;
}

export namespace NextClosestRatingPuzzleIdResponse {
  export type AsObject = {
    puzzleId: string,
    queryResult: PuzzleQueryResultMap[keyof PuzzleQueryResultMap],
  }
}

export class PuzzleRequest extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleRequest): PuzzleRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleRequest;
  static deserializeBinaryFromReader(message: PuzzleRequest, reader: jspb.BinaryReader): PuzzleRequest;
}

export namespace PuzzleRequest {
  export type AsObject = {
    puzzleId: string,
  }
}

export class AnswerResponse extends jspb.Message {
  hasCorrectAnswer(): boolean;
  clearCorrectAnswer(): void;
  getCorrectAnswer(): macondo_api_proto_macondo_macondo_pb.GameEvent | undefined;
  setCorrectAnswer(value?: macondo_api_proto_macondo_macondo_pb.GameEvent): void;

  getStatus(): PuzzleStatusMap[keyof PuzzleStatusMap];
  setStatus(value: PuzzleStatusMap[keyof PuzzleStatusMap]): void;

  getAttempts(): number;
  setAttempts(value: number): void;

  getGameId(): string;
  setGameId(value: string): void;

  getTurnNumber(): number;
  setTurnNumber(value: number): void;

  getAfterText(): string;
  setAfterText(value: string): void;

  getNewUserRating(): number;
  setNewUserRating(value: number): void;

  getNewPuzzleRating(): number;
  setNewPuzzleRating(value: number): void;

  hasFirstAttemptTime(): boolean;
  clearFirstAttemptTime(): void;
  getFirstAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastAttemptTime(): boolean;
  clearLastAttemptTime(): void;
  getLastAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): AnswerResponse.AsObject;
  static toObject(includeInstance: boolean, msg: AnswerResponse): AnswerResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: AnswerResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): AnswerResponse;
  static deserializeBinaryFromReader(message: AnswerResponse, reader: jspb.BinaryReader): AnswerResponse;
}

export namespace AnswerResponse {
  export type AsObject = {
    correctAnswer?: macondo_api_proto_macondo_macondo_pb.GameEvent.AsObject,
    status: PuzzleStatusMap[keyof PuzzleStatusMap],
    attempts: number,
    gameId: string,
    turnNumber: number,
    afterText: string,
    newUserRating: number,
    newPuzzleRating: number,
    firstAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class PuzzleResponse extends jspb.Message {
  hasHistory(): boolean;
  clearHistory(): void;
  getHistory(): macondo_api_proto_macondo_macondo_pb.GameHistory | undefined;
  setHistory(value?: macondo_api_proto_macondo_macondo_pb.GameHistory): void;

  getBeforeText(): string;
  setBeforeText(value: string): void;

  hasAnswer(): boolean;
  clearAnswer(): void;
  getAnswer(): AnswerResponse | undefined;
  setAnswer(value?: AnswerResponse): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleResponse): PuzzleResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleResponse;
  static deserializeBinaryFromReader(message: PuzzleResponse, reader: jspb.BinaryReader): PuzzleResponse;
}

export namespace PuzzleResponse {
  export type AsObject = {
    history?: macondo_api_proto_macondo_macondo_pb.GameHistory.AsObject,
    beforeText: string,
    answer?: AnswerResponse.AsObject,
  }
}

export class SubmissionRequest extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  hasAnswer(): boolean;
  clearAnswer(): void;
  getAnswer(): api_proto_ipc_omgwords_pb.ClientGameplayEvent | undefined;
  setAnswer(value?: api_proto_ipc_omgwords_pb.ClientGameplayEvent): void;

  getShowSolution(): boolean;
  setShowSolution(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubmissionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SubmissionRequest): SubmissionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubmissionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubmissionRequest;
  static deserializeBinaryFromReader(message: SubmissionRequest, reader: jspb.BinaryReader): SubmissionRequest;
}

export namespace SubmissionRequest {
  export type AsObject = {
    puzzleId: string,
    answer?: api_proto_ipc_omgwords_pb.ClientGameplayEvent.AsObject,
    showSolution: boolean,
  }
}

export class SubmissionResponse extends jspb.Message {
  getUserIsCorrect(): boolean;
  setUserIsCorrect(value: boolean): void;

  hasAnswer(): boolean;
  clearAnswer(): void;
  getAnswer(): AnswerResponse | undefined;
  setAnswer(value?: AnswerResponse): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SubmissionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SubmissionResponse): SubmissionResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SubmissionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SubmissionResponse;
  static deserializeBinaryFromReader(message: SubmissionResponse, reader: jspb.BinaryReader): SubmissionResponse;
}

export namespace SubmissionResponse {
  export type AsObject = {
    userIsCorrect: boolean,
    answer?: AnswerResponse.AsObject,
  }
}

export class PreviousPuzzleRequest extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PreviousPuzzleRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PreviousPuzzleRequest): PreviousPuzzleRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PreviousPuzzleRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PreviousPuzzleRequest;
  static deserializeBinaryFromReader(message: PreviousPuzzleRequest, reader: jspb.BinaryReader): PreviousPuzzleRequest;
}

export namespace PreviousPuzzleRequest {
  export type AsObject = {
    puzzleId: string,
  }
}

export class PreviousPuzzleResponse extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PreviousPuzzleResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PreviousPuzzleResponse): PreviousPuzzleResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PreviousPuzzleResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PreviousPuzzleResponse;
  static deserializeBinaryFromReader(message: PreviousPuzzleResponse, reader: jspb.BinaryReader): PreviousPuzzleResponse;
}

export namespace PreviousPuzzleResponse {
  export type AsObject = {
    puzzleId: string,
  }
}

export class PuzzleVoteRequest extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  getVote(): number;
  setVote(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleVoteRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleVoteRequest): PuzzleVoteRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleVoteRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleVoteRequest;
  static deserializeBinaryFromReader(message: PuzzleVoteRequest, reader: jspb.BinaryReader): PuzzleVoteRequest;
}

export namespace PuzzleVoteRequest {
  export type AsObject = {
    puzzleId: string,
    vote: number,
  }
}

export class PuzzleVoteResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleVoteResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleVoteResponse): PuzzleVoteResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleVoteResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleVoteResponse;
  static deserializeBinaryFromReader(message: PuzzleVoteResponse, reader: jspb.BinaryReader): PuzzleVoteResponse;
}

export namespace PuzzleVoteResponse {
  export type AsObject = {
  }
}

export class PuzzleGenerationJobRequest extends jspb.Message {
  getBotVsBot(): boolean;
  setBotVsBot(value: boolean): void;

  getLexicon(): string;
  setLexicon(value: string): void;

  getLetterDistribution(): string;
  setLetterDistribution(value: string): void;

  getSqlOffset(): number;
  setSqlOffset(value: number): void;

  getGameConsiderationLimit(): number;
  setGameConsiderationLimit(value: number): void;

  getGameCreationLimit(): number;
  setGameCreationLimit(value: number): void;

  hasRequest(): boolean;
  clearRequest(): void;
  getRequest(): macondo_api_proto_macondo_macondo_pb.PuzzleGenerationRequest | undefined;
  setRequest(value?: macondo_api_proto_macondo_macondo_pb.PuzzleGenerationRequest): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleGenerationJobRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleGenerationJobRequest): PuzzleGenerationJobRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleGenerationJobRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleGenerationJobRequest;
  static deserializeBinaryFromReader(message: PuzzleGenerationJobRequest, reader: jspb.BinaryReader): PuzzleGenerationJobRequest;
}

export namespace PuzzleGenerationJobRequest {
  export type AsObject = {
    botVsBot: boolean,
    lexicon: string,
    letterDistribution: string,
    sqlOffset: number,
    gameConsiderationLimit: number,
    gameCreationLimit: number,
    request?: macondo_api_proto_macondo_macondo_pb.PuzzleGenerationRequest.AsObject,
  }
}

export class APIPuzzleGenerationJobResponse extends jspb.Message {
  getStarted(): boolean;
  setStarted(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): APIPuzzleGenerationJobResponse.AsObject;
  static toObject(includeInstance: boolean, msg: APIPuzzleGenerationJobResponse): APIPuzzleGenerationJobResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: APIPuzzleGenerationJobResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): APIPuzzleGenerationJobResponse;
  static deserializeBinaryFromReader(message: APIPuzzleGenerationJobResponse, reader: jspb.BinaryReader): APIPuzzleGenerationJobResponse;
}

export namespace APIPuzzleGenerationJobResponse {
  export type AsObject = {
    started: boolean,
  }
}

export class APIPuzzleGenerationJobRequest extends jspb.Message {
  hasRequest(): boolean;
  clearRequest(): void;
  getRequest(): PuzzleGenerationJobRequest | undefined;
  setRequest(value?: PuzzleGenerationJobRequest): void;

  getSecretKey(): string;
  setSecretKey(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): APIPuzzleGenerationJobRequest.AsObject;
  static toObject(includeInstance: boolean, msg: APIPuzzleGenerationJobRequest): APIPuzzleGenerationJobRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: APIPuzzleGenerationJobRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): APIPuzzleGenerationJobRequest;
  static deserializeBinaryFromReader(message: APIPuzzleGenerationJobRequest, reader: jspb.BinaryReader): APIPuzzleGenerationJobRequest;
}

export namespace APIPuzzleGenerationJobRequest {
  export type AsObject = {
    request?: PuzzleGenerationJobRequest.AsObject,
    secretKey: string,
  }
}

export class PuzzleJobLogsRequest extends jspb.Message {
  getOffset(): number;
  setOffset(value: number): void;

  getLimit(): number;
  setLimit(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleJobLogsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleJobLogsRequest): PuzzleJobLogsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleJobLogsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleJobLogsRequest;
  static deserializeBinaryFromReader(message: PuzzleJobLogsRequest, reader: jspb.BinaryReader): PuzzleJobLogsRequest;
}

export namespace PuzzleJobLogsRequest {
  export type AsObject = {
    offset: number,
    limit: number,
  }
}

export class PuzzleJobLog extends jspb.Message {
  getId(): number;
  setId(value: number): void;

  hasRequest(): boolean;
  clearRequest(): void;
  getRequest(): PuzzleGenerationJobRequest | undefined;
  setRequest(value?: PuzzleGenerationJobRequest): void;

  getFulfilled(): boolean;
  setFulfilled(value: boolean): void;

  getErrorStatus(): string;
  setErrorStatus(value: string): void;

  hasCreatedAt(): boolean;
  clearCreatedAt(): void;
  getCreatedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCreatedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasCompletedAt(): boolean;
  clearCompletedAt(): void;
  getCompletedAt(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setCompletedAt(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleJobLog.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleJobLog): PuzzleJobLog.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleJobLog, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleJobLog;
  static deserializeBinaryFromReader(message: PuzzleJobLog, reader: jspb.BinaryReader): PuzzleJobLog;
}

export namespace PuzzleJobLog {
  export type AsObject = {
    id: number,
    request?: PuzzleGenerationJobRequest.AsObject,
    fulfilled: boolean,
    errorStatus: string,
    createdAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    completedAt?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class PuzzleJobLogsResponse extends jspb.Message {
  clearLogsList(): void;
  getLogsList(): Array<PuzzleJobLog>;
  setLogsList(value: Array<PuzzleJobLog>): void;
  addLogs(value?: PuzzleJobLog, index?: number): PuzzleJobLog;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PuzzleJobLogsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PuzzleJobLogsResponse): PuzzleJobLogsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PuzzleJobLogsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PuzzleJobLogsResponse;
  static deserializeBinaryFromReader(message: PuzzleJobLogsResponse, reader: jspb.BinaryReader): PuzzleJobLogsResponse;
}

export namespace PuzzleJobLogsResponse {
  export type AsObject = {
    logsList: Array<PuzzleJobLog.AsObject>,
  }
}

export interface PuzzleQueryResultMap {
  UNSEEN: 0;
  UNRATED: 1;
  UNFINISHED: 2;
  EXHAUSTED: 3;
  RANDOM: 4;
  START: 5;
}

export const PuzzleQueryResult: PuzzleQueryResultMap;

export interface PuzzleStatusMap {
  UNANSWERED: 0;
  CORRECT: 1;
  INCORRECT: 2;
}

export const PuzzleStatus: PuzzleStatusMap;

