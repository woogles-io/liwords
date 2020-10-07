import { Button, List } from 'antd';
import React from 'react';
import moment from 'moment';
import { GameMetadata } from '../gameroom/game_info';

type Props = {
  games: Array<GameMetadata>;
  username: string;
  fetchPrev: () => void;
  fetchNext: () => void;
};

const itemTitle = (item: GameMetadata, username: string) => {
  // XXX: this breaks if you change your username
  const userplace = item.players[0].nickname === username ? 0 : 1;
  const opponent = item.players[1 - userplace].nickname;
  let wlt = '';
  if (item.winner === -1) {
    wlt = 'Tie';
  } else if (item.winner === userplace) {
    wlt = 'Win';
  } else {
    wlt = 'Loss';
  }
  return <a href={`/game/${item.game_id}`}>{`${wlt} vs. ${opponent}`}</a>;
};

const itemDescription = (item: GameMetadata, username: string) => {
  const userplace = item.players[0].nickname === username ? 0 : 1;
  let description = '';
  let scores = '';
  if (item.scores) {
    scores = `${item.scores[userplace]} - ${item.scores[1 - userplace]}`;
  }
  // The users placement in the array aren't necessarily by who went first.
  if (item.players[userplace].first) {
    description = 'Went 1st - ';
  } else {
    description = 'Went 2nd - ';
  }
  switch (item.game_end_reason) {
    case 'TIME':
      description += '(ended on time)';
      break;
    case 'CONSECUTIVE_ZEROES':
      description += '(Six-zero rule)';
      break;
    case 'RESIGNED':
      description += '(ragequit!)';
      break;
    case 'ABANDONED':
      description += '(abandoned game)';
      break;
    case 'TRIPLE_CHALLENGE':
      description += '(3x challenge)';
      break;
    case 'STANDARD':
      description += `Final: ${scores}`;
  }
  description += ` ${item.time_control_name} (${
    item.initial_time_seconds / 60
  }/${item.increment_seconds})`;
  description += ` ${item.rating_mode}`;
  return description;
};

export const GamesHistoryCard = (props: Props) => {
  // Turn the game metadata into a list.
  return (
    <List
      itemLayout="vertical"
      dataSource={props.games}
      header={
        <>
          <Button onClick={props.fetchPrev}>Prev</Button>
          <Button onClick={props.fetchNext}>Next</Button>
        </>
      }
      renderItem={(item) => (
        <List.Item>
          <List.Item.Meta
            title={itemTitle(item, props.username)}
            description={moment(
              item.created_at ? item.created_at : ''
            ).fromNow()}
          />
          <div>{itemDescription(item, props.username)}</div>
        </List.Item>
      )}
    />
  );
};
