import { useQuery } from "@connectrpc/connect-query";
import { getGameOwner } from "../../gen/api/proto/omgwords_service/omgwords-GameEventService_connectquery";
import { useLoginStateStoreContext } from "../../store/store";

// True when the logged-in user created this annotated game. Such users go
// back to the editor rather than the lobby when leaving it.
export const useOwnsAnnotatedGame = (
  gameID: string | undefined,
  annotated: boolean | undefined,
) => {
  const { loginState } = useLoginStateStoreContext();
  const { userID } = loginState;
  const { data: gameOwner } = useQuery(
    getGameOwner,
    { gameId: gameID ?? "" },
    { enabled: !!(annotated && gameID && userID) },
  );
  return (
    !!annotated &&
    !!userID &&
    !!gameOwner?.found &&
    gameOwner.creatorId === userID
  );
};
