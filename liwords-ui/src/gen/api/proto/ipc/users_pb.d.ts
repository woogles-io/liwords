// package: ipc
// file: api/proto/ipc/users.proto

import * as jspb from "google-protobuf";

export class ProfileUpdate extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  getRatingsMap(): jspb.Map<string, ProfileUpdate.Rating>;
  clearRatingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ProfileUpdate.AsObject;
  static toObject(includeInstance: boolean, msg: ProfileUpdate): ProfileUpdate.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ProfileUpdate, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ProfileUpdate;
  static deserializeBinaryFromReader(message: ProfileUpdate, reader: jspb.BinaryReader): ProfileUpdate;
}

export namespace ProfileUpdate {
  export type AsObject = {
    userId: string,
    ratingsMap: Array<[string, ProfileUpdate.Rating.AsObject]>,
  }

  export class Rating extends jspb.Message {
    getRating(): number;
    setRating(value: number): void;

    getDeviation(): number;
    setDeviation(value: number): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Rating.AsObject;
    static toObject(includeInstance: boolean, msg: Rating): Rating.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Rating, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Rating;
    static deserializeBinaryFromReader(message: Rating, reader: jspb.BinaryReader): Rating;
  }

  export namespace Rating {
    export type AsObject = {
      rating: number,
      deviation: number,
    }
  }
}

export interface ChildStatusMap {
  CHILD: 0;
  NOT_CHILD: 1;
  UNKNOWN: 2;
}

export const ChildStatus: ChildStatusMap;

