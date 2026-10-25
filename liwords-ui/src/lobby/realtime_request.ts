import { GameMode } from "../gen/api/proto/ipc/omgwords_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";

// The real-time seeks and match requests this user has open, next to any
// correspondence ones. The correspondence tab does not list them, so that tab
// points to them instead.
export const findMyRealtimeRequests = (
  soughtGames: SoughtGame[],
  matchRequests: SoughtGame[],
  userID?: string,
  username?: string,
): Array<SoughtGame> => {
  const isMine = (sg: SoughtGame) =>
    (!!userID && sg.seekerID === userID) ||
    (!!username && sg.seeker === username);
  return [...soughtGames, ...matchRequests].filter(
    (sg) => sg.gameMode !== GameMode.CORRESPONDENCE && isMine(sg),
  );
};
