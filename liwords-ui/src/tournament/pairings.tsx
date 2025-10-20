import React, { ReactNode, useMemo } from "react";
import { useTournamentStoreContext } from "../store/store";
import { Affix, Alert, Button, List, Table, Tag } from "antd";
import { Division, SinglePairing } from "../store/reducers/tournament_reducer";

import { useNavigate } from "react-router";
import { ReadyButton } from "./ready_button";
import {
  TournamentPerson,
  TournamentGameResult,
  TournamentPersonSchema,
} from "../gen/api/proto/ipc/tournament_pb";
import { create } from "@bufbuild/protobuf";
import { useTournamentCompetitorState } from "../hooks/use_tournament_competitor_state";
import { TourneyStatus } from "../store/selectors/tournament_selectors";
import { ScoreEditPopover } from "./score_edit_popover";
// import { PlayerTag } from './player_tags';

const usernameFromPlayerEntry = (p: string) =>
  p.split(":").length > 0 ? p.split(":")[1] : "Unknown player";

const pairingsForRound = (
  round: number,
  division: Division,
): [Array<SinglePairing>, Set<string>] => {
  const m = new Set<string>();
  const n = new Array<SinglePairing>();
  if (!division || !division.pairings[round]) {
    return [n, new Set<string>()];
  }
  const unpairedPlayers = new Set(division.players.map((tp) => tp.id));

  const key = (persons: TournamentPerson[]): string => {
    let k = persons[0].id + "-" + persons[1].id;
    if (persons[1].id < persons[0].id) {
      k = persons[1].id + "-" + persons[0].id;
    }
    return k;
  };

  division.pairings[round].roundPairings.forEach((value: SinglePairing) => {
    if (value.players) {
      const k = key(value.players);
      if (k && !m.has(k)) {
        n.push(value);
        m.add(k);
        unpairedPlayers.delete(value.players[0].id);
        unpairedPlayers.delete(value.players[1].id);
        // count repeats.
        let pairingCt = 1;
        for (let i = 0; i < round; i++) {
          const dp = division.pairings[i];
          for (let j = 0; j < dp.roundPairings.length; j++) {
            const v = dp.roundPairings[j];
            if (v.players) {
              const kk = key(v.players);
              if (kk === k) {
                pairingCt += 1;
                break;
              }
            }
          }
        }
        value.pairingCount = pairingCt;
      }
    }
  });
  return [n, unpairedPlayers];
};

type record = {
  wins: number;
  draws: number;
  losses: number;
  spread: number;
  sortKey: number;
};

const recordToString = (rec: record) => {
  return `(${rec.wins + rec.draws / 2}-${rec.losses + rec.draws / 2})`;
  // this looks too crowded:
  // ${rec.spread >= 0 ? '+' : ''}${rec.spread}`;
};

const getPerformance = (
  playerName: string,
  viewedRound: number,
  division: Division,
): record => {
  const currentTournamentRound = division.currentRound;
  let roundOfRecord =
    viewedRound > currentTournamentRound ? currentTournamentRound : viewedRound;
  if (roundOfRecord < 0) {
    roundOfRecord = 0;
  }
  const results = division.standingsMap[roundOfRecord]?.standings.find((s) =>
    s.playerId.endsWith(`:${playerName}`),
  );

  const wins = results?.wins || 0;
  const losses = results?.losses || 0;
  const draws = results?.draws || 0;

  const totalGames = wins + losses + draws;
  let weightedPts = 0;
  if (totalGames > 0) {
    // this sort key will always be <= 1 and >= 0
    weightedPts = (wins + draws / 2) / totalGames;
  }

  return {
    wins,
    losses,
    draws,
    spread: results?.spread || 0,
    sortKey: weightedPts,
  };
};

const getScores = (playerName: string, pairing: SinglePairing) => {
  const playerIndex = pairing.players[0].id.endsWith(`:${playerName}`) ? 0 : 1;
  const results = pairing.outcomes;
  if (
    pairing.games.length &&
    pairing.games[0].scores.length &&
    results[playerIndex] !== TournamentGameResult.NO_RESULT &&
    results[playerIndex] !== TournamentGameResult.ELIMINATED
  ) {
    return pairing.games[0].scores[playerIndex];
  }
  return "";
};

type Props = {
  selectedDivision?: string;
  selectedRound: number;
  username?: string;
  sendReady: () => void;
  isDirector: boolean;
  showFirst?: boolean;
  tentative: boolean;
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

export const Pairings = React.memo((props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const competitorState = useTournamentCompetitorState();
  const navigate = useNavigate();
  const currentRound = useMemo(
    () =>
      props.selectedDivision && divisions[props.selectedDivision]
        ? divisions[props.selectedDivision].currentRound
        : competitorState.currentRound,
    [props.selectedDivision, divisions, competitorState],
  );

  if (!props.selectedDivision) {
    return null;
  }

  const [pairings, unpairedPlayers] = pairingsForRound(
    props.selectedRound,
    divisions[props.selectedDivision],
  );

  const formatPairingsData = (
    division: Division,
    round: number,
    pairings: Array<SinglePairing>,
  ): PairingTableData[] => {
    if (!division) {
      return new Array<PairingTableData>();
    }
    // Hide initial pairings from anyone except directors
    if (currentRound === -1 && !props.isDirector) {
      return new Array<PairingTableData>();
    }
    const { status } = competitorState;

    const findGameIdFromActive = (playerName: string) => {
      //This assumes one game per round per user
      const game = tournamentContext.activeGames.find((game) => {
        return game.players.map((pm) => pm.displayName).includes(playerName);
      });
      return game?.gameID;
    };
    const pairingsData = pairings.map(
      (pairing: SinglePairing): PairingTableData => {
        const playerFullIDs = pairing.players.map((v) => v.id);
        const playerNames = playerFullIDs.map(usernameFromPlayerEntry);
        const isBye = pairing.outcomes[0] === TournamentGameResult.BYE;
        const isForfeit =
          pairing.outcomes[0] === TournamentGameResult.FORFEIT_LOSS;
        const isVoid = pairing.outcomes[0] === TournamentGameResult.VOID;
        const isMyGame = props.username && playerNames.includes(props.username);
        // sortPriorty -- The higher the number, the higher up the list,
        // we start by giving your own games a + 2 boost, and other people's byes a -2 deficit.
        // than we add the win lost percentage
        // This results in a list sorted with your game at the top,
        // followed by games in order of combined wl percentage, followed by
        // byes (ranked in order of their participants w/l percentage.
        let sortPriority =
          isBye || isForfeit
            ? -2
            : getPerformance(playerNames[0], round, division).sortKey +
              getPerformance(playerNames[1], round, division).sortKey;
        if (isMyGame) {
          sortPriority = 2;
        }
        const isRemoved = (playerID: string) =>
          division.players[division.playerIndexMap[playerID]]?.suspended;

        const isGibsonized = (playerID: string) => {
          return (
            division.standingsMap[round]?.standings?.find(
              (p) => p.playerId === playerID && p.gibsonized,
            ) !== undefined
          );
        };

        const pairingCt = pairing.pairingCount || 1;
        const repeatCount =
          pairingCt <= 1
            ? ""
            : pairingCt === 2
              ? "Repeat"
              : `${pairingCt}-peat`;

        const players =
          playerNames[0] === playerNames[1] ? (
            <div>
              <p>
                {playerNames[0]}{" "}
                {
                  // <PlayerTag
                  //   username={playerNames[0]}
                  //   players={division.players}
                  //   tournamentSlug={tournamentContext.metadata.slug}
                  // />
                }
                {isBye && <Tag className="ant-tag-bye">Bye</Tag>}
                {isBye && pairingCt > 1 && (
                  <Tag className="ant-tag-repeat">{repeatCount}</Tag>
                )}
                {isForfeit && <Tag className="ant-tag-forfeit">Forfeit</Tag>}
                {isVoid && <Tag className="ant-tag-bye">Not playing</Tag>}
                {isRemoved(playerFullIDs[0]) && (
                  <Tag className="ant-tag-removed">Removed</Tag>
                )}
                {isGibsonized(playerFullIDs[0]) && (
                  <Tag className="ant-tag-gibsonized">Gibsonized</Tag>
                )}
              </p>
            </div>
          ) : (
            <div>
              {playerFullIDs.map((playerID, idx) => (
                <p key={playerID}>
                  {usernameFromPlayerEntry(playerID)}{" "}
                  {
                    // <PlayerTag
                    //   username={playerName}
                    //   players={division.players}
                    //   tournamentSlug={tournamentContext.metadata.slug}
                    // />
                  }
                  {idx === 0 && props.showFirst && <Tag color="green">1st</Tag>}
                  {isRemoved(playerID) && (
                    <Tag className="ant-tag-removed">Removed</Tag>
                  )}
                  {idx === 0 && pairingCt > 1 && (
                    <Tag className="ant-tag-repeat">{repeatCount}</Tag>
                  )}
                  {isGibsonized(playerID) && (
                    <Tag className="ant-tag-gibsonized">Gibsonized</Tag>
                  )}
                  {/* {props.isDirector && pairing.readyStates[idx] !== '' && (
                    <CheckCircleTwoTone />
                    Temporary for https://github.com/woogles-io/liwords/issues/825

                    This requires a back-end change to send the ready state to
                    the directors as well as the involved players.

                    The reducer would have to significantly change as well,
                    as it expects any Ready state messages to be directed
                    to a player currently in the tournament.
                  )} */}
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
              actions = <ReadyButton sendReady={props.sendReady} />;
            } else {
              if (status === TourneyStatus.ROUND_READY) {
                actions = <p>Waiting for opponent</p>;
              } else {
                if (
                  status === TourneyStatus.ROUND_GAME_ACTIVE &&
                  props.username &&
                  findGameIdFromActive(props.username)
                ) {
                  actions = (
                    <Button
                      className="primary"
                      onClick={() => {
                        if (props.username) {
                          navigate(
                            `/game/${encodeURIComponent(
                              findGameIdFromActive(props.username) || "",
                            )}`,
                          );
                        }
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

            if (otherGameId && !pairing.games[0]?.gameEndReason) {
              actions = (
                <Button
                  className="watch"
                  onClick={(event) => {
                    if (event.ctrlKey || event.altKey || event.metaKey) {
                      window.open(`/game/${encodeURIComponent(otherGameId)}`);
                    } else {
                      navigate(`/game/${encodeURIComponent(otherGameId)}`);
                      console.log("redirecting to", otherGameId);
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
                    navigate(`/game/${encodeURIComponent(finishedGame)}`);
                    console.log("redirecting to", finishedGame);
                  }
                }}
                onAuxClick={(event) => {
                  if (event.button === 1) {
                    // middle-click
                    window.open(`/game/${encodeURIComponent(finishedGame)}`);
                  }
                }}
              >
                Analyze
              </Button>
            );
          }
        }
        const wl =
          playerNames[0] === playerNames[1] && props.selectedDivision ? (
            <p key={`${playerNames[0]}wl`}>
              {recordToString(
                getPerformance(
                  playerNames[0],
                  round,
                  divisions[props.selectedDivision],
                ),
              )}
            </p>
          ) : (
            playerNames.map(
              (playerName) =>
                props.selectedDivision && (
                  <p key={`${playerName}wl`}>
                    {recordToString(
                      getPerformance(
                        playerName,
                        round,
                        divisions[props.selectedDivision],
                      ),
                    )}
                  </p>
                ),
            )
          );
        const isSelfPaired = playerNames[0] === playerNames[1];
        let scores: ReactNode = isSelfPaired
          ? null
          : playerNames.map((playerName) => (
              <p key={`${playerName}scores`}>
                {getScores(playerName, pairing)}
              </p>
            ));

        // Wrap scores in popover for directors (for both paired and self-paired players)
        if (props.isDirector && props.selectedDivision) {
          if (isSelfPaired) {
            // For self-paired players, show a clickable edit trigger
            const selfPairedTrigger = (
              <div className="score-edit-trigger" style={{ cursor: "pointer" }}>
                <p>—</p>
              </div>
            );

            scores = (
              <ScoreEditPopover
                tournamentID={divisions[props.selectedDivision].tournamentID}
                division={props.selectedDivision}
                round={round}
                players={pairing.players}
                currentResults={pairing.outcomes}
                isUnpaired={true}
              >
                {selfPairedTrigger}
              </ScoreEditPopover>
            );
          } else if (scores) {
            // For paired players with scores
            const currentScores = pairing.games[0]?.scores || [];
            const scoresContent = (
              <div className="score-edit-trigger" style={{ cursor: "pointer" }}>
                {scores}
              </div>
            );

            scores = (
              <ScoreEditPopover
                tournamentID={divisions[props.selectedDivision].tournamentID}
                division={props.selectedDivision}
                round={round}
                players={pairing.players}
                currentScores={currentScores}
                currentResults={pairing.outcomes}
              >
                {scoresContent}
              </ScoreEditPopover>
            );
          }
        }

        return {
          players,
          key: playerNames.join(":"),
          sort: sortPriority || 0,
          isMine: isMyGame || false,
          wl,
          scores,
          actions: actions || null,
        };
      },
    );
    return pairingsData.sort((a, b) => b.sort - a.sort);
  };

  const columns = [
    {
      title: "Players",
      dataIndex: "players",
      key: "players",
      className: "players",
    },
    {
      title: "W/L",
      dataIndex: "wl",
      key: "wl",

      className: "wl",
    },
  ];

  if (!(props.selectedRound > currentRound)) {
    columns.push({
      title: "Score",
      dataIndex: "scores",
      key: "scores",
      className: "scores",
    });
  }
  columns.push({
    title: " ",
    dataIndex: "actions",
    key: "actions",
    className: "actions",
  });

  const tableData = formatPairingsData(
    divisions[props.selectedDivision],
    props.selectedRound,
    pairings,
  );

  return (
    <>
      {props.tentative && tableData.length > 0 && (
        <Affix offsetTop={10}>
          <Alert
            style={{ marginBottom: 0, marginTop: 10 }}
            type="error"
            message={`Round ${props.selectedRound + 1} is not open yet. Please do not start playing as these pairings may still be tentative.`}
          ></Alert>
        </Affix>
      )}
      {props.tentative && tableData.length > 0 && !props.isDirector && (
        <div
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            width: "100%",
            height: "75%",
            zIndex: 10,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            pointerEvents: "none",
          }}
        >
          <span
            style={{
              fontSize: "5rem",
              fontWeight: "bold",
              color: "#a92e2e",
              letterSpacing: "0.2em",
              opacity: 0.8,
            }}
          >
            TENTATIVE
          </span>
        </div>
      )}
      <Table
        className={`pairings ${
          currentRound < props.selectedRound
            ? "future"
            : currentRound > props.selectedRound
              ? "completed"
              : "current"
        }`}
        columns={columns}
        pagination={false}
        rowKey={(record) => {
          return `${record.key}`;
        }}
        locale={{
          emptyText: "The pairings are not yet available for this round.",
        }}
        dataSource={tableData}
        rowClassName={(record) => {
          let computedClass = `single-pairing ${competitorState.status}`;
          if (record.isMine) {
            computedClass += " mine";
          }
          if (props.selectedRound === currentRound) {
            computedClass += " current";
          }
          return computedClass;
        }}
      />
      {props.tentative && tableData.length > 0 && (
        <Alert
          type="error"
          message={`Round ${props.selectedRound + 1} is not open yet. Please do not start playing as these pairings may still be tentative.`}
        ></Alert>
      )}
      {unpairedPlayers.size && tableData.length ? (
        <>
          <h5 style={{ marginTop: 10 }}>Unpaired players</h5>
          <List
            size="small"
            dataSource={Array.from(unpairedPlayers)}
            renderItem={(playerID) => {
              const username = playerID.split(":")[1];
              const listItem = (
                <List.Item className="readable-text-color">
                  {username}
                </List.Item>
              );

              // Wrap in popover for directors
              if (props.isDirector && props.selectedDivision) {
                const player = create(TournamentPersonSchema, {
                  id: playerID,
                  rating: 0,
                  suspended: false,
                  checkedIn: false,
                });

                return (
                  <ScoreEditPopover
                    tournamentID={
                      divisions[props.selectedDivision].tournamentID
                    }
                    division={props.selectedDivision}
                    round={props.selectedRound}
                    players={[player]}
                    isUnpaired={true}
                    onSuccess={() => {
                      // Refresh will happen automatically via websocket
                    }}
                  >
                    <div
                      className="score-edit-trigger"
                      style={{
                        cursor: "pointer",
                        display: "inline-block",
                        padding: "2px 6px",
                        borderRadius: "4px",
                      }}
                    >
                      {username}
                    </div>
                  </ScoreEditPopover>
                );
              }

              return listItem;
            }}
          />
        </>
      ) : null}
    </>
  );
});
