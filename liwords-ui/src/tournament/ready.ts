import { create, toBinary } from "@bufbuild/protobuf";
import { ReadyForTournamentGameSchema } from "../gen/api/proto/ipc/tournament_pb";
import { CompetitorState } from "../store/selectors/tournament_selectors";
import { encodeToSocketFmt } from "../utils/protobuf";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";

export const readyForTournamentGame = (
  sendSocketMsg: (msg: Uint8Array) => void,
  tournamentID: string,
  competitorState: CompetitorState,
) => {
  const evt = create(ReadyForTournamentGameSchema, {});
  const division = competitorState.division;
  if (!division) {
    return;
  }
  const round = competitorState.currentRound;
  evt.division = division;
  evt.tournamentId = tournamentID;
  evt.round = round;
  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.READY_FOR_TOURNAMENT_GAME,
      toBinary(ReadyForTournamentGameSchema, evt),
    ),
  );
};
