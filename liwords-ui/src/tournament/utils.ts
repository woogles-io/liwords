import { useEffect } from 'react';
import { Action, ActionType } from '../actions/actions';
import {
  GetTournamentMetadataRequest,
  GetTournamentRequest,
  TType,
} from '../gen/api/proto/tournament_service/tournament_service_pb';
import { LoginState } from '../store/login_state';
import { message } from 'antd';
import { useClient } from '../utils/hooks/connect';
import { TournamentService } from '../gen/api/proto/tournament_service/tournament_service_pb';

export const useTourneyMetadata = (
  path: string,
  tournamentID: string,
  dispatchTournamentContext: (action: Action) => void,
  loginState: LoginState,
  setBadTournament: React.Dispatch<React.SetStateAction<boolean>> | undefined
) => {
  const tournamentClient = useClient(TournamentService);
  useEffect(() => {
    if (!loginState.connectedToSocket) {
      return;
    }

    const getTourneyMetadata = async (
      path: string,
      tournamentID: string,
      dispatchTournamentContext: (action: Action) => void,
      loginState: LoginState,
      setBadTournament:
        | React.Dispatch<React.SetStateAction<boolean>>
        | undefined
    ) => {
      if (!path && !tournamentID) {
        return;
      }
      const tmreq = new GetTournamentMetadataRequest();
      if (tournamentID) {
        tmreq.id = tournamentID;
      } else if (path) {
        tmreq.slug = path;
      }

      try {
        const meta = await tournamentClient.getTournamentMetadata(tmreq);
        console.log('got meta', meta);
        dispatchTournamentContext({
          actionType: ActionType.SetTourneyMetadata,
          payload: {
            directors: meta.directors,
            metadata: meta.metadata,
          },
        });
        const ttype = meta.metadata?.type;
        if (ttype === TType.LEGACY || ttype === TType.CLUB) {
          // This tournament does not have built-in pairings, so no need to fetch
          // tournament divisions.
          return;
        }
        const treq = new GetTournamentRequest();
        if (meta.metadata) {
          treq.id = meta.metadata.id;
        }
        const tresp = await tournamentClient.getTournament(treq);

        dispatchTournamentContext({
          actionType: ActionType.SetDivisionsData,
          payload: {
            fullDivisions: tresp,
            loginState: loginState,
          },
        });
      } catch (err) {
        message.error({
          content: 'Error fetching tournament data',
          duration: 5,
        });
        if (setBadTournament) {
          setBadTournament(true);
        }
      }
    };

    getTourneyMetadata(
      path,
      tournamentID,
      dispatchTournamentContext,
      loginState,
      setBadTournament
    );
  }, [
    dispatchTournamentContext,
    loginState,
    path,
    setBadTournament,
    tournamentID,
    tournamentClient,
  ]);
};
