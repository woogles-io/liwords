import { useEffect } from 'react';
import { Action, ActionType } from '../actions/actions';
import { postProto } from '../api/api';
import {
  GetTournamentMetadataRequest,
  GetTournamentRequest,
  TournamentMetadataResponse,
  TType,
} from '../gen/api/proto/tournament_service/tournament_service_pb';
import { FullTournamentDivisions } from '../gen/api/proto/realtime/realtime_pb';
import { LoginState } from '../store/login_state';
import { message } from 'antd';

export const useTourneyMetadata = (
  path: string,
  tournamentID: string,
  dispatchTournamentContext: (action: Action) => void,
  loginState: LoginState,
  setBadTournament: React.Dispatch<React.SetStateAction<boolean>> | undefined
) => {
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
        tmreq.setId(tournamentID);
      } else if (path) {
        tmreq.setSlug(path);
      }

      try {
        const meta = await postProto(
          TournamentMetadataResponse,
          'tournament_service.TournamentService',
          'GetTournamentMetadata',
          tmreq
        );
        console.log('got meta', meta);
        dispatchTournamentContext({
          actionType: ActionType.SetTourneyMetadata,
          payload: {
            directors: meta.getDirectorsList(),
            metadata: meta.getMetadata(),
          },
        });
        const ttype = meta.getMetadata()?.getType();
        if (ttype === TType.LEGACY || ttype === TType.CLUB) {
          // This tournament does not have built-in pairings, so no need to fetch
          // tournament divisions.
          return;
        }
        const treq = new GetTournamentRequest();
        treq.setId(meta.getMetadata()?.getId()!);

        const tresp = await postProto(
          FullTournamentDivisions,
          'tournament_service.TournamentService',
          'GetTournament',
          treq
        );

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
  ]);
};
