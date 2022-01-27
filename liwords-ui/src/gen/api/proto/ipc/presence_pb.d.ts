// package: ipc
// file: api/proto/ipc/presence.proto

import * as jspb from "google-protobuf";

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

