import React from 'react';
import { Card, Row } from 'antd';
import { FullPlayerInfo } from '../utils/cwgame/game';

type CardProps = {
  player: FullPlayerInfo | null;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }
  return (
    <Card>
      <Row>
        {props.player.nickname} {props.player.onturn ? '*' : ''}
      </Row>
      <Row>
        {props.player.flag} {props.player.title} {props.player.rating}
      </Row>
      <Row>
        Score: {props.player.score} Time: {props.player.timeRemainingSec}
      </Row>
    </Card>
  );
};

type CardsProps = {
  player1: FullPlayerInfo | null; // player1 goes first
  player2: FullPlayerInfo | null;
};

export const PlayerCards = (props: CardsProps) => (
  <>
    <PlayerCard player={props.player1} />
    <PlayerCard player={props.player2} />
  </>
);
