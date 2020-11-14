import React from 'react';
import { Link } from 'react-router-dom';
import moment from 'moment';
import { Button, Table, Tag } from 'antd';
import { useResetStoreContext } from '../store/store';
import { RecentGame } from './recent_game';

const colors = require('../base.scss');

type Props = {
  games: Array<RecentGame>;
  fetchPrev?: () => void;
  fetchNext?: () => void;
};

type playerLinkProps = {
  username: string;
  winner: boolean;
  loser: boolean;
};

const PlayerLink = (props: playerLinkProps) => {
  const { resetStore } = useResetStoreContext();

  return (
    <Link
      to={`/profile/${encodeURIComponent(props.username)}`}
      onClick={resetStore}
    >
      {props.username}
      <br />
      {props.winner ? <Tag color={colors.colorPrimary}>Win</Tag> : null}
      {props.loser ? <Tag color={colors.colorBoardTWS}>Loss</Tag> : null}
    </Link>
  );
};

export const RecentTourneyGames = React.memo((props: Props) => {
  const { resetStore } = useResetStoreContext();

  const formattedGames = props.games
    .filter((item) => item.players?.length && item.end_reason !== 'CANCELLED')
    .map((item) => {
      const p1 = (
        <PlayerLink
          username={item.players[0].username}
          winner={item.players[0].result === 'WIN'}
          loser={item.players[0].result === 'LOSS'}
        />
      );
      const p2 = (
        <PlayerLink
          username={item.players[1].username}
          winner={item.players[0].result === 'LOSS'}
          loser={item.players[0].result === 'WIN'}
        />
      );
      const scores = (
        <Link
          to={`/game/${encodeURIComponent(String(item.game_id ?? ''))}`}
          onClick={resetStore}
        >
          {item.players[0].score} - {item.players[1].score}
        </Link>
      );

      const when = moment.unix(item.time ? item.time : 0).fromNow();
      let endReason = '';
      switch (item.end_reason) {
        case 'TIME':
          endReason = 'Time out';
          break;
        case 'CONSECUTIVE_ZEROES':
          endReason = 'Six-zero rule';
          break;
        case 'RESIGNED':
          endReason = 'Resignation';
          break;
        case 'ABANDONED':
          endReason = 'Abandoned';
          break;
        case 'CANCELLED':
          endReason = 'Cancelled';
          break;
        case 'TRIPLE_CHALLENGE':
          endReason = 'Triple Challenge';
          break;
        case 'STANDARD':
          endReason = 'Completed';
      }

      return {
        game_id: item.game_id, // used by rowKey
        p1,
        p2,
        scores,
        endReason,
        when,
      };
    })
    .filter((item) => item !== null);
  const columns = [
    {
      className: 'when',
      dataIndex: 'when',
      key: 'when',
    },
    {
      dataIndex: 'p1',
      key: 'p1',
      title: 'First',
    },
    {
      dataIndex: 'p2',
      key: 'p2',
      title: 'Second',
    },
    {
      className: 'score',
      dataIndex: 'scores',
      key: 'scores',
      title: 'Final Score',
    },
    {
      className: 'end-reason',
      dataIndex: 'endReason',
      key: 'endReason',
      title: 'End',
    },
  ];
  // TODO: use the normal Ant table pagination when the backend can give us a total
  return (
    <>
      <Table
        className="game-history"
        columns={columns}
        dataSource={formattedGames}
        pagination={{
          hideOnSinglePage: true,
        }}
        rowKey="game_id"
      />
      <div className="game-history-controls">
        {props.fetchPrev && <Button onClick={props.fetchPrev}>Prev</Button>}
        {props.fetchNext && <Button onClick={props.fetchNext}>Next</Button>}
      </div>
    </>
  );
});
