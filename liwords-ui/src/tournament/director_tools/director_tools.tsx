import React, { useCallback } from 'react';
// import { toAPIUrl } from '../api/api';
// import { useMountedState } from '../utils/mounted';
import { useTournamentStoreContext } from '../../store/store';
import './director_tools.scss';
import { UsernameWithContext } from '../../shared/usernameWithContext';
import { Button, Divider } from 'antd';
import { postJsonObj } from '../../api/api';
import { GhettoTools } from './ghetto_tools';
/*
import { AddPlayerForm, playersToAdd } from './add_player_form';
import axios from 'axios';
import { ModifyDivisionsForm } from './modify_divisions_form';
import { Modal } from '../utils/focus_modal';
import { Store } from 'antd/lib/form/interface';
import { SoughtGame } from '../store/reducers/lobby_reducer';
*/

type DTProps = {
  tournamentID: string;
};

export const DirectorTools = React.memo((props: DTProps) => {
  // const { useState } = useMountedState();

  const { tournamentContext } = useTournamentStoreContext();

  /*   const addPlayers = (p: playersToAdd) => {
    Object.entries(p).forEach(([div, players]) => {
      axios
        .post<{}>(
          toAPIUrl('tournament_service.TournamentService', 'AddPlayers'),
          {
            id: props.tournamentID,
            division: div,
            persons: players.map((p) => ({
              person_id: p.userID,
              person_int: p.rating,
            })),
          }
        )
        .then((res) => {
          console.log('success');
        })
        .catch((err) => {
          window.alert('Error adding players to division ' + div + ': ' + err);
        });
    });
  };

  // Add players, divisions

  const divisionFormSubmit = (g: SoughtGame, v?: Store) => {
      setDivisionModalVisible(false);
      console.log('g is', g, 'v is', v);
    };

   const addDivisionModal = (
      <Modal
        title="Add a Division"
        className="seek-modal"
        visible={divisionModalVisible}
        destroyOnClose
        onCancel={() => {
          setDivisionModalVisible(false);
        }}
        footer={[
          <Button
            key="back"
            onClick={() => {
              setDivisionModalVisible(false);
            }}
          >
            Cancel
          </Button>,
          <button
            className="primary"
            key="submit"
            form="division-settings-form"
            type="submit"
          >
            Add Division
          </button>,
        ]}
      >
        <ModifyDivisionsForm
          tournamentID={props.tournamentID}
          onFormSubmit={divisionFormSubmit}
        />
      </Modal>
    );*/

  const renderRoster = useCallback(() => {
    return Object.values(tournamentContext.divisions).map((d) => {
      return (
        <div key={d.divisionID}>
          <h4 className="division-name">{d.divisionID} entrants</h4>
          <ul>
            {d.players.map((p) => {
              const [userID, playerName] = p.getId().split(':');
              return (
                <li key={p.getId()} className="player-name">
                  <UsernameWithContext
                    username={playerName}
                    userID={userID}
                    omitSendMessage
                    omitBlock
                  />
                  {/* &nbsp;{d.checkedInPlayers.has(p) ? '✓' : ''} */}
                </li>
              );
            })}
          </ul>
        </div>
      );
    });
  }, [tournamentContext.divisions]);

  const renderStartButton = () => {
    const startTournament = async () => {
      postJsonObj(
        'tournament_service.TournamentService',
        'StartRoundCountdown',
        {
          id: props.tournamentID,
          start_all_rounds: true,
        }
      );
    };
    if (
      Object.values(tournamentContext.divisions).length &&
      Object.values(tournamentContext.divisions)[0].currentRound === -1 &&
      !tournamentContext.started
    ) {
      return (
        <>
          <Button className="primary" onClick={startTournament}>
            Start tournament
          </Button>
        </>
      );
    }

    return null;
  };

  const renderGhettoTools = () => {
    return (
      <>
        <Divider />
        <GhettoTools tournamentID={props.tournamentID} />
      </>
    );
  };

  return (
    <div className="director-tools">
      {renderStartButton()}
      {renderRoster()}
      {renderGhettoTools()}
    </div>
  );
});
