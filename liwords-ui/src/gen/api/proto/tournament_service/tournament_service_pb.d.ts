// package: tournament_service
// file: api/proto/tournament_service/tournament_service.proto

import * as jspb from "google-protobuf";
import * as api_proto_realtime_realtime_pb from "../../../api/proto/realtime/realtime_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class StartRoundRequest extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): StartRoundRequest.AsObject;
  static toObject(includeInstance: boolean, msg: StartRoundRequest): StartRoundRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: StartRoundRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): StartRoundRequest;
  static deserializeBinaryFromReader(message: StartRoundRequest, reader: jspb.BinaryReader): StartRoundRequest;
}

export namespace StartRoundRequest {
  export type AsObject = {
    tournamentId: string,
    round: number,
  }
}

export class NewTournamentRequest extends jspb.Message {
  getSlug(): string;
  setSlug(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  clearDirectorUsernamesList(): void;
  getDirectorUsernamesList(): Array<string>;
  setDirectorUsernamesList(value: Array<string>): void;
  addDirectorUsernames(value: string, index?: number): string;

  getType(): TTypeMap[keyof TTypeMap];
  setType(value: TTypeMap[keyof TTypeMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NewTournamentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: NewTournamentRequest): NewTournamentRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NewTournamentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NewTournamentRequest;
  static deserializeBinaryFromReader(message: NewTournamentRequest, reader: jspb.BinaryReader): NewTournamentRequest;
}

export namespace NewTournamentRequest {
  export type AsObject = {
    slug: string,
    name: string,
    description: string,
    directorUsernamesList: Array<string>,
    type: TTypeMap[keyof TTypeMap],
  }
}

export class TournamentMetadata extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getSlug(): string;
  setSlug(value: string): void;

  getType(): TTypeMap[keyof TTypeMap];
  setType(value: TTypeMap[keyof TTypeMap]): void;

  getDisclaimer(): string;
  setDisclaimer(value: string): void;

  getTileStyle(): string;
  setTileStyle(value: string): void;

  getBoardStyle(): string;
  setBoardStyle(value: string): void;

  hasDefaultClubSettings(): boolean;
  clearDefaultClubSettings(): void;
  getDefaultClubSettings(): api_proto_realtime_realtime_pb.GameRequest | undefined;
  setDefaultClubSettings(value?: api_proto_realtime_realtime_pb.GameRequest): void;

  clearFreeformClubSettingFieldsList(): void;
  getFreeformClubSettingFieldsList(): Array<string>;
  setFreeformClubSettingFieldsList(value: Array<string>): void;
  addFreeformClubSettingFields(value: string, index?: number): string;

  getPassword(): string;
  setPassword(value: string): void;

  getLogo(): string;
  setLogo(value: string): void;

  getColor(): string;
  setColor(value: string): void;

  getPrivateAnalysis(): boolean;
  setPrivateAnalysis(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentMetadata.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentMetadata): TournamentMetadata.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentMetadata, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentMetadata;
  static deserializeBinaryFromReader(message: TournamentMetadata, reader: jspb.BinaryReader): TournamentMetadata;
}

export namespace TournamentMetadata {
  export type AsObject = {
    id: string,
    name: string,
    description: string,
    slug: string,
    type: TTypeMap[keyof TTypeMap],
    disclaimer: string,
    tileStyle: string,
    boardStyle: string,
    defaultClubSettings?: api_proto_realtime_realtime_pb.GameRequest.AsObject,
    freeformClubSettingFieldsList: Array<string>,
    password: string,
    logo: string,
    color: string,
    privateAnalysis: boolean,
  }
}

export class SetTournamentMetadataRequest extends jspb.Message {
  hasMetadata(): boolean;
  clearMetadata(): void;
  getMetadata(): TournamentMetadata | undefined;
  setMetadata(value?: TournamentMetadata): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetTournamentMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SetTournamentMetadataRequest): SetTournamentMetadataRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SetTournamentMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetTournamentMetadataRequest;
  static deserializeBinaryFromReader(message: SetTournamentMetadataRequest, reader: jspb.BinaryReader): SetTournamentMetadataRequest;
}

export namespace SetTournamentMetadataRequest {
  export type AsObject = {
    metadata?: TournamentMetadata.AsObject,
  }
}

export class SingleRoundControlsRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  hasRoundControls(): boolean;
  clearRoundControls(): void;
  getRoundControls(): api_proto_realtime_realtime_pb.RoundControl | undefined;
  setRoundControls(value?: api_proto_realtime_realtime_pb.RoundControl): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SingleRoundControlsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SingleRoundControlsRequest): SingleRoundControlsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: SingleRoundControlsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SingleRoundControlsRequest;
  static deserializeBinaryFromReader(message: SingleRoundControlsRequest, reader: jspb.BinaryReader): SingleRoundControlsRequest;
}

export namespace SingleRoundControlsRequest {
  export type AsObject = {
    id: string,
    division: string,
    round: number,
    roundControls?: api_proto_realtime_realtime_pb.RoundControl.AsObject,
  }
}

export class PairRoundRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getPreserveByes(): boolean;
  setPreserveByes(value: boolean): void;

  getDeletePairings(): boolean;
  setDeletePairings(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PairRoundRequest.AsObject;
  static toObject(includeInstance: boolean, msg: PairRoundRequest): PairRoundRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PairRoundRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PairRoundRequest;
  static deserializeBinaryFromReader(message: PairRoundRequest, reader: jspb.BinaryReader): PairRoundRequest;
}

export namespace PairRoundRequest {
  export type AsObject = {
    id: string,
    division: string,
    round: number,
    preserveByes: boolean,
    deletePairings: boolean,
  }
}

export class TournamentDivisionRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDivisionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDivisionRequest): TournamentDivisionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDivisionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDivisionRequest;
  static deserializeBinaryFromReader(message: TournamentDivisionRequest, reader: jspb.BinaryReader): TournamentDivisionRequest;
}

export namespace TournamentDivisionRequest {
  export type AsObject = {
    id: string,
    division: string,
  }
}

export class TournamentPairingRequest extends jspb.Message {
  getPlayerOneId(): string;
  setPlayerOneId(value: string): void;

  getPlayerTwoId(): string;
  setPlayerTwoId(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getSelfPlayResult(): api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap];
  setSelfPlayResult(value: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap]): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPairingRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPairingRequest): TournamentPairingRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPairingRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPairingRequest;
  static deserializeBinaryFromReader(message: TournamentPairingRequest, reader: jspb.BinaryReader): TournamentPairingRequest;
}

export namespace TournamentPairingRequest {
  export type AsObject = {
    playerOneId: string,
    playerTwoId: string,
    round: number,
    selfPlayResult: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap],
  }
}

export class TournamentPairingsRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearPairingsList(): void;
  getPairingsList(): Array<TournamentPairingRequest>;
  setPairingsList(value: Array<TournamentPairingRequest>): void;
  addPairings(value?: TournamentPairingRequest, index?: number): TournamentPairingRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPairingsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPairingsRequest): TournamentPairingsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPairingsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPairingsRequest;
  static deserializeBinaryFromReader(message: TournamentPairingsRequest, reader: jspb.BinaryReader): TournamentPairingsRequest;
}

export namespace TournamentPairingsRequest {
  export type AsObject = {
    id: string,
    division: string,
    pairingsList: Array<TournamentPairingRequest.AsObject>,
  }
}

export class TournamentResultOverrideRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getPlayerOneId(): string;
  setPlayerOneId(value: string): void;

  getPlayerTwoId(): string;
  setPlayerTwoId(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getPlayerOneScore(): number;
  setPlayerOneScore(value: number): void;

  getPlayerTwoScore(): number;
  setPlayerTwoScore(value: number): void;

  getPlayerOneResult(): api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap];
  setPlayerOneResult(value: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap]): void;

  getPlayerTwoResult(): api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap];
  setPlayerTwoResult(value: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap]): void;

  getGameEndReason(): api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap];
  setGameEndReason(value: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap]): void;

  getAmendment(): boolean;
  setAmendment(value: boolean): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentResultOverrideRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentResultOverrideRequest): TournamentResultOverrideRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentResultOverrideRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentResultOverrideRequest;
  static deserializeBinaryFromReader(message: TournamentResultOverrideRequest, reader: jspb.BinaryReader): TournamentResultOverrideRequest;
}

export namespace TournamentResultOverrideRequest {
  export type AsObject = {
    id: string,
    division: string,
    playerOneId: string,
    playerTwoId: string,
    round: number,
    playerOneScore: number,
    playerTwoScore: number,
    playerOneResult: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap],
    playerTwoResult: api_proto_realtime_realtime_pb.TournamentGameResultMap[keyof api_proto_realtime_realtime_pb.TournamentGameResultMap],
    gameEndReason: api_proto_realtime_realtime_pb.GameEndReasonMap[keyof api_proto_realtime_realtime_pb.GameEndReasonMap],
    amendment: boolean,
    gameIndex: number,
  }
}

export class TournamentStartRoundCountdownRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getStartAllRounds(): boolean;
  setStartAllRounds(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentStartRoundCountdownRequest.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentStartRoundCountdownRequest): TournamentStartRoundCountdownRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentStartRoundCountdownRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentStartRoundCountdownRequest;
  static deserializeBinaryFromReader(message: TournamentStartRoundCountdownRequest, reader: jspb.BinaryReader): TournamentStartRoundCountdownRequest;
}

export namespace TournamentStartRoundCountdownRequest {
  export type AsObject = {
    id: string,
    division: string,
    round: number,
    startAllRounds: boolean,
  }
}

export class TournamentResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentResponse): TournamentResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentResponse;
  static deserializeBinaryFromReader(message: TournamentResponse, reader: jspb.BinaryReader): TournamentResponse;
}

export namespace TournamentResponse {
  export type AsObject = {
  }
}

export class NewTournamentResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getSlug(): string;
  setSlug(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NewTournamentResponse.AsObject;
  static toObject(includeInstance: boolean, msg: NewTournamentResponse): NewTournamentResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NewTournamentResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NewTournamentResponse;
  static deserializeBinaryFromReader(message: NewTournamentResponse, reader: jspb.BinaryReader): NewTournamentResponse;
}

export namespace NewTournamentResponse {
  export type AsObject = {
    id: string,
    slug: string,
  }
}

export class GetTournamentMetadataRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getSlug(): string;
  setSlug(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTournamentMetadataRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTournamentMetadataRequest): GetTournamentMetadataRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetTournamentMetadataRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTournamentMetadataRequest;
  static deserializeBinaryFromReader(message: GetTournamentMetadataRequest, reader: jspb.BinaryReader): GetTournamentMetadataRequest;
}

export namespace GetTournamentMetadataRequest {
  export type AsObject = {
    id: string,
    slug: string,
  }
}

export class GetTournamentRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetTournamentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetTournamentRequest): GetTournamentRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: GetTournamentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetTournamentRequest;
  static deserializeBinaryFromReader(message: GetTournamentRequest, reader: jspb.BinaryReader): GetTournamentRequest;
}

export namespace GetTournamentRequest {
  export type AsObject = {
    id: string,
  }
}

export class FinishTournamentRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FinishTournamentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: FinishTournamentRequest): FinishTournamentRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FinishTournamentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FinishTournamentRequest;
  static deserializeBinaryFromReader(message: FinishTournamentRequest, reader: jspb.BinaryReader): FinishTournamentRequest;
}

export namespace FinishTournamentRequest {
  export type AsObject = {
    id: string,
  }
}

export class TournamentMetadataResponse extends jspb.Message {
  hasMetadata(): boolean;
  clearMetadata(): void;
  getMetadata(): TournamentMetadata | undefined;
  setMetadata(value?: TournamentMetadata): void;

  clearDirectorsList(): void;
  getDirectorsList(): Array<string>;
  setDirectorsList(value: Array<string>): void;
  addDirectors(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentMetadataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentMetadataResponse): TournamentMetadataResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentMetadataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentMetadataResponse;
  static deserializeBinaryFromReader(message: TournamentMetadataResponse, reader: jspb.BinaryReader): TournamentMetadataResponse;
}

export namespace TournamentMetadataResponse {
  export type AsObject = {
    metadata?: TournamentMetadata.AsObject,
    directorsList: Array<string>,
  }
}

export class RecentGamesRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getNumGames(): number;
  setNumGames(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RecentGamesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RecentGamesRequest): RecentGamesRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RecentGamesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RecentGamesRequest;
  static deserializeBinaryFromReader(message: RecentGamesRequest, reader: jspb.BinaryReader): RecentGamesRequest;
}

export namespace RecentGamesRequest {
  export type AsObject = {
    id: string,
    numGames: number,
    offset: number,
  }
}

export class RecentGamesResponse extends jspb.Message {
  clearGamesList(): void;
  getGamesList(): Array<api_proto_realtime_realtime_pb.TournamentGameEndedEvent>;
  setGamesList(value: Array<api_proto_realtime_realtime_pb.TournamentGameEndedEvent>): void;
  addGames(value?: api_proto_realtime_realtime_pb.TournamentGameEndedEvent, index?: number): api_proto_realtime_realtime_pb.TournamentGameEndedEvent;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RecentGamesResponse.AsObject;
  static toObject(includeInstance: boolean, msg: RecentGamesResponse): RecentGamesResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RecentGamesResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RecentGamesResponse;
  static deserializeBinaryFromReader(message: RecentGamesResponse, reader: jspb.BinaryReader): RecentGamesResponse;
}

export namespace RecentGamesResponse {
  export type AsObject = {
    gamesList: Array<api_proto_realtime_realtime_pb.TournamentGameEndedEvent.AsObject>,
  }
}

export class UnstartTournamentRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UnstartTournamentRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UnstartTournamentRequest): UnstartTournamentRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UnstartTournamentRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UnstartTournamentRequest;
  static deserializeBinaryFromReader(message: UnstartTournamentRequest, reader: jspb.BinaryReader): UnstartTournamentRequest;
}

export namespace UnstartTournamentRequest {
  export type AsObject = {
    id: string,
  }
}

export class UncheckInRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): UncheckInRequest.AsObject;
  static toObject(includeInstance: boolean, msg: UncheckInRequest): UncheckInRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: UncheckInRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): UncheckInRequest;
  static deserializeBinaryFromReader(message: UncheckInRequest, reader: jspb.BinaryReader): UncheckInRequest;
}

export namespace UncheckInRequest {
  export type AsObject = {
    id: string,
  }
}

export class CheckinRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): CheckinRequest.AsObject;
  static toObject(includeInstance: boolean, msg: CheckinRequest): CheckinRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: CheckinRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): CheckinRequest;
  static deserializeBinaryFromReader(message: CheckinRequest, reader: jspb.BinaryReader): CheckinRequest;
}

export namespace CheckinRequest {
  export type AsObject = {
    id: string,
  }
}

export class DisassociateClubGameRequest extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getGameId(): string;
  setGameId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DisassociateClubGameRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DisassociateClubGameRequest): DisassociateClubGameRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DisassociateClubGameRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DisassociateClubGameRequest;
  static deserializeBinaryFromReader(message: DisassociateClubGameRequest, reader: jspb.BinaryReader): DisassociateClubGameRequest;
}

export namespace DisassociateClubGameRequest {
  export type AsObject = {
    tournamentId: string,
    gameId: string,
  }
}

export class NewClubSessionRequest extends jspb.Message {
  hasDate(): boolean;
  clearDate(): void;
  getDate(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDate(value?: google_protobuf_timestamp_pb.Timestamp): void;

  getClubId(): string;
  setClubId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NewClubSessionRequest.AsObject;
  static toObject(includeInstance: boolean, msg: NewClubSessionRequest): NewClubSessionRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NewClubSessionRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NewClubSessionRequest;
  static deserializeBinaryFromReader(message: NewClubSessionRequest, reader: jspb.BinaryReader): NewClubSessionRequest;
}

export namespace NewClubSessionRequest {
  export type AsObject = {
    date?: google_protobuf_timestamp_pb.Timestamp.AsObject,
    clubId: string,
  }
}

export class ClubSessionResponse extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getSlug(): string;
  setSlug(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClubSessionResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ClubSessionResponse): ClubSessionResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClubSessionResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClubSessionResponse;
  static deserializeBinaryFromReader(message: ClubSessionResponse, reader: jspb.BinaryReader): ClubSessionResponse;
}

export namespace ClubSessionResponse {
  export type AsObject = {
    tournamentId: string,
    slug: string,
  }
}

export class RecentClubSessionsRequest extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getCount(): number;
  setCount(value: number): void;

  getOffset(): number;
  setOffset(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RecentClubSessionsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: RecentClubSessionsRequest): RecentClubSessionsRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RecentClubSessionsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RecentClubSessionsRequest;
  static deserializeBinaryFromReader(message: RecentClubSessionsRequest, reader: jspb.BinaryReader): RecentClubSessionsRequest;
}

export namespace RecentClubSessionsRequest {
  export type AsObject = {
    id: string,
    count: number,
    offset: number,
  }
}

export class ClubSessionsResponse extends jspb.Message {
  clearSessionsList(): void;
  getSessionsList(): Array<ClubSessionResponse>;
  setSessionsList(value: Array<ClubSessionResponse>): void;
  addSessions(value?: ClubSessionResponse, index?: number): ClubSessionResponse;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ClubSessionsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: ClubSessionsResponse): ClubSessionsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ClubSessionsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ClubSessionsResponse;
  static deserializeBinaryFromReader(message: ClubSessionsResponse, reader: jspb.BinaryReader): ClubSessionsResponse;
}

export namespace ClubSessionsResponse {
  export type AsObject = {
    sessionsList: Array<ClubSessionResponse.AsObject>,
  }
}

export interface TTypeMap {
  STANDARD: 0;
  CLUB: 1;
  CHILD: 2;
  LEGACY: 3;
}

export const TType: TTypeMap;

