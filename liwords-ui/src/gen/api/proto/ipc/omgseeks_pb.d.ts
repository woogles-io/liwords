// package: ipc
// file: api/proto/ipc/omgseeks.proto

import * as jspb from "google-protobuf";
import * as api_proto_ipc_omgwords_pb from "../../../api/proto/ipc/omgwords_pb";

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

export class SeekRequest extends jspb.Message {
  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): api_proto_ipc_omgwords_pb.GameRequest | undefined;
  setGameRequest(value?: api_proto_ipc_omgwords_pb.GameRequest): void;

  hasUser(): boolean;
  clearUser(): void;
  getUser(): MatchUser | undefined;
  setUser(value?: MatchUser): void;

  getMinimumRatingRange(): number;
  setMinimumRatingRange(value: number): void;

  getMaximumRatingRange(): number;
  setMaximumRatingRange(value: number): void;

  getSeekerConnectionId(): string;
  setSeekerConnectionId(value: string): void;

  hasReceivingUser(): boolean;
  clearReceivingUser(): void;
  getReceivingUser(): MatchUser | undefined;
  setReceivingUser(value?: MatchUser): void;

  getUserState(): SeekStateMap[keyof SeekStateMap];
  setUserState(value: SeekStateMap[keyof SeekStateMap]): void;

  getReceiverState(): SeekStateMap[keyof SeekStateMap];
  setReceiverState(value: SeekStateMap[keyof SeekStateMap]): void;

  getReceiverConnectionId(): string;
  setReceiverConnectionId(value: string): void;

  clearBootedReceiversList(): void;
  getBootedReceiversList(): Array<string>;
  setBootedReceiversList(value: Array<string>): void;
  addBootedReceivers(value: string, index?: number): string;

  getRematchFor(): string;
  setRematchFor(value: string): void;

  getTournamentId(): string;
  setTournamentId(value: string): void;

  getReceiverIsPermanent(): boolean;
  setReceiverIsPermanent(value: boolean): void;

  getRatingKey(): string;
  setRatingKey(value: string): void;

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
    gameRequest?: api_proto_ipc_omgwords_pb.GameRequest.AsObject,
    user?: MatchUser.AsObject,
    minimumRatingRange: number,
    maximumRatingRange: number,
    seekerConnectionId: string,
    receivingUser?: MatchUser.AsObject,
    userState: SeekStateMap[keyof SeekStateMap],
    receiverState: SeekStateMap[keyof SeekStateMap],
    receiverConnectionId: string,
    bootedReceiversList: Array<string>,
    rematchFor: string,
    tournamentId: string,
    receiverIsPermanent: boolean,
    ratingKey: string,
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

export class DeclineSeekRequest extends jspb.Message {
  getRequestId(): string;
  setRequestId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeclineSeekRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeclineSeekRequest): DeclineSeekRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeclineSeekRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeclineSeekRequest;
  static deserializeBinaryFromReader(message: DeclineSeekRequest, reader: jspb.BinaryReader): DeclineSeekRequest;
}

export namespace DeclineSeekRequest {
  export type AsObject = {
    requestId: string,
  }
}

export interface SeekStateMap {
  ABSENT: 0;
  PRESENT: 1;
  READY: 2;
}

export const SeekState: SeekStateMap;

