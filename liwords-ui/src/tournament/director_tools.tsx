import React from 'react';
// import { toAPIUrl } from '../api/api';
// import { useMountedState } from '../utils/mounted';
import { useTournamentStoreContext } from '../store/store';
import './director_tools.scss';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { Button, message, Popconfirm } from 'antd';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { Division } from '../store/reducers/tournament_reducer';
/*
import { AddPlayerForm, playersToAdd } from './add_player_form';
import axios from 'axios';
import { ModifyDivisionsForm } from './modify_divisions_form';
import Modal from 'antd/lib/modal/Modal';
import { Store } from 'antd/lib/form/interface';
import { SoughtGame } from '../store/reducers/lobby_reducer';
*/

type DTProps = {
  tournamentID: string;
};

export const DirectorTools = React.memo((props: DTProps) => {
  // const { useState } = useMountedState();

  const { tournamentContext } = useTournamentStoreContext();

  const divisions = tournamentContext.divisions;

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

  const renderRoster = () => {
    const removePlayer = (playerName: string, divisionId: string) => {
      axios
        .post(
          toAPIUrl('tournament_service.TournamentService', 'StartTournament'),
          {
            id: props.tournamentID,
            division: divisionId,
            persons: [{ [playerName]: 0 }],
          },
          { withCredentials: true }
        )
        .catch((err) => {
          message.error({
            content:
              'Player cannot be removed. Please check with the Woogles team.',
            duration: 8,
          });
          console.log('Error removing player: ' + err.response?.data?.msg);
        });
    };
    return Object.values(divisions).map((d: Division) => (
      <div key={d.divisionID}>
        <h4 className="division-name">{d.divisionID} entrants</h4>
        <ul>
          {d.players.map((p: string) => {
            const [userID, playerName] = p.split(':');
            const additional = (
              <Popconfirm
                title="Are you sure you wish to remove this person? This is not reversible once the tournament has been started."
                onConfirm={() => {
                  removePlayer(playerName, d.divisionID);
                }}
              >
                <li className="link plain">Remove from tournament</li>
              </Popconfirm>
            );
            return (
              <li key={p} className="player-name">
                <UsernameWithContext
                  username={playerName}
                  userID={userID}
                  omitSendMessage
                  omitBlock
                  additionalMenuItems={additional}
                />
              </li>
            );
          })}
        </ul>
      </div>
    ));
  };
  const renderStartButton = () => {
    const startTournament = () => {
      axios
        .post(
          toAPIUrl('tournament_service.TournamentService', 'StartTournament'),
          {
            id: props.tournamentID,
          },
          { withCredentials: true }
        )
        .catch((err) => {
          message.error({
            content:
              'Tournament cannot be started yet. Please check with the Woogles team.',
            duration: 8,
          });
          console.log('Error starting tournament: ' + err.response?.data?.msg);
        });
    };
    if (
      Object.values(divisions).length &&
      Object.values(divisions)[0].currentRound === -1 &&
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

  return (
    <div className="director-tools">
      {renderStartButton()}
      {renderRoster()}
    </div>
  );
});
