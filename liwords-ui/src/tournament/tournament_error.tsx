import { errorInfo } from "../utils/parse_woogles_error";
import { WooglesError } from "../gen/api/proto/ipc/errors_pb";
import { flashError } from "../utils/hooks/connect";
import { TournamentState } from "../store/reducers/tournament_reducer";
import { WarningOutlined } from "@ant-design/icons";
import { MessageInstance } from "antd/lib/message/interface";
import { NotificationInstance } from "antd/lib/notification/interface";
import { ConnectError } from "@connectrpc/connect";

export const flashTournamentError = (
  message: MessageInstance,
  notification: NotificationInstance,
  e: unknown,
  tc: TournamentState,
  time = 10,
) => {
  if (e instanceof ConnectError) {
    const [errCode, data] = errorInfo(e.rawMessage);
    if (errCode === 0) {
      message.error({
        content: "Unknown tournament error, see console",
        duration: time,
      });
      console.error("Unknown tournament error", e);

      return;
    }

    switch (errCode) {
      case WooglesError.TOURNAMENT_ROUND_NOT_COMPLETE:
        const errDisplay = parseRoundNotCompleteErr(data, tc);
        notification.error({
          message: `Round cannot be opened`,
          description: errDisplay,
          duration: time,
          icon: <WarningOutlined />,
        });
        break;

      default:
        // not handled, flash the old way.
        flashError(e);
    }
  } else {
    flashError(e);
  }
};

const parseRoundNotCompleteErr = (data: string[], tc: TournamentState) => {
  let explanation = "";
  const division = data[1];
  const missingPlayer = parseInt(data[3], 10);
  const round = parseInt(data[2], 10) - 1; // 0-indexed

  if (missingPlayer != undefined) {
    const player =
      tc.divisions[division].players[missingPlayer].id.split(":")[1];
    const pairings = tc.divisions[division].pairings[round].roundPairings;
    if (pairings[missingPlayer].players == undefined) {
      // This player is not paired at all.
      explanation = `${player} does not have a result for round ${
        round + 1
      }. Please assign them a bye or forfeit win/loss with the "Set single pairing" button.`;
    } else {
      explanation = `${player} does not have a result for round ${
        round + 1
      }. Please wait for their game result to be available, or assign a result to their game with the "Set game result" button.`;
    }
    console.log(tc.divisions[division]);
  }

  return (
    <>
      <div>{`Round ${data[2]} is not complete.`}</div>
      <div style={{ marginTop: 20 }}>{explanation}</div>
    </>
  );
};
