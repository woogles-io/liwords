// package: ipc
// file: api/proto/ipc/ipc.proto

import * as jspb from "google-protobuf";

export class RegisterRealmRequest extends jspb.Message {
  getPath(): string;
  setPath(value: string): void;

  getUserId(): string;
  setUserId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterRealmRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RegisterRealmRequest): RegisterRealmRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RegisterRealmRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegisterRealmRequest;
  static deserializeBinaryFromReader(message: RegisterRealmRequest, reader: jspb.BinaryReader): RegisterRealmRequest;
}

export namespace RegisterRealmRequest {
  export type AsObject = {
    path: string,
    userId: string,
  }
}

export class RegisterRealmResponse extends jspb.Message {
  clearRealmsList(): void;
  getRealmsList(): Array<string>;
  setRealmsList(value: Array<string>): void;
  addRealms(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RegisterRealmResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RegisterRealmResponse): RegisterRealmResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RegisterRealmResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RegisterRealmResponse;
  static deserializeBinaryFromReader(message: RegisterRealmResponse, reader: jspb.BinaryReader): RegisterRealmResponse;
}

export namespace RegisterRealmResponse {
  export type AsObject = {
    realmsList: Array<string>,
  }
}

export class InitRealmInfo extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  clearRealmsList(): void;
  getRealmsList(): Array<string>;
  setRealmsList(value: Array<string>): void;
  addRealms(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): InitRealmInfo.AsObject;
  static toObject(includeInstance: boolean, msg: InitRealmInfo): InitRealmInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: InitRealmInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): InitRealmInfo;
  static deserializeBinaryFromReader(message: InitRealmInfo, reader: jspb.BinaryReader): InitRealmInfo;
}

export namespace InitRealmInfo {
  export type AsObject = {
    userId: string,
    realmsList: Array<string>,
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

export class Pong extends jspb.Message {
  getIps(): string;
  setIps(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Pong.AsObject;
  static toObject(includeInstance: boolean, msg: Pong): Pong.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Pong, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Pong;
  static deserializeBinaryFromReader(message: Pong, reader: jspb.BinaryReader): Pong;
}

export namespace Pong {
  export type AsObject = {
    ips: string,
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
  ONGOING_GAME_EVENT: 12;
  TIMED_OUT: 13;
  ONGOING_GAMES: 14;
  READY_FOR_TOURNAMENT_GAME: 15;
  TOURNAMENT_ROUND_STARTED: 16;
  GAME_DELETION: 17;
  MATCH_REQUESTS: 18;
  DECLINE_SEEK_REQUEST: 19;
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
  PROFILE_UPDATE_EVENT: 43;
  GAME_INSTANTIATION: 44;
}

export const MessageType: MessageTypeMap;

