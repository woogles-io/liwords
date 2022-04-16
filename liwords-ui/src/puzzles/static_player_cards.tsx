import React from 'react';
import { Button, Card, Row } from 'antd';
import { PlayerAvatar } from '../shared/player_avatar';

type StaticPlayerCardProps = {
  p0Score: number;
  p1Score: number;
  playerOnTurn: number; // 0 based
};

type MiniProps = {
  playerName: string;
  iconName: string;
  onTurn: boolean;
  score: number;
};

export const MiniPlayerCard = React.memo((props: MiniProps) => {
  return (
    <div className={`player-card mini-player${props.onTurn ? ' on-turn' : ''}`}>
      <Row className="player">
        <PlayerAvatar username={props.iconName} player={undefined} />
        <p className="player-name">{props.playerName}</p>
      </Row>
      <Row className="score-timer">
        <Button className="score" type="primary">
          {props.score}
        </Button>
      </Row>
    </div>
  );
});

export const StaticPlayerCards = React.memo((props: StaticPlayerCardProps) => {
  return (
    <Card className="player-cards horizontal">
      <MiniPlayerCard
        iconName="1"
        playerName="Player 1"
        score={props.p0Score}
        onTurn={props.playerOnTurn === 0}
      />
      <MiniPlayerCard
        iconName="2"
        playerName="Player 2"
        score={props.p1Score}
        onTurn={props.playerOnTurn === 1}
      />
    </Card>
  );
});
