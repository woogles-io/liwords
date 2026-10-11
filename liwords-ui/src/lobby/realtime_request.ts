import { GameMode } from "../gen/api/proto/ipc/omgwords_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";

// The real-time seek or match request this user has open, if any (there can
// be one, next to any number of correspondence ones). The correspondence tab
// does not list it, so that tab points to it instead.
export const findMyRealtimeRequest = (
  soughtGames: SoughtGame[],
  matchRequests: SoughtGame[],
  userID?: string,
  username?: string,
): SoughtGame | undefined => {
  const isMine = (sg: SoughtGame) =>
    (!!userID && sg.seekerID === userID) ||
    (!!username && sg.seeker === username);
  return [...soughtGames, ...matchRequests].find(
    (sg) => sg.gameMode !== GameMode.CORRESPONDENCE && isMine(sg),
  );
};
