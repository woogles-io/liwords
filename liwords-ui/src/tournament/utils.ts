import { useEffect } from 'react';
import { Action, ActionType } from '../actions/actions';
import axios from 'axios';
import { TournamentMetadata } from '../store/reducers/tournament_reducer';
import { postBinary, toAPIUrl } from '../api/api';
import { GetTournamentRequest } from '../gen/api/proto/tournament_service/tournament_service_pb';
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
      let metadataArg;
      if (tournamentID) {
        metadataArg = {
          id: tournamentID,
        };
      } else if (path) {
        metadataArg = {
          slug: path,
        };
      }

      try {
        const resp = await axios.post<TournamentMetadata>(
          toAPIUrl(
            'tournament_service.TournamentService',
            'GetTournamentMetadata'
          ),
          metadataArg
        );
        dispatchTournamentContext({
          actionType: ActionType.SetTourneyMetadata,
          payload: resp.data,
        });

        if (resp.data.type === 'LEGACY' || resp.data.type === 'CLUB') {
          // This tournament does not have built-in pairings, so no need to fetch
          // tournament divisions.
          return;
        }
        const treq = new GetTournamentRequest();
        treq.setId(resp.data.id);

        const tresp = await postBinary(
          'tournament_service.TournamentService',
          'GetTournament',
          treq
        );

        console.log(
          'after GetTournament, in useTourneyMetadata',
          FullTournamentDivisions.deserializeBinary(tresp.data).toObject()
        );

        dispatchTournamentContext({
          actionType: ActionType.SetDivisionsData,
          payload: {
            fullDivisions: FullTournamentDivisions.deserializeBinary(
              tresp.data
            ),
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
