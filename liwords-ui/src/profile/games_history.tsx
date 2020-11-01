import React from 'react';
import { Link } from 'react-router-dom';
import moment from 'moment';
import { Button, Card, Table, Tag, Tooltip } from 'antd';
import { CheckCircleTwoTone } from '@ant-design/icons';
import { FundOutlined } from '@ant-design/icons/lib';
import { GameMetadata } from '../gameroom/game_info';
import { timeToString } from '../store/constants';
import { useResetStoreContext } from '../store/store';

const colors = require('../base.scss');

type Props = {
  games: Array<GameMetadata>;
  username: string;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  userID: string;
};

export const GamesHistoryCard = React.memo((props: Props) => {
  const { resetStore } = useResetStoreContext();
  const { userID } = props;

  const formattedGames = props.games
    .filter(
      (item) => item.players?.length && item.game_end_reason !== 'CANCELLED'
    )
    .map((item) => {
      const userplace = item.players[0].user_id === userID ? 0 : 1;
      const opponent = (
        <Link
          to={`/profile/${encodeURIComponent(
            item.players[1 - userplace].nickname
          )}`}
          onClick={resetStore}
        >
          {item.players[1 - userplace].nickname}
        </Link>
      );
      const scores = item.scores ? (
        <Link
          to={`/game/${encodeURIComponent(String(item.game_id ?? ''))}`}
          onClick={resetStore}
        >
          {item.scores[userplace]} - {item.scores[1 - userplace]}
        </Link>
      ) : (
        ''
      );
      let result = <Tag color={colors.colorBoardTWS}>Loss</Tag>;
      const challenge = {
        FIVE_POINT: '+5',
        TEN_POINT: '+10',
        SINGLE: 'x1',
        DOUBLE: 'x2',
        TRIPLE: 'x3',
        VOID: 'Void',
      }[item.challenge_rule];
      const getDetails = () => {
        return (
          <>
            <span className={`challenge-rule mode_${challenge}`}>
              {challenge}
            </span>
            {item.rating_mode === 'RATED' ? (
              <Tooltip title="Rated">
                <FundOutlined />
              </Tooltip>
            ) : null}
          </>
        );
      };
      if (item.winner === -1) {
        result = <Tag color="#888">Tie</Tag>;
      } else if (item.winner === userplace) {
        result = <Tag color={colors.colorPrimary}>Win</Tag>;
      }
      let turnOrder = null;
      if (item.players[userplace].first) {
        turnOrder = <CheckCircleTwoTone twoToneColor="#52c41a" />;
      }
      const when = moment(item.created_at ? item.created_at : '').fromNow();
      let endReason = '';
      switch (item.game_end_reason) {
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
      const time = `${item.time_control_name} ${timeToString(
        item.initial_time_seconds,
        item.increment_seconds,
        item.max_overtime_minutes
      )}`;
      return {
        game_id: item.game_id, // used by rowKey
        details: getDetails(),
        result,
        opponent,
        scores,
        turnOrder,
        endReason,
        time,
        when,
      };
    })
    .filter((item) => item !== null);
  const columns = [
    {
      className: 'result',
      dataIndex: 'result',
      key: 'result',
    },
    {
      className: 'when',
      dataIndex: 'when',
      key: 'when',
    },
    {
      className: 'opponent',
      dataIndex: 'opponent',
      key: 'opponent',
      title: 'Opponent',
    },
    {
      className: 'score',
      dataIndex: 'scores',
      key: 'scores',
      title: 'Final Score',
    },
    {
      className: 'turn-order',
      dataIndex: 'turnOrder',
      key: 'turnOrder',
      title: 'First',
    },
    {
      className: 'end-reason',
      dataIndex: 'endReason',
      key: 'endReason',
      title: 'End',
    },
    {
      className: 'time',
      dataIndex: 'time',
      key: 'time',
      title: 'Time Settings',
    },
    {
      title: 'Details',
      className: 'details',
      dataIndex: 'details',
      key: 'details',
    },
  ];
  // TODO: use the normal Ant table pagination when the backend can give us a total
  return (
    <Card title="Game History">
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
    </Card>
  );
});
