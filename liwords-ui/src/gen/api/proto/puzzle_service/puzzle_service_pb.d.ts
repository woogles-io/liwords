// package: puzzle_service
// file: api/proto/puzzle_service/puzzle_service.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as api_proto_ipc_omgwords_pb from "../../../api/proto/ipc/omgwords_pb";

export class RandomUnansweredPuzzleIdRequest extends jspb.Message {
  getLexicon(): string;
  setLexicon(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RandomUnansweredPuzzleIdRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RandomUnansweredPuzzleIdRequest): RandomUnansweredPuzzleIdRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RandomUnansweredPuzzleIdRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RandomUnansweredPuzzleIdRequest;
  static deserializeBinaryFromReader(message: RandomUnansweredPuzzleIdRequest, reader: jspb.BinaryReader): RandomUnansweredPuzzleIdRequest;
}

export namespace RandomUnansweredPuzzleIdRequest {
  export type AsObject = {
    lexicon: string,
  }
}

export class RandomUnansweredPuzzleIdResponse extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RandomUnansweredPuzzleIdResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RandomUnansweredPuzzleIdResponse): RandomUnansweredPuzzleIdResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RandomUnansweredPuzzleIdResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RandomUnansweredPuzzleIdResponse;
  static deserializeBinaryFromReader(message: RandomUnansweredPuzzleIdResponse, reader: jspb.BinaryReader): RandomUnansweredPuzzleIdResponse;
}

export namespace RandomUnansweredPuzzleIdResponse {
  export type AsObject = {
    puzzleId: string,
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

export class PuzzleResponse extends jspb.Message {
  hasHistory(): boolean;
  clearHistory(): void;
  getHistory(): macondo_api_proto_macondo_macondo_pb.GameHistory | undefined;
  setHistory(value?: macondo_api_proto_macondo_macondo_pb.GameHistory): void;

  getBeforeText(): string;
  setBeforeText(value: string): void;

  getAttempts(): number;
  setAttempts(value: number): void;

  getStatus(): PuzzleStatusMap[keyof PuzzleStatusMap];
  setStatus(value: PuzzleStatusMap[keyof PuzzleStatusMap]): void;

  hasFirstAttemptTime(): boolean;
  clearFirstAttemptTime(): void;
  getFirstAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastAttemptTime(): boolean;
  clearLastAttemptTime(): void;
  getLastAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

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
    attempts: number,
    status: PuzzleStatusMap[keyof PuzzleStatusMap],
    firstAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
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
  getStatus(): PuzzleStatusMap[keyof PuzzleStatusMap];
  setStatus(value: PuzzleStatusMap[keyof PuzzleStatusMap]): void;

  hasCorrectAnswer(): boolean;
  clearCorrectAnswer(): void;
  getCorrectAnswer(): macondo_api_proto_macondo_macondo_pb.GameEvent | undefined;
  setCorrectAnswer(value?: macondo_api_proto_macondo_macondo_pb.GameEvent): void;

  getGameId(): string;
  setGameId(value: string): void;

  getAfterText(): string;
  setAfterText(value: string): void;

  getAttempts(): number;
  setAttempts(value: number): void;

  hasFirstAttemptTime(): boolean;
  clearFirstAttemptTime(): void;
  getFirstAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setFirstAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  hasLastAttemptTime(): boolean;
  clearLastAttemptTime(): void;
  getLastAttemptTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setLastAttemptTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

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
    status: PuzzleStatusMap[keyof PuzzleStatusMap],
    correctAnswer?: macondo_api_proto_macondo_macondo_pb.GameEvent.AsObject,
    gameId: string,
    afterText: string,
    attempts: number,
    firstAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    lastAttemptTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
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

export interface PuzzleStatusMap {
  UNANSWERED: 0;
  CORRECT: 1;
  INCORRECT: 2;
}

export const PuzzleStatus: PuzzleStatusMap;

