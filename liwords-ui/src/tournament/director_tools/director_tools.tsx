import React, { useCallback, useMemo } from "react";
// import { toAPIUrl } from '../api/api';
import {
  useTournamentStoreContext,
  useLoginStateStoreContext,
} from "../../store/store";
import "./director_tools.scss";
import { UsernameWithContext } from "../../shared/usernameWithContext";
import { Button, Divider } from "antd";
import { GhettoTools } from "./ghetto_tools";
import { TournamentService } from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../../utils/hooks/connect";
import { CheckCircleOutlined, CameraOutlined } from "@ant-design/icons";
import { useSearchParams } from "react-router";
/*
import { AddPlayerForm, playersToAdd } from './add_player_form';
import { ModifyDivisionsForm } from './modify_divisions_form';
import { Modal } from '../utils/focus_modal';
import { Store } from 'antd/lib/form/interface';
import { SoughtGame } from '../store/reducers/lobby_reducer';
*/

type DTProps = {
  tournamentID: string;
};

export const DirectorTools = React.memo((props: DTProps) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const [searchParams, setSearchParams] = useSearchParams();
  const username = loginState.username;

  // HACK: Check if current user is a read-only director
  // TODO: Replace with proper permissions field when backend schema is updated
  const isReadOnlyDirector = useMemo(() => {
    if (!username) return false;
    return tournamentContext.directors.some(
      (director) => director === `${username}:readonly`,
    );
  }, [username, tournamentContext.directors]);

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
              const [userID, playerName] = p.id.split(":");
              return (
                <li key={p.id} className="player-name">
                  {p.checkedIn && (
                    <span className="checked-in">
                      <CheckCircleOutlined />
                      &nbsp;
                    </span>
                  )}
                  <UsernameWithContext
                    username={playerName}
                    userID={userID}
                    omitSendMessage
                    omitBlock
                  />
                  <small>&nbsp;&nbsp;&nbsp; ({p.rating})</small>
                </li>
              );
            })}
          </ul>
        </div>
      );
    });
  }, [tournamentContext.divisions]);

  const tournamentClient = useClient(TournamentService);

  const renderStartButton = () => {
    const startTournament = async () => {
      try {
        await tournamentClient.startRoundCountdown({
          id: props.tournamentID,
          startAllRounds: true,
        });
      } catch (e) {
        flashError(e);
      }
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
        <Divider />
        <small>Internal tournament ID: {props.tournamentID}</small>
      </>
    );
  };

  const openDirectorDashboard = () => {
    const newParams = new URLSearchParams(searchParams);
    newParams.set("director-dashboard", "true");
    setSearchParams(newParams);
  };

  const renderMonitoringControls = () => {
    if (!tournamentContext.metadata.monitored) {
      return null;
    }

    return (
      <>
        <Divider />
        <h4>Monitoring</h4>
        <Button
          type="primary"
          icon={<CameraOutlined />}
          onClick={openDirectorDashboard}
        >
          Open Director Dashboard
        </Button>
        <p style={{ marginTop: "8px", fontSize: "12px", color: "#666" }}>
          View and manage all participant camera and screen streams
        </p>
      </>
    );
  };

  return (
    <div className="director-tools">
      {!isReadOnlyDirector && renderStartButton()}
      {renderMonitoringControls()}
      {renderRoster()}
      {!isReadOnlyDirector && renderGhettoTools()}
    </div>
  );
});
