import React, { ReactNode, useMemo } from 'react';
import { useTournamentStoreContext } from '../store/store';
import { UsernameWithContext } from '../shared/usernameWithContext';
import { Table } from 'antd';
// import { PlayerTag } from './player_tags';

type Props = {
  selectedDivision: string;
};

type StandingsTableData = {
  rank: number;
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
  const currentRound = useMemo(
    () =>
      divisions.hasOwnProperty(props.selectedDivision)
        ? divisions[props.selectedDivision].currentRound
        : 0,
    [props.selectedDivision, divisions]
  );
  if (!props.selectedDivision) {
    return null;
  }
  const division = divisions[props.selectedDivision];
  if (!division) {
    return null;
  }

  let formatStandings;
  if (currentRound > -1) {
    formatStandings = division.standingsMap[currentRound]?.standings.map(
      (standing, index): StandingsTableData => {
        const [playerId, playerName] = standing.playerId.split(':');
        return {
          rank: index + 1,
          player: (
            <>
              <UsernameWithContext
                username={playerName}
                userID={playerId}
                omitSendMessage
                omitBlock
              />{' '}
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
          //actions: null, //scorecard button goes here
        };
      }
    );
  }
  const columns = [
    {
      title: '',
      dataIndex: 'rank',
      key: 'rank',
      className: 'rank',
    },
    {
      title: 'Player',
      dataIndex: 'player',
      key: 'players',
      className: 'player',
    },
    {
      title: 'W',
      dataIndex: 'wins',
      key: 'wins',
      className: 'wins',
    },
    {
      title: 'L',
      dataIndex: 'losses',
      key: 'losses',
      className: 'losses',
    },
    {
      title: 'Spread',
      dataIndex: 'spread',
      key: 'spread',
      className: 'spread',
    },
    /*    {
      title: '',
      dataIndex: 'actions',
      key: 'actions',
      className: 'actions',
    },*/
  ];
  return (
    <Table
      className="standings"
      columns={columns}
      rowKey={(record) => {
        return `${record.rank}`;
      }}
      locale={{
        emptyText: 'Standings are not yet available.',
      }}
      dataSource={formatStandings}
      pagination={false}
    />
  );
};
