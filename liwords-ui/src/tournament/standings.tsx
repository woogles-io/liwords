import React, { ReactNode, useMemo, useState } from "react";
import { useTournamentStoreContext } from "../store/store";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { Table } from "antd";
import { signed } from "./format";
import { PlayerScorecardModal } from "./player_scorecard_modal";
import { competitionRanks, rankLabel } from "./ranks";
// import { PlayerTag } from './player_tags';

type Props = {
  selectedDivision: string;
  selectedRound: number;
};

type StandingsTableData = {
  // The row's position, which the table needs as a key, and its rank as a
  // competition scores it, which ties can share.
  rank: number;
  displayRank: string;
  player: ReactNode;
  //rating: number;
  wins: number;
  losses: number;
  spread: number;
  //actions: ReactNode;
};
export const Standings = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const { divisions } = tournamentContext;
  const [scorecardPlayer, setScorecardPlayer] = useState<string>();
  // An in-real-life division holds names rather than accounts, so its "players"
  // have no profile to open and cannot be befriended.
  const irlMode = tournamentContext.metadata?.irlMode ?? false;
  const currentRound = useMemo(
    () =>
      divisions.hasOwnProperty(props.selectedDivision)
        ? divisions[props.selectedDivision].currentRound
        : 0,
    [props.selectedDivision, divisions],
  );
  if (!props.selectedDivision) {
    return null;
  }
  const division = divisions[props.selectedDivision];
  if (!division) {
    return null;
  }

  let formatStandings;
  // Standings accumulate the results so far, so the server will happily compute
  // them for a round that has not been opened yet -- they come back as a copy
  // of the current round's. Gate on the round rather than on the map, or the
  // tentative next round would silently repeat the current standings under its
  // own heading.
  if (currentRound > -1 && props.selectedRound <= currentRound) {
    const roundStandings =
      division.standingsMap[props.selectedRound]?.standings;
    const ranks = competitionRanks(roundStandings ?? []);
    formatStandings = roundStandings?.map(
      (standing, index): StandingsTableData => {
        const [playerId, playerName] = standing.playerId.split(":");
        return {
          rank: index + 1,
          displayRank: rankLabel(ranks[index], index + 1),
          player: (
            <>
              <UsernameWithContext
                username={playerName}
                userID={playerId}
                omitSendMessage
                omitBlock
                omitProfileLink={irlMode}
                omitFriend={irlMode}
                infoText="View scorecard"
                handleInfoText={() => setScorecardPlayer(standing.playerId)}
              />{" "}
              {/* <PlayerTag
                username={playerName}
                players={division.players}
                tournamentSlug={tournamentContext.metadata.slug}
              /> */}
            </>
          ),
          wins: standing.wins + standing.draws / 2,
          losses: standing.losses + standing.draws / 2,
          spread: standing.spread,
        };
      },
    );
  }
  const columns = [
    {
      title: "",
      dataIndex: "displayRank",
      key: "rank",
      className: "rank",
    },
    {
      title: "Player",
      dataIndex: "player",
      key: "players",
      className: "player",
    },
    {
      title: "W",
      dataIndex: "wins",
      key: "wins",
      className: "wins",
    },
    {
      title: "L",
      dataIndex: "losses",
      key: "losses",
      className: "losses",
    },
    {
      // Signed, so a spread of 50 cannot be read as a bare count. Already
      // right-aligned.
      title: "Spread",
      dataIndex: "spread",
      key: "spread",
      className: "spread",
      render: (spread: number) => signed(spread),
    },
    /*    {
      title: '',
      dataIndex: 'actions',
      key: 'actions',
      className: 'actions',
    },*/
  ];
  return (
    <>
      <Table
        className="standings"
        columns={columns}
        rowKey={(record) => {
          return `${record.rank}`;
        }}
        locale={{
          emptyText: "Standings are not yet available.",
        }}
        dataSource={formatStandings}
        pagination={false}
      />
      {scorecardPlayer && (
        <PlayerScorecardModal
          division={division}
          playerId={scorecardPlayer}
          throughRound={props.selectedRound}
          irlMode={irlMode}
          onClose={() => setScorecardPlayer(undefined)}
          onSelectPlayer={setScorecardPlayer}
        />
      )}
    </>
  );
};
