import React from 'react';
import moment from 'moment';
import { Card, Tag, Tooltip } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';
import { Link } from 'react-router-dom';
import { VariantIcon } from '../shared/variant_icons';
import { FundOutlined } from '@ant-design/icons';
import { timeToString } from '../store/constants';
import { GameMetadata } from '../gameroom/game_info';

type GameCardProps = {
  game: GameMetadata;
  userID: string;
};
export const GameCard = React.memo((props: GameCardProps) => {
  const { game, userID } = props;
  const special = ['Unwoogler', 'AnotherUnwoogler', userID];
  const {
    created_at,
    game_id,
    players,
    winner,
    scores,
    game_request,
    game_end_reason,
    time_control_name,
  } = game;
  const whenMoment = moment(created_at || '');
  const when = (
    <Tooltip title={whenMoment.format('LLL')}>{whenMoment.fromNow()}</Tooltip>
  );
  if (!(players?.length > 1)) {
    return null;
  }
  const userplace =
    special.indexOf(players[0].user_id) > special.indexOf(players[1].user_id)
      ? 0
      : 1;
  const opponent = players[1 - userplace];
  const opponentLink = (
    <div className="opponent-link">
      <PlayerAvatar
        player={{
          user_id: opponent.user_id,
          nickname: opponent.nickname,
        }}
      />
      <Link to={`/profile/${encodeURIComponent(opponent.nickname)}`}>
        {players[1 - userplace].nickname}
      </Link>
    </div>
  );

  const challenge = {
    FIVE_POINT: '+5',
    TEN_POINT: '+10',
    SINGLE: 'x1',
    DOUBLE: 'x2',
    TRIPLE: 'x3',
    VOID: 'Void',
  }[game_request.challenge_rule as string];

  let endReason = '';
  switch (game_end_reason) {
    case 'TIME':
      endReason = 'Time out';
      break;
    case 'CONSECUTIVE_ZEROES':
      endReason = 'Six-zero rule';
      break;
    case 'RESIGNED':
      endReason = 'Resignation';
      break;
    case 'FORCE_FORFEIT':
      endReason = 'Forfeit';
      break;
    case 'ABORTED':
      endReason = 'Aborted';
      break;
    case 'CANCELLED':
      endReason = 'Cancelled';
      break;
    case 'TRIPLE_CHALLENGE':
      endReason = 'Triple challenge';
      break;
    case 'STANDARD':
      endReason = 'Completed';
  }

  const getDetails = (
    <div className="detail-icons">
      <VariantIcon vcode="game_request.rules.variant_name" />
      {game_request.rating_mode === 'RATED' ? (
        <Tooltip title="Rated">
          <FundOutlined />
        </Tooltip>
      ) : null}
      <Tooltip title="Challenge Mode">
        <span className={`challenge-rule mode_${challenge}`}>{challenge}</span>
      </Tooltip>
      {players[userplace].first && <Tag className="ant-tag-first">1st</Tag>}
    </div>
  );

  let result = 'Loss';
  if (winner === -1) {
    result = 'Tie';
  } else if (winner === userplace) {
    result = 'Win';
  }
  const actions = [
    <Link
      key="examine-action"
      to={`/game/${encodeURIComponent(String(game_id ?? ''))}`}
    >
      Examine
    </Link>,
  ];
  const scoreDisplay = (
    <>
      <div>
        <h3>
          {(scores?.[userplace] || 0).toString() +
            ' - ' +
            (scores?.[1 - userplace] || 0).toString()}
        </h3>
        <p>{when}</p>
      </div>
      <Tag className={`ant-tag-${result.toLowerCase()}`}>{result}</Tag>
    </>
  );
  const time = `${time_control_name} ${timeToString(
    game_request.initial_time_seconds,
    game_request.increment_seconds,
    game_request.max_overtime_minutes
  )}`;
  return (
    <Card
      className={`game-info ${result.toLowerCase()}`}
      title={scoreDisplay}
      actions={actions}
    >
      {opponentLink}
      <div className="variant-info">
        {game_request.lexicon} - <span className="time-control">{time}</span>
      </div>
      <p>{endReason}</p>
      {getDetails}
    </Card>
  );
});
