import React from 'react';
import { Card, Row } from 'antd';
import { FullPlayerInfo } from '../store/reducers/game_reducer';
import { useStoreContext } from '../store/store';
import { Millis, millisToTimeStr } from '../store/timer_controller';

type CardProps = {
  player: FullPlayerInfo | undefined;
  time: Millis;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }
  const timeStr = millisToTimeStr(props.time);
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
  const { gameContext, timerContext } = useStoreContext();
  return (
    <>
      <PlayerCard player={gameContext?.players[0]} time={timerContext.p0} />
      <PlayerCard player={gameContext?.players[1]} time={timerContext.p1} />
    </>
  );
};
