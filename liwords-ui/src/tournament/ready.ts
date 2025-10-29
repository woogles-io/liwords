import { create, toBinary } from "@bufbuild/protobuf";
import { ReadyForTournamentGameSchema } from "../gen/api/proto/ipc/tournament_pb";
import { CompetitorState } from "../store/selectors/tournament_selectors";
import { encodeToSocketFmt } from "../utils/protobuf";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";
import { message } from "antd";

export const readyForTournamentGame = (
  sendSocketMsg: (msg: Uint8Array) => void,
  tournamentID: string,
  competitorState: CompetitorState,
): boolean => {
  const evt = create(ReadyForTournamentGameSchema, {});
  const division = competitorState.division;
  if (!division) {
    console.error("Cannot ready - no division set", {
      competitorState,
      tournamentID,
    });
    message.error(
      "Cannot mark ready: division not loaded. Please refresh the page.",
    );
    return false;
  }
  const round = competitorState.currentRound;
  evt.division = division;
  evt.tournamentId = tournamentID;
  evt.round = round;

  console.log("Sending ready message", {
    division,
    round,
    tournamentID,
  });

  sendSocketMsg(
    encodeToSocketFmt(
      MessageType.READY_FOR_TOURNAMENT_GAME,
      toBinary(ReadyForTournamentGameSchema, evt),
    ),
  );
  return true;
};
