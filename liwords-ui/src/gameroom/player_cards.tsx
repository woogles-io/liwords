import React from 'react';
import { Card, Row } from 'antd';
import { FullPlayerInfo } from '../store/reducers/game_reducer';
import { useStoreContext } from '../store/store';

type CardProps = {
  player: FullPlayerInfo | undefined;
};

const msecsToTimeStr = (s: number): string => {
  const mins = Math.floor(s / 60000);
  const secs = Math.floor(s / 1000) % 60;
  const minStr = mins.toString().padStart(2, '0');
  const secStr = secs.toString().padStart(2, '0');
  return `${minStr}:${secStr}`;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }
  console.log('playercard', props.player.nickname, props.player.onturn);
  // const timeStr = msecsToTimeStr(props.player.timeRemainingMsec);
  const timeStr = '15:00 (fake)';
  return (
    <Card>
      <Row>
        {props.player.nickname} {props.player.onturn ? '*' : ''}
      </Row>
      <Row>
        {props.player.countryFlag} {props.player.title} {props.player.rating}
      </Row>
      <Row>
        Score: {props.player.score} Time: {timeStr}
      </Row>
    </Card>
  );
};

export const PlayerCards = () => {
  const { gameContext } = useStoreContext();

  return (
    <>
      <PlayerCard player={gameContext?.players[0]} />
      <PlayerCard player={gameContext?.players[1]} />
    </>
  );
};
