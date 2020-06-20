import React from 'react';
import { Card, Row, Button } from 'antd';
import { FullPlayerInfo } from '../store/reducers/game_reducer';
import { useStoreContext } from '../store/store';
import { Millis, millisToTimeStr } from '../store/timer_controller';
import { PlayerAvatar } from '../shared/player_avatar';
import './scss/playerCards.scss';

type CardProps = {
  player: FullPlayerInfo | undefined;
  time: Millis;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }
  const timeStr = millisToTimeStr(props.time);
  // TODO: what we consider low time likely be set somewhere and not a magic number
  const timeLow = props.time <= 180000 && props.time > 0;
  const timeOut = props.time <= 0;
  return (
    <div
      className={`player-card${props.player.onturn ? ' on-turn' : ''}
      ${timeLow ? ' time-low' : ''}${timeOut ? ' time-out' : ''}`}
    >
      <Row className="player">
      <PlayerAvatar player={props.player} />
      <div className="player-info">
        <p className="player-name">
          {props.player.fullName || props.player.nickname}
        </p>
        {props.player.countryFlag ?
          <img
            className="player-flag"
            src={props.player.countryFlag}
            // Todo: It would be better if FullPlayerInfo included a displayable country name, for screen readers, etc.
            alt="Country Flag"
          />
          : ''}
        <div className="player-details">
          {props.player.rating || 'Unrated'}
        </div>
      </div>
      </Row>
      <Row className="score-timer">
        <Button
          className="score"
          type="primary"
        >
          {props.player.score}
        </Button>
        <Button
          className="timer"
          type="primary"
        >
          {timeStr}
        </Button>
      </Row>
    </div>
  );
};

export const PlayerCards = () => {
  const { gameContext, timerContext } = useStoreContext();
  return (
    <Card className="player-cards">
      <PlayerCard player={gameContext?.players[0]} time={timerContext.p0} />
      <PlayerCard player={gameContext?.players[1]} time={timerContext.p1} />
    </Card>
  );
};
