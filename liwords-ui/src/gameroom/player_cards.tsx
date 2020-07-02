import React from 'react';
import { Card, Row, Button } from 'antd';
import { RawPlayerInfo } from '../store/reducers/game_reducer';
import { useStoreContext } from '../store/store';
import { Millis, millisToTimeStr } from '../store/timer_controller';
import { PlayerAvatar } from '../shared/player_avatar';
import './scss/playerCards.scss';
import { PlayerMetadata } from './game_info';
import { PlayState } from '../gen/macondo/api/proto/macondo/macondo_pb';

type CardProps = {
  player: RawPlayerInfo | undefined;
  time: Millis;
  meta: Array<PlayerMetadata>;
  playing: boolean;
};

const PlayerCard = (props: CardProps) => {
  if (!props.player) {
    return <Card />;
  }

  // Find the metadata for this player.
  const meta = props.meta.find((pi) => pi.user_id === props.player?.userID);
  const timeStr = props.playing ? millisToTimeStr(props.time) : '--:--';
  // TODO: what we consider low time likely be set somewhere and not a magic number
  const timeLow = props.time <= 180000 && props.time > 0;
  const timeOut = props.time <= 0;
  return (
    <div
      className={`player-card${props.player.onturn ? ' on-turn' : ''}
      ${timeLow ? ' time-low' : ''}${timeOut ? ' time-out' : ''}`}
    >
      <Row className="player">
        <PlayerAvatar player={meta} />
        <div className="player-info">
          <p className="player-name">{meta?.full_name || meta?.nickname}</p>
          {meta?.country_code ? (
            <img
              className="player-flag"
              src={meta.country_code}
              // Todo: It would be better if FullPlayerInfo included a displayable country name, for screen readers, etc.
              alt="Country Flag"
            />
          ) : (
            ''
          )}
          <div className="player-details">{meta?.rating || 'Unrated'}</div>
        </div>
      </Row>
      <Row className="score-timer">
        <Button className="score" type="primary">
          {props.player.score}
        </Button>
        <Button className="timer" type="primary">
          {timeStr}
        </Button>
      </Row>
    </div>
  );
};

type Props = {
  playerMeta: Array<PlayerMetadata>;
};

export const PlayerCards = (props: Props) => {
  const { gameContext, timerContext } = useStoreContext();

  // If the gameContext is not yet available, we should try displaying player cards
  // from the meta information, until the information comes in.
  let p0 = gameContext?.players[0];
  let p1 = gameContext?.players[1];
  if (!p0) {
    if (props.playerMeta[0]) {
      p0 = {
        userID: props.playerMeta[0].user_id,
        score: 0,
        onturn: false,
        currentRack: '',
      };
    }
  }

  if (!p1) {
    if (props.playerMeta[1]) {
      p1 = {
        userID: props.playerMeta[1].user_id,
        score: 0,
        onturn: false,
        currentRack: '',
      };
    }
  }

  return (
    <Card className="player-cards">
      <PlayerCard
        player={p0}
        meta={props.playerMeta}
        time={timerContext.p0}
        playing={gameContext.playState !== PlayState.GAME_OVER}
      />
      <PlayerCard
        player={p1}
        meta={props.playerMeta}
        time={timerContext.p1}
        playing={gameContext.playState !== PlayState.GAME_OVER}
      />
    </Card>
  );
};
