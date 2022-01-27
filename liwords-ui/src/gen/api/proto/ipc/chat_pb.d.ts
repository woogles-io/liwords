// package: ipc
// file: api/proto/ipc/chat.proto

import * as jspb from "google-protobuf";

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

