import React, { ReactNode } from 'react';
import { useTournamentStoreContext } from '../store/store';
import { Button, Table, Tag } from 'antd';
import {
  Division,
  SinglePairing,
  TourneyStatus,
} from '../store/reducers/tournament_reducer';
import { TournamentGameResult } from '../gen/api/proto/realtime/realtime_pb';
import { useHistory } from 'react-router-dom';

const usernameFromPlayerEntry = (p: string) =>
  p.split(':').length > 0 ? p.split(':')[1] : 'Unknown player';

const pairingsForRound = (
  round: number,
  division: Division
): Array<SinglePairing> => {
  const m = new Set<string>();
  const n = new Array<string>();
  const numPlayers = division.players.length;
  // round in this case is 1-indexed (it is the displayed round).
  for (let idx = (round - 1) * numPlayers; idx < round * numPlayers; idx++) {
    const key = division.roundInfo[idx];
    if (key && !m.has(key)) {
      n.push(key);
      m.add(key);
    }
  }
  return n.map((key) => division.pairingMap[key]);
};

type Props = {
  selectedDivision?: string;
  selectedRound: number;
  username?: string;
  sendReady?: () => void;
};

type PairingTableData = {
  players: ReactNode;
  // ratings: ReactNode;
  //wl: ReactNode;
  //scores: ReactNode;
  sort: number;
  isMine: boolean;
  actions: ReactNode;
};
export const Pairings = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const history = useHistory();
  const formatPairingsData = (
    division: Division,
    round: number
  ): PairingTableData[] => {
    if (!division) {
      return new Array<PairingTableData>();
    }
    const { status } = tournamentContext.competitorState;
    const pairings = pairingsForRound(props.selectedRound, division);

    const findGameId = (playerName: string) => {
      //This assumes one game per round per user
      const game = tournamentContext.activeGames.find((game) => {
        return game.players.map((pm) => pm.displayName).includes(playerName);
      });
      return game?.gameID;
    };
    const pairingsData = pairings.map(
      (pairing: SinglePairing): PairingTableData => {
        const playerNames = pairing.players.map(usernameFromPlayerEntry);
        const isBye = pairing.outcomes[0] === TournamentGameResult.BYE;
        const currentRound = props.selectedDivision
          ? tournamentContext.divisions[props.selectedDivision].currentRound + 1 //zero based here
          : tournamentContext.competitorState.currentRound; // 1 based here

        console.log('tc', tournamentContext, props.selectedRound);
        const isMyGame = props.username && playerNames.includes(props.username);
        // sortPriorty -- The higher the number, the higher up the list,
        // we start by giving your own games a + 2 boost, and other people's byes a -2 deficit.
        // than we add the win lost percentage
        // This results in a list sorted with your game at the top,
        // followed by games in order of combined wl percentage, followed by
        // byes (ranked in order of their participants w/l percentage.
        let sortPriority = isBye ? -2 : 0;
        if (isMyGame) {
          sortPriority = 2;
        }

        const players = isBye ? (
          <div>
            <p>
              {playerNames[0]}
              <Tag className="ant-tag-bye">Bye</Tag>
            </p>
          </div>
        ) : (
          <div>
            {playerNames.map((playerName) => (
              <p key={playerName}>{playerName}</p>
            ))}
          </div>
        );
        let actions;
        //Current round gets special buttons
        if (props.selectedRound === currentRound) {
          if (isMyGame && status) {
            if (
              [
                TourneyStatus.ROUND_OPEN,
                TourneyStatus.ROUND_LATE,
                TourneyStatus.ROUND_OPPONENT_WAITING,
              ].includes(status)
            ) {
              actions = (
                <Button className="primary" onClick={props.sendReady}>
                  I'm ready
                </Button>
              );
            } else {
              if (status === TourneyStatus.ROUND_READY) {
                actions = <p>Waiting for opponent</p>;
              } else {
                if (
                  status === TourneyStatus.ROUND_GAME_ACTIVE &&
                  findGameId(props.username!)
                ) {
                  actions = (
                    <Button
                      className="primary"
                      onClick={() => {
                        history.replace(
                          `/game/${encodeURIComponent(
                            findGameId(props.username!) || ''
                          )}`
                        );
                        console.log(
                          'redirecting to',
                          findGameId(props.username!)
                        );
                      }}
                    >
                      Resume
                    </Button>
                  );
                }
              }
            }
          } else {
            //it's not my game
            const otherGameId = findGameId(playerNames[0]);
            if (otherGameId) {
              actions = (
                <Button
                  className="watch"
                  onClick={(event) => {
                    if (event.ctrlKey || event.altKey || event.metaKey) {
                      window.open(`/game/${encodeURIComponent(otherGameId)}`);
                    } else {
                      history.replace(
                        `/game/${encodeURIComponent(otherGameId)}`
                      );
                      console.log('redirecting to', otherGameId);
                    }
                  }}
                  onAuxClick={(event) => {
                    if (event.button === 1) {
                      // middle-click
                      window.open(`/game/${encodeURIComponent(otherGameId)}`);
                    }
                  }}
                >
                  Watch
                </Button>
              );
            }
          }
        }
        return {
          players,
          sort: sortPriority || 0,
          isMine: isMyGame || false,
          actions: actions || null,
        };
      }
    );
    return pairingsData.sort((a, b) => b.sort - a.sort);
  };

  const columns = [
    {
      title: 'Players',
      dataIndex: 'players',
      key: 'players',
      className: 'players',
    },
    /*    {
      title: 'Ratings',
      dataIndex: 'ratings',
      key: 'ratings',
    },
    {
      title: 'W/L',
      dataIndex: 'wl',
      key: 'wl',
    },
    {
      title: 'Scores',
      dataIndex: 'scores',
      key: 'scores',
    },

    */
    {
      title: '',
      dataIndex: 'actions',
      key: 'actions',
      className: 'actions',
    },
  ];
  if (!props.selectedDivision) {
    return null;
  }
  return (
    <Table
      className="pairings"
      columns={columns}
      pagination={false}
      rowKey="pairing"
      locale={{
        emptyText: 'The pairings are not yet available for this round.',
      }}
      dataSource={formatPairingsData(
        divisions[props.selectedDivision],
        props.selectedRound
      )}
      rowClassName={(record) => {
        const currentRound = props.selectedDivision
          ? tournamentContext.divisions[props.selectedDivision].currentRound + 1 //zero based here
          : tournamentContext.competitorState.currentRound; // 1 based here
        let computedClass = `single-pairing ${tournamentContext.competitorState.status}`;
        if (record.isMine) {
          computedClass += ' mine';
        }
        if (props.selectedRound === currentRound) {
          computedClass += ' current';
        }
        return computedClass;
      }}
    />
  );
};
