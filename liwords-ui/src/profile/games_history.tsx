import React from 'react';
import moment from 'moment';
import { GameMetadata } from '../gameroom/game_info';
import { Button, Card, Table, Tag, Tooltip } from 'antd';
import { CheckCircleTwoTone } from '@ant-design/icons';
import { FundOutlined } from '@ant-design/icons/lib';

const colors = require('../base.scss');

type Props = {
  games: Array<GameMetadata>;
  username: string;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  userID: string;
};

export const GamesHistoryCard = React.memo((props: Props) => {
  const { userID } = props;

  const formattedGames = props.games
    .filter((item) => item.players?.length)
    .map((item) => {
      const userplace = item.players[0].user_id === userID ? 0 : 1;
      const opponent = (
        <a href={`/profile/${item.players[1 - userplace].nickname}`}>
          {item.players[1 - userplace].nickname}
        </a>
      );
      const scores = item.scores ? (
        <a href={`/game/${item.game_id}`}>
          {item.scores[userplace]} - {item.scores[1 - userplace]}
        </a>
      ) : (
        ''
      );
      let result = <Tag color={colors.colorBoardTWS}>Loss</Tag>;
      const challenge = {
        FIVE_POINT: '-5',
        TEN_POINT: '-10',
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
            {item.rating_mode ? (
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
        case 'TRIPLE_CHALLENGE':
          endReason = 'Triple Challenge';
          break;
        case 'STANDARD':
          endReason = 'Completed';
      }
      const time = `${item.time_control_name} (${
        item.initial_time_seconds / 60
      }/${item.increment_seconds})`;
      return {
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
