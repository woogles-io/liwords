// package: game_service
// file: api/proto/game_service/game_service.proto

import * as jspb from "google-protobuf";
import * as api_proto_realtime_realtime_pb from "../../../api/proto/realtime/realtime_pb";
import * as macondo_api_proto_macondo_macondo_pb from "../../../macondo/api/proto/macondo/macondo_pb";

export class GameInfoRequest extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameInfoRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GameInfoRequest): GameInfoRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameInfoRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameInfoRequest;
  static deserializeBinaryFromReader(message: GameInfoRequest, reader: jspb.BinaryReader): GameInfoRequest;
}

export namespace GameInfoRequest {
  export type AsObject = {
    gameId: string,
  }
}

export class PlayerInfo extends jspb.Message {
  getUserId(): string;
  setUserId(value: string): void;

  getNickname(): string;
  setNickname(value: string): void;

  getFullName(): string;
  setFullName(value: string): void;

  getCountryCode(): string;
  setCountryCode(value: string): void;

  getRating(): string;
  setRating(value: string): void;

  getTitle(): string;
  setTitle(value: string): void;

  getAvatarUrl(): string;
  setAvatarUrl(value: string): void;

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
    userId: string,
    nickname: string,
    fullName: string,
    countryCode: string,
    rating: string,
    title: string,
    avatarUrl: string,
  }
}

export class GameInfoResponse extends jspb.Message {
  clearPlayersList(): void;
  getPlayersList(): Array<PlayerInfo>;
  setPlayersList(value: Array<PlayerInfo>): void;
  addPlayers(value?: PlayerInfo, index?: number): PlayerInfo;

  getLexicon(): string;
  setLexicon(value: string): void;

  getVariant(): string;
  setVariant(value: string): void;

  getTimeControlName(): string;
  setTimeControlName(value: string): void;

  getTimeControl(): string;
  setTimeControl(value: string): void;

  getTournamentName(): string;
  setTournamentName(value: string): void;

  getChallengeRule(): macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap];
  setChallengeRule(value: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap]): void;

  getRatingMode(): api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap];
  setRatingMode(value: api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap]): void;

  getDone(): boolean;
  setDone(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GameInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: GameInfoResponse): GameInfoResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GameInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GameInfoResponse;
  static deserializeBinaryFromReader(message: GameInfoResponse, reader: jspb.BinaryReader): GameInfoResponse;
}

export namespace GameInfoResponse {
  export type AsObject = {
    playersList: Array<PlayerInfo.AsObject>,
    lexicon: string,
    variant: string,
    timeControlName: string,
    timeControl: string,
    tournamentName: string,
    challengeRule: macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap[keyof macondo_api_proto_macondo_macondo_pb.ChallengeRuleMap],
    ratingMode: api_proto_realtime_realtime_pb.RatingModeMap[keyof api_proto_realtime_realtime_pb.RatingModeMap],
    done: boolean,
  }
}

