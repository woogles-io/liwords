import React, { ReactNode, useMemo } from 'react';
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
  for (let idx = round * numPlayers; idx < (round + 1) * numPlayers; idx++) {
    const key = division.roundInfo[idx];
    if (key && !m.has(key)) {
      n.push(key);
      m.add(key);
    }
  }
  return n.map((key) => division.pairingMap[key]);
};

const getPerformance = (
  playerName: string,
  viewedRound: number,
  division: Division
) => {
  const currentTournamentRound = division.currentRound;
  let roundOfRecord =
    viewedRound > currentTournamentRound ? currentTournamentRound : viewedRound;
  if (roundOfRecord < 0) {
    roundOfRecord = 0;
  }
  const results = division.standingsMap[roundOfRecord].standingsList.find((s) =>
    s.player.endsWith(`:${playerName}`)
  );
  return results
    ? `(${results.wins + results.draws / 2}-${
        results.losses + results.draws / 2
      })`
    : '(0-0)';
};

const getScores = (
  playerName: string,
  viewedRound: number,
  pairing: SinglePairing
) => {
  const playerIndex = pairing.players[0].endsWith(`:${playerName}`) ? 0 : 1;
  const results = pairing.outcomes;
  if (
    pairing.games.length &&
    pairing.games[0].scores.length &&
    results[playerIndex] !== TournamentGameResult.NO_RESULT &&
    results[playerIndex] !== TournamentGameResult.ELIMINATED
  ) {
    return pairing.games[0].scores[playerIndex];
  }
  return '';
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
  wl: ReactNode;
  scores: ReactNode;
  key: string;
  sort: number;
  isMine: boolean;
  actions: ReactNode;
};
export const Pairings = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const history = useHistory();
  const currentRound = useMemo(
    () =>
      props.selectedDivision
        ? divisions[props.selectedDivision].currentRound
        : tournamentContext.competitorState.currentRound,
    [props.selectedDivision, divisions, tournamentContext.competitorState]
  );
  const formatPairingsData = (
    division: Division,
    round: number
  ): PairingTableData[] => {
    if (!division || currentRound === -1) {
      return new Array<PairingTableData>();
    }
    const { status } = tournamentContext.competitorState;
    const pairings = pairingsForRound(props.selectedRound, division);
    const findGameIdFromActive = (playerName: string) => {
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
        const isForfeit =
          pairing.outcomes[0] === TournamentGameResult.FORFEIT_LOSS;
        const isMyGame = props.username && playerNames.includes(props.username);
        // sortPriorty -- The higher the number, the higher up the list,
        // we start by giving your own games a + 2 boost, and other people's byes a -2 deficit.
        // than we add the win lost percentage
        // This results in a list sorted with your game at the top,
        // followed by games in order of combined wl percentage, followed by
        // byes (ranked in order of their participants w/l percentage.
        let sortPriority = isBye || isForfeit ? -2 : 0;
        if (isMyGame) {
          sortPriority = 2;
        }
        const isRemoved = (playerName: string) =>
          division.removedPlayers.includes(playerName);

        const players =
          playerNames[0] === playerNames[1] ? (
            <div>
              <p>
                {playerNames[0]}
                {isBye && <Tag className="ant-tag-bye">Bye</Tag>}
                {isForfeit && <Tag className="ant-tag-forfeit">Forfeit</Tag>}
                {isRemoved(playerNames[0]) && (
                  <Tag className="ant-tag-removed">Removed</Tag>
                )}
              </p>
            </div>
          ) : (
            <div>
              {playerNames.map((playerName) => (
                <p key={playerName}>
                  {playerName}
                  {isRemoved(playerName) && (
                    <Tag className="ant-tag-removed">Removed</Tag>
                  )}
                </p>
              ))}
            </div>
          );
        let actions;
        //Current round gets special buttons
        if (round === currentRound) {
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
                  findGameIdFromActive(props.username!)
                ) {
                  actions = (
                    <Button
                      className="primary"
                      onClick={() => {
                        history.replace(
                          `/game/${encodeURIComponent(
                            findGameIdFromActive(props.username!) || ''
                          )}`
                        );
                        console.log(
                          'redirecting to',
                          findGameIdFromActive(props.username!)
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
            const otherGameId = findGameIdFromActive(playerNames[0]);

            if (otherGameId && !pairing.games[0].gameEndReason) {
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
        if (!actions) {
          const finishedGame = pairing.games.map((game) => game.id).length
            ? pairing.games.map((game) => game.id)[0]
            : null;
          if (finishedGame) {
            actions = (
              <Button
                className="examine"
                onClick={(event) => {
                  if (event.ctrlKey || event.altKey || event.metaKey) {
                    window.open(`/game/${encodeURIComponent(finishedGame)}`);
                  } else {
                    history.replace(
                      `/game/${encodeURIComponent(finishedGame)}`
                    );
                    console.log('redirecting to', finishedGame);
                  }
                }}
                onAuxClick={(event) => {
                  if (event.button === 1) {
                    // middle-click
                    window.open(`/game/${encodeURIComponent(finishedGame)}`);
                  }
                }}
              >
                Examine
              </Button>
            );
          }
        }
        const wl =
          playerNames[0] === playerNames[1] ? (
            <p key={`${playerNames[0]}wl`}>
              {getPerformance(
                playerNames[0],
                round,
                divisions[props.selectedDivision!]
              )}
            </p>
          ) : (
            playerNames.map((playerName) => (
              <p key={`${playerName}wl`}>
                {getPerformance(
                  playerName,
                  round,
                  divisions[props.selectedDivision!]
                )}
              </p>
            ))
          );
        const scores =
          playerNames[0] === playerNames[1]
            ? null
            : playerNames.map((playerName) => (
                <p key={`${playerName}wl`}>
                  {getScores(playerName, round, pairing)}
                </p>
              ));
        return {
          players,
          key: playerNames.join(':'),
          sort: sortPriority || 0,
          isMine: isMyGame || false,
          wl,
          scores,
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
    {
      title: 'W/L',
      dataIndex: 'wl',
      key: 'wl',

      className: 'wl',
    },
  ];

  if (!(props.selectedRound > currentRound)) {
    columns.push({
      title: 'Score',
      dataIndex: 'scores',
      key: 'scores',
      className: 'scores',
    });
  }
  columns.push({
    title: '',
    dataIndex: 'actions',
    key: 'actions',
    className: 'actions',
  });

  if (!props.selectedDivision) {
    return null;
  }

  return (
    <Table
      className={`pairings ${
        currentRound < props.selectedRound
          ? 'future'
          : currentRound > props.selectedRound
          ? 'completed'
          : 'current'
      }`}
      columns={columns}
      pagination={false}
      rowKey={(record) => {
        return `${record.key}`;
      }}
      locale={{
        emptyText: 'The pairings are not yet available for this round.',
      }}
      dataSource={formatPairingsData(
        divisions[props.selectedDivision],
        props.selectedRound
      )}
      rowClassName={(record) => {
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
