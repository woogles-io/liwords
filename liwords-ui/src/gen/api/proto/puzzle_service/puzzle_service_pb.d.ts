// package: puzzle_service
// file: api/proto/puzzle_service/puzzle_service.proto

import * as jspb from "google-protobuf";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";

export class RandomUnansweredPuzzleIdRequest extends jspb.Message {
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
  }
}

export class SubmissionRequest extends jspb.Message {
  getPuzzleId(): string;
  setPuzzleId(value: string): void;

  hasAnswer(): boolean;
  clearAnswer(): void;
  getAnswer(): macondo_api_proto_macondo_macondo_pb.GameEvent | undefined;
  setAnswer(value?: macondo_api_proto_macondo_macondo_pb.GameEvent): void;

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
    answer?: macondo_api_proto_macondo_macondo_pb.GameEvent.AsObject,
    showSolution: boolean,
  }
}

export class SubmissionResponse extends jspb.Message {
  getUserIsCorrect(): boolean;
  setUserIsCorrect(value: boolean): void;

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
    correctAnswer?: macondo_api_proto_macondo_macondo_pb.GameEvent.AsObject,
    gameId: string,
    afterText: string,
    attempts: number,
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

