import { AutoComplete, Col, Row } from 'antd';
import React, { useCallback } from 'react';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import { debounce } from '../utils/debounce';
import { AddPlayerForm, playersToAdd } from './add_player_form';
import axios from 'axios';

type DTProps = {
  tournamentID: string;
};

/**
 *
 * @param props
message TournamentPerson {
  string person_id = 1;
  int32 person_int = 2;
}

message TournamentPersons {
  string id = 1;
  string division = 2;
  repeated TournamentPerson persons = 3;
}
 */

// Style me later...
export const DirectorTools = React.memo((props: DTProps) => {
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

  // Add players, divisions
  return (
    <div>
      <h3>Entrants List</h3>
      <Row>
        <Col span={12}>
          <AddPlayerForm divisions={['foo', 'bar']} addPlayers={addPlayers} />
        </Col>
      </Row>
    </div>
  );
});
