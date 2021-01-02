import { Button, Col, Row } from 'antd';
import React from 'react';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import { AddPlayerForm, playersToAdd } from './add_player_form';
import axios from 'axios';
import { ModifyDivisionsForm } from './modify_divisions_form';
import { useTournamentStoreContext } from '../store/store';
import Modal from 'antd/lib/modal/Modal';
import { Store } from 'antd/lib/form/interface';
import { SoughtGame } from '../store/reducers/lobby_reducer';

type DTProps = {
  tournamentID: string;
};

// Style me later...
export const DirectorTools = React.memo((props: DTProps) => {
  const { useState } = useMountedState();

  const [divisionModalVisible, setDivisionModalVisible] = useState(false);

  const { tournamentContext } = useTournamentStoreContext();
  const addPlayers = (p: playersToAdd) => {
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

  const divisions = tournamentContext.metadata.divisions;
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
  );

  return (
    <div>
      <h3>Divisions</h3>
      Current Divisions:{' '}
      <ul>
        {divisions?.map((d) => (
          <li key={d}>d</li>
        ))}
      </ul>
      <Button onClick={() => setDivisionModalVisible(true)}>+ Division</Button>
      <h3>Entrants List</h3>
      <Row>
        <Col span={12}>
          <AddPlayerForm divisions={['foo', 'bar']} addPlayers={addPlayers} />
        </Col>
      </Row>
      {addDivisionModal}
    </div>
  );
});
