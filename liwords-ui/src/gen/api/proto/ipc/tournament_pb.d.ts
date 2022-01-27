// package: ipc
// file: api/proto/ipc/tournament.proto

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as api_proto_ipc_omgwords_pb from "../../../api/proto/ipc/omgwords_pb";

export class TournamentGameEndedEvent extends jspb.Message {
  getGameId(): string;
  setGameId(value: string): void;

  clearPlayersList(): void;
  getPlayersList(): Array<TournamentGameEndedEvent.Player>;
  setPlayersList(value: Array<TournamentGameEndedEvent.Player>): void;
  addPlayers(value?: TournamentGameEndedEvent.Player, index?: number): TournamentGameEndedEvent.Player;

  getEndReason(): api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap];
  setEndReason(value: api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap]): void;

  getTime(): number;
  setTime(value: number): void;

  getRound(): number;
  setRound(value: number): void;

  getDivision(): string;
  setDivision(value: string): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentGameEndedEvent.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentGameEndedEvent): TournamentGameEndedEvent.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentGameEndedEvent, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentGameEndedEvent;
  static deserializeBinaryFromReader(message: TournamentGameEndedEvent, reader: jspb.BinaryReader): TournamentGameEndedEvent;
}

export namespace TournamentGameEndedEvent {
  export type AsObject = {
    gameId: string,
    playersList: Array<TournamentGameEndedEvent.Player.AsObject>,
    endReason: api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap],
    time: number,
    round: number,
    division: string,
    gameIndex: number,
  }

  export class Player extends jspb.Message {
    getUsername(): string;
    setUsername(value: string): void;

    getScore(): number;
    setScore(value: number): void;

    getResult(): TournamentGameResultMap[keyof TournamentGameResultMap];
    setResult(value: TournamentGameResultMap[keyof TournamentGameResultMap]): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Player.AsObject;
    static toObject(includeInstance: boolean, msg: Player): Player.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Player, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Player;
    static deserializeBinaryFromReader(message: Player, reader: jspb.BinaryReader): Player;
  }

  export namespace Player {
    export type AsObject = {
      username: string,
      score: number,
      result: TournamentGameResultMap[keyof TournamentGameResultMap],
    }
  }
}

export class TournamentRoundStarted extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  hasDeadline(): boolean;
  clearDeadline(): void;
  getDeadline(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setDeadline(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentRoundStarted.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentRoundStarted): TournamentRoundStarted.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentRoundStarted, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentRoundStarted;
  static deserializeBinaryFromReader(message: TournamentRoundStarted, reader: jspb.BinaryReader): TournamentRoundStarted;
}

export namespace TournamentRoundStarted {
  export type AsObject = {
    tournamentId: string,
    division: string,
    round: number,
    gameIndex: number,
    deadline?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class ReadyForTournamentGame extends jspb.Message {
  getTournamentId(): string;
  setTournamentId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  getPlayerId(): string;
  setPlayerId(value: string): void;

  getGameIndex(): number;
  setGameIndex(value: number): void;

  getUnready(): boolean;
  setUnready(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ReadyForTournamentGame.AsObject;
  static toObject(includeInstance: boolean, msg: ReadyForTournamentGame): ReadyForTournamentGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ReadyForTournamentGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ReadyForTournamentGame;
  static deserializeBinaryFromReader(message: ReadyForTournamentGame, reader: jspb.BinaryReader): ReadyForTournamentGame;
}

export namespace ReadyForTournamentGame {
  export type AsObject = {
    tournamentId: string,
    division: string,
    round: number,
    playerId: string,
    gameIndex: number,
    unready: boolean,
  }
}

export class TournamentPerson extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getRating(): number;
  setRating(value: number): void;

  getSuspended(): boolean;
  setSuspended(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPerson.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPerson): TournamentPerson.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPerson, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPerson;
  static deserializeBinaryFromReader(message: TournamentPerson, reader: jspb.BinaryReader): TournamentPerson;
}

export namespace TournamentPerson {
  export type AsObject = {
    id: string,
    rating: number,
    suspended: boolean,
  }
}

export class TournamentPersons extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearPersonsList(): void;
  getPersonsList(): Array<TournamentPerson>;
  setPersonsList(value: Array<TournamentPerson>): void;
  addPersons(value?: TournamentPerson, index?: number): TournamentPerson;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentPersons.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentPersons): TournamentPersons.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentPersons, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentPersons;
  static deserializeBinaryFromReader(message: TournamentPersons, reader: jspb.BinaryReader): TournamentPersons;
}

export namespace TournamentPersons {
  export type AsObject = {
    id: string,
    division: string,
    personsList: Array<TournamentPerson.AsObject>,
  }
}

export class RoundControl extends jspb.Message {
  getPairingMethod(): PairingMethodMap[keyof PairingMethodMap];
  setPairingMethod(value: PairingMethodMap[keyof PairingMethodMap]): void;

  getFirstMethod(): FirstMethodMap[keyof FirstMethodMap];
  setFirstMethod(value: FirstMethodMap[keyof FirstMethodMap]): void;

  getGamesPerRound(): number;
  setGamesPerRound(value: number): void;

  getRound(): number;
  setRound(value: number): void;

  getFactor(): number;
  setFactor(value: number): void;

  getInitialFontes(): number;
  setInitialFontes(value: number): void;

  getMaxRepeats(): number;
  setMaxRepeats(value: number): void;

  getAllowOverMaxRepeats(): boolean;
  setAllowOverMaxRepeats(value: boolean): void;

  getRepeatRelativeWeight(): number;
  setRepeatRelativeWeight(value: number): void;

  getWinDifferenceRelativeWeight(): number;
  setWinDifferenceRelativeWeight(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RoundControl.AsObject;
  static toObject(includeInstance: boolean, msg: RoundControl): RoundControl.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RoundControl, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RoundControl;
  static deserializeBinaryFromReader(message: RoundControl, reader: jspb.BinaryReader): RoundControl;
}

export namespace RoundControl {
  export type AsObject = {
    pairingMethod: PairingMethodMap[keyof PairingMethodMap],
    firstMethod: FirstMethodMap[keyof FirstMethodMap],
    gamesPerRound: number,
    round: number,
    factor: number,
    initialFontes: number,
    maxRepeats: number,
    allowOverMaxRepeats: boolean,
    repeatRelativeWeight: number,
    winDifferenceRelativeWeight: number,
  }
}

export class DivisionControls extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasGameRequest(): boolean;
  clearGameRequest(): void;
  getGameRequest(): api_proto_ipc_omgwords_pb.GameRequest | undefined;
  setGameRequest(value?: api_proto_ipc_omgwords_pb.GameRequest): void;

  getSuspendedResult(): TournamentGameResultMap[keyof TournamentGameResultMap];
  setSuspendedResult(value: TournamentGameResultMap[keyof TournamentGameResultMap]): void;

  getSuspendedSpread(): number;
  setSuspendedSpread(value: number): void;

  getAutoStart(): boolean;
  setAutoStart(value: boolean): void;

  getSpreadCap(): number;
  setSpreadCap(value: number): void;

  getGibsonize(): boolean;
  setGibsonize(value: boolean): void;

  getGibsonSpread(): number;
  setGibsonSpread(value: number): void;

  getMinimumPlacement(): number;
  setMinimumPlacement(value: number): void;

  getMaximumByePlacement(): number;
  setMaximumByePlacement(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionControls.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionControls): DivisionControls.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionControls, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionControls;
  static deserializeBinaryFromReader(message: DivisionControls, reader: jspb.BinaryReader): DivisionControls;
}

export namespace DivisionControls {
  export type AsObject = {
    id: string,
    division: string,
    gameRequest?: api_proto_ipc_omgwords_pb.GameRequest.AsObject,
    suspendedResult: TournamentGameResultMap[keyof TournamentGameResultMap],
    suspendedSpread: number,
    autoStart: boolean,
    spreadCap: number,
    gibsonize: boolean,
    gibsonSpread: number,
    minimumPlacement: number,
    maximumByePlacement: number,
  }
}

export class TournamentGame extends jspb.Message {
  clearScoresList(): void;
  getScoresList(): Array<number>;
  setScoresList(value: Array<number>): void;
  addScores(value: number, index?: number): number;

  clearResultsList(): void;
  getResultsList(): Array<TournamentGameResultMap[keyof TournamentGameResultMap]>;
  setResultsList(value: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>): void;
  addResults(value: TournamentGameResultMap[keyof TournamentGameResultMap], index?: number): TournamentGameResultMap[keyof TournamentGameResultMap];

  getGameEndReason(): api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap];
  setGameEndReason(value: api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap]): void;

  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentGame.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentGame): TournamentGame.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentGame, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentGame;
  static deserializeBinaryFromReader(message: TournamentGame, reader: jspb.BinaryReader): TournamentGame;
}

export namespace TournamentGame {
  export type AsObject = {
    scoresList: Array<number>,
    resultsList: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>,
    gameEndReason: api_proto_ipc_omgwords_pb.GameEndReasonMap[keyof api_proto_ipc_omgwords_pb.GameEndReasonMap],
    id: string,
  }
}

export class Pairing extends jspb.Message {
  clearPlayersList(): void;
  getPlayersList(): Array<number>;
  setPlayersList(value: Array<number>): void;
  addPlayers(value: number, index?: number): number;

  getRound(): number;
  setRound(value: number): void;

  clearGamesList(): void;
  getGamesList(): Array<TournamentGame>;
  setGamesList(value: Array<TournamentGame>): void;
  addGames(value?: TournamentGame, index?: number): TournamentGame;

  clearOutcomesList(): void;
  getOutcomesList(): Array<TournamentGameResultMap[keyof TournamentGameResultMap]>;
  setOutcomesList(value: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>): void;
  addOutcomes(value: TournamentGameResultMap[keyof TournamentGameResultMap], index?: number): TournamentGameResultMap[keyof TournamentGameResultMap];

  clearReadyStatesList(): void;
  getReadyStatesList(): Array<string>;
  setReadyStatesList(value: Array<string>): void;
  addReadyStates(value: string, index?: number): string;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): Pairing.AsObject;
  static toObject(includeInstance: boolean, msg: Pairing): Pairing.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: Pairing, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): Pairing;
  static deserializeBinaryFromReader(message: Pairing, reader: jspb.BinaryReader): Pairing;
}

export namespace Pairing {
  export type AsObject = {
    playersList: Array<number>,
    round: number,
    gamesList: Array<TournamentGame.AsObject>,
    outcomesList: Array<TournamentGameResultMap[keyof TournamentGameResultMap]>,
    readyStatesList: Array<string>,
  }
}

export class PlayerStanding extends jspb.Message {
  getPlayerId(): string;
  setPlayerId(value: string): void;

  getWins(): number;
  setWins(value: number): void;

  getLosses(): number;
  setLosses(value: number): void;

  getDraws(): number;
  setDraws(value: number): void;

  getSpread(): number;
  setSpread(value: number): void;

  getGibsonized(): boolean;
  setGibsonized(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PlayerStanding.AsObject;
  static toObject(includeInstance: boolean, msg: PlayerStanding): PlayerStanding.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PlayerStanding, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PlayerStanding;
  static deserializeBinaryFromReader(message: PlayerStanding, reader: jspb.BinaryReader): PlayerStanding;
}

export namespace PlayerStanding {
  export type AsObject = {
    playerId: string,
    wins: number,
    losses: number,
    draws: number,
    spread: number,
    gibsonized: boolean,
  }
}

export class RoundStandings extends jspb.Message {
  clearStandingsList(): void;
  getStandingsList(): Array<PlayerStanding>;
  setStandingsList(value: Array<PlayerStanding>): void;
  addStandings(value?: PlayerStanding, index?: number): PlayerStanding;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): RoundStandings.AsObject;
  static toObject(includeInstance: boolean, msg: RoundStandings): RoundStandings.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: RoundStandings, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): RoundStandings;
  static deserializeBinaryFromReader(message: RoundStandings, reader: jspb.BinaryReader): RoundStandings;
}

export namespace RoundStandings {
  export type AsObject = {
    standingsList: Array<PlayerStanding.AsObject>,
  }
}

export class DivisionPairingsResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionPairingsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionPairingsResponse): DivisionPairingsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionPairingsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionPairingsResponse;
  static deserializeBinaryFromReader(message: DivisionPairingsResponse, reader: jspb.BinaryReader): DivisionPairingsResponse;
}

export namespace DivisionPairingsResponse {
  export type AsObject = {
    id: string,
    division: string,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class DivisionPairingsDeletedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  getRound(): number;
  setRound(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionPairingsDeletedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionPairingsDeletedResponse): DivisionPairingsDeletedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionPairingsDeletedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionPairingsDeletedResponse;
  static deserializeBinaryFromReader(message: DivisionPairingsDeletedResponse, reader: jspb.BinaryReader): DivisionPairingsDeletedResponse;
}

export namespace DivisionPairingsDeletedResponse {
  export type AsObject = {
    id: string,
    division: string,
    round: number,
  }
}

export class PlayersAddedOrRemovedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasPlayers(): boolean;
  clearPlayers(): void;
  getPlayers(): TournamentPersons | undefined;
  setPlayers(value?: TournamentPersons): void;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): PlayersAddedOrRemovedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: PlayersAddedOrRemovedResponse): PlayersAddedOrRemovedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: PlayersAddedOrRemovedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): PlayersAddedOrRemovedResponse;
  static deserializeBinaryFromReader(message: PlayersAddedOrRemovedResponse, reader: jspb.BinaryReader): PlayersAddedOrRemovedResponse;
}

export namespace PlayersAddedOrRemovedResponse {
  export type AsObject = {
    id: string,
    division: string,
    players?: TournamentPersons.AsObject,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class DivisionRoundControls extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  clearRoundControlsList(): void;
  getRoundControlsList(): Array<RoundControl>;
  setRoundControlsList(value: Array<RoundControl>): void;
  addRoundControls(value?: RoundControl, index?: number): RoundControl;

  clearDivisionPairingsList(): void;
  getDivisionPairingsList(): Array<Pairing>;
  setDivisionPairingsList(value: Array<Pairing>): void;
  addDivisionPairings(value?: Pairing, index?: number): Pairing;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionRoundControls.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionRoundControls): DivisionRoundControls.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionRoundControls, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionRoundControls;
  static deserializeBinaryFromReader(message: DivisionRoundControls, reader: jspb.BinaryReader): DivisionRoundControls;
}

export namespace DivisionRoundControls {
  export type AsObject = {
    id: string,
    division: string,
    roundControlsList: Array<RoundControl.AsObject>,
    divisionPairingsList: Array<Pairing.AsObject>,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class DivisionControlsResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasDivisionControls(): boolean;
  clearDivisionControls(): void;
  getDivisionControls(): DivisionControls | undefined;
  setDivisionControls(value?: DivisionControls): void;

  getDivisionStandingsMap(): jspb.Map<number, RoundStandings>;
  clearDivisionStandingsMap(): void;
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DivisionControlsResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DivisionControlsResponse): DivisionControlsResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DivisionControlsResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DivisionControlsResponse;
  static deserializeBinaryFromReader(message: DivisionControlsResponse, reader: jspb.BinaryReader): DivisionControlsResponse;
}

export namespace DivisionControlsResponse {
  export type AsObject = {
    id: string,
    division: string,
    divisionControls?: DivisionControls.AsObject,
    divisionStandingsMap: Array<[number, RoundStandings.AsObject]>,
  }
}

export class TournamentDivisionDataResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  hasPlayers(): boolean;
  clearPlayers(): void;
  getPlayers(): TournamentPersons | undefined;
  setPlayers(value?: TournamentPersons): void;

  getStandingsMap(): jspb.Map<number, RoundStandings>;
  clearStandingsMap(): void;
  getPairingMapMap(): jspb.Map<string, Pairing>;
  clearPairingMapMap(): void;
  hasControls(): boolean;
  clearControls(): void;
  getControls(): DivisionControls | undefined;
  setControls(value?: DivisionControls): void;

  clearRoundControlsList(): void;
  getRoundControlsList(): Array<RoundControl>;
  setRoundControlsList(value: Array<RoundControl>): void;
  addRoundControls(value?: RoundControl, index?: number): RoundControl;

  getCurrentRound(): number;
  setCurrentRound(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDivisionDataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDivisionDataResponse): TournamentDivisionDataResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDivisionDataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDivisionDataResponse;
  static deserializeBinaryFromReader(message: TournamentDivisionDataResponse, reader: jspb.BinaryReader): TournamentDivisionDataResponse;
}

export namespace TournamentDivisionDataResponse {
  export type AsObject = {
    id: string,
    division: string,
    players?: TournamentPersons.AsObject,
    standingsMap: Array<[number, RoundStandings.AsObject]>,
    pairingMapMap: Array<[string, Pairing.AsObject]>,
    controls?: DivisionControls.AsObject,
    roundControlsList: Array<RoundControl.AsObject>,
    currentRound: number,
  }
}

export class FullTournamentDivisions extends jspb.Message {
  getDivisionsMap(): jspb.Map<string, TournamentDivisionDataResponse>;
  clearDivisionsMap(): void;
  getStarted(): boolean;
  setStarted(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): FullTournamentDivisions.AsObject;
  static toObject(includeInstance: boolean, msg: FullTournamentDivisions): FullTournamentDivisions.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: FullTournamentDivisions, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): FullTournamentDivisions;
  static deserializeBinaryFromReader(message: FullTournamentDivisions, reader: jspb.BinaryReader): FullTournamentDivisions;
}

export namespace FullTournamentDivisions {
  export type AsObject = {
    divisionsMap: Array<[string, TournamentDivisionDataResponse.AsObject]>,
    started: boolean,
  }
}

export class TournamentFinishedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentFinishedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentFinishedResponse): TournamentFinishedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentFinishedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentFinishedResponse;
  static deserializeBinaryFromReader(message: TournamentFinishedResponse, reader: jspb.BinaryReader): TournamentFinishedResponse;
}

export namespace TournamentFinishedResponse {
  export type AsObject = {
    id: string,
  }
}

export class TournamentDataResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getExecutiveDirector(): string;
  setExecutiveDirector(value: string): void;

  hasDirectors(): boolean;
  clearDirectors(): void;
  getDirectors(): TournamentPersons | undefined;
  setDirectors(value?: TournamentPersons): void;

  getIsStarted(): boolean;
  setIsStarted(value: boolean): void;

  hasStartTime(): boolean;
  clearStartTime(): void;
  getStartTime(): google_protobuf_timestamp_pb.Timestamp | undefined;
  setStartTime(value?: google_protobuf_timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDataResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDataResponse): TournamentDataResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDataResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDataResponse;
  static deserializeBinaryFromReader(message: TournamentDataResponse, reader: jspb.BinaryReader): TournamentDataResponse;
}

export namespace TournamentDataResponse {
  export type AsObject = {
    id: string,
    name: string,
    description: string,
    executiveDirector: string,
    directors?: TournamentPersons.AsObject,
    isStarted: boolean,
    startTime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
  }
}

export class TournamentDivisionDeletedResponse extends jspb.Message {
  getId(): string;
  setId(value: string): void;

  getDivision(): string;
  setDivision(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TournamentDivisionDeletedResponse.AsObject;
  static toObject(includeInstance: boolean, msg: TournamentDivisionDeletedResponse): TournamentDivisionDeletedResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TournamentDivisionDeletedResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TournamentDivisionDeletedResponse;
  static deserializeBinaryFromReader(message: TournamentDivisionDeletedResponse, reader: jspb.BinaryReader): TournamentDivisionDeletedResponse;
}

export namespace TournamentDivisionDeletedResponse {
  export type AsObject = {
    id: string,
    division: string,
  }
}

export interface TournamentGameResultMap {
  NO_RESULT: 0;
  WIN: 1;
  LOSS: 2;
  DRAW: 3;
  BYE: 4;
  FORFEIT_WIN: 5;
  FORFEIT_LOSS: 6;
  ELIMINATED: 7;
  VOID: 8;
}

export const TournamentGameResult: TournamentGameResultMap;

export interface PairingMethodMap {
  RANDOM: 0;
  ROUND_ROBIN: 1;
  KING_OF_THE_HILL: 2;
  ELIMINATION: 3;
  FACTOR: 4;
  INITIAL_FONTES: 5;
  SWISS: 6;
  QUICKPAIR: 7;
  MANUAL: 8;
  TEAM_ROUND_ROBIN: 9;
}

export const PairingMethod: PairingMethodMap;

export interface FirstMethodMap {
  MANUAL_FIRST: 0;
  RANDOM_FIRST: 1;
  AUTOMATIC_FIRST: 2;
}

export const FirstMethod: FirstMethodMap;

