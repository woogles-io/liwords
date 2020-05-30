import React from 'react';
import { Card, Row } from 'antd';
import { FullPlayerInfo } from '../utils/cwgame/game';

type CardProps = {
  player: FullPlayerInfo | null;
};

const secsToTimeStr = (s: number): string => {
  const mins = Math.floor(s / 60);
  const secs = Math.floor(s) % 60;
  const minStr = mins.toString().padStart(2, '0');
  const secStr = secs.toString().padStart(2, '0');
  return `${minStr}:${secStr}`;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }
  const timeStr = secsToTimeStr(props.player.timeRemainingSec);
  return (
    <Card>
      <Row>
        {props.player.nickname} {props.player.onturn ? '*' : ''}
      </Row>
      <Row>
        {props.player.flag} {props.player.title} {props.player.rating}
      </Row>
      <Row>
        Score: {props.player.score} Time: {timeStr}
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
