import React from 'react';
import { Link } from 'react-router-dom';
import moment from 'moment';
import { Button, Table, Tag, Tooltip } from 'antd';
import { RecentGame } from './recent_game';

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
  return (
    <Link to={`/profile/${encodeURIComponent(props.username)}`}>
      {props.username}
      <br />
      {props.winner ? <Tag color="red">Win</Tag> : null}
      {props.loser ? <Tag color="blue">Loss</Tag> : null}
      {!props.winner && !props.loser ? <Tag color="gray">Tie</Tag> : null}
    </Link>
  );
};

export const RecentTourneyGames = React.memo((props: Props) => {
  let lastDate = 0;
  const formattedGames = props.games
    .filter((item) => item.players?.length && item.endReason !== 'CANCELLED')
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
        <Link to={`/game/${encodeURIComponent(String(item.gameId ?? ''))}`}>
          {item.players[0].score} - {item.players[1].score}
        </Link>
      );
      const whenMoment = moment.unix(item.time ? Number(item.time) : 0);

      let when: string | JSX.Element = whenMoment.format('HH:mm');
      if (whenMoment.dayOfYear() !== moment.unix(lastDate).dayOfYear()) {
        when = whenMoment.format('MMM Do HH:mm');
      }
      when = <Tooltip title={whenMoment.format('LLL')}>{when}</Tooltip>;

      lastDate = Number(item.time);
      let endReason = '';
      switch (item.endReason) {
        case 'TIME':
          endReason = 'Time';
          break;
        case 'CONSECUTIVE_ZEROES':
          endReason = 'Six 0';
          break;
        case 'RESIGNED':
          endReason = 'Resign';
          break;
        case 'ABORTED':
          endReason = 'Abort';
          break;
        case 'CANCELLED':
          endReason = 'Cancel';
          break;
        case 'TRIPLE_CHALLENGE':
          endReason = 'Triple';
          break;
        case 'STANDARD':
          endReason = 'Complete';
      }

      return {
        gameId: item.gameId, // used by rowKey
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
      dataIndex: 'p1',
      key: 'p1',
      title: '1st',
    },
    {
      dataIndex: 'p2',
      key: 'p2',
      title: '2nd',
    },
    {
      className: 'score',
      dataIndex: 'scores',
      key: 'scores',
      title: 'Score',
    },
    {
      className: 'end-reason',
      dataIndex: 'endReason',
      key: 'endReason',
      title: 'End',
    },
    {
      className: 'when',
      dataIndex: 'when',
      key: 'when',
    },
  ];
  // TODO: use the normal Ant table pagination when the backend can give us a total
  return (
    <>
      <Table
        className="game-history"
        columns={columns}
        dataSource={formattedGames}
        pagination={false}
        rowKey="gameId"
      />
      <div className="game-history-controls">
        {props.fetchPrev && <Button onClick={props.fetchPrev}>Prev</Button>}
        {props.fetchNext && <Button onClick={props.fetchNext}>Next</Button>}
      </div>
    </>
  );
});
