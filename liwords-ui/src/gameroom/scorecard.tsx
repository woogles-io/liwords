import React, { useMemo, useRef, useEffect } from 'react';
import { Card } from 'antd';
import {
  GameTurn,
  GameEvent,
} from '../gen/macondo/api/proto/macondo/macondo_pb';
import { Board } from '../utils/cwgame/board';
import { PlayerAvatar } from '../shared/player_avatar';
import { millisToTimeStr } from '../store/timer_controller';
import { tilePlacementEventDisplay } from '../utils/cwgame/game_event';
import { PlayerMetadata } from './game_info';

type Props = {
  playing: boolean;
  username: string;
  turns: Array<GameTurn>;
  currentTurn: GameTurn;
  board: Board;
  playerMeta: Array<PlayerMetadata>;
};

type turnProps = {
  playing: boolean;
  username: string;
  turn: GameTurn;
  board: Board;
};

type MoveEntityObj = {
  player: Partial<PlayerMetadata>;
  coords: string;
  timeRemaining: string;
  rack: string;
  play: string;
  score: string;
  oldScore: number;
  cumulative: number;
};

const displaySummary = (evt: GameEvent, board: Board) => {
  // Handle just a subset of the possible moves here. These may be modified
  // later on.
  switch (evt.getType()) {
    case GameEvent.Type.EXCHANGE:
      return `Exch. ${evt.getExchanged()}`;

    case GameEvent.Type.PASS:
      return 'Passed.';

    case GameEvent.Type.TILE_PLACEMENT_MOVE:
      return tilePlacementEventDisplay(evt, board);

    case GameEvent.Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
      return 'Challenged!';
  }
  return '';
};

const Turn = (props: turnProps) => {
  const evts = props.turn.getEventsList();
  const memoizedTurn: MoveEntityObj = useMemo(() => {
    // Create a base turn, and modify it accordingly. This is memoized as we
    // don't want to do this relatively expensive computation all the time.
    console.log('computing memoized', evts);
    const turn = {
      player: {
        nickname: evts[0].getNickname(),
        // XXX: FIX THIS. avatar url should be set.
        full_name: '',
        avatar_url: '',
      },
      coords: evts[0].getPosition(),
      timeRemaining: millisToTimeStr(evts[0].getMillisRemaining(), false),
      rack: evts[0].getRack(),
      play: displaySummary(evts[0], props.board),
      score: `${evts[0].getScore()}`,
      cumulative: evts[0].getCumulative(),
      oldScore: evts[0].getCumulative() - evts[0].getScore(),
    };
    if (evts.length === 1) {
      return turn;
    }
    // Otherwise, we have to make some modifications.
    if (evts[1].getType() === GameEvent.Type.PHONY_TILES_RETURNED) {
      turn.score = '0';
      turn.cumulative = evts[1].getCumulative();
      turn.play = `(${turn.play})`;
    } else {
      // Otherwise, just add/subtract as needed.
      for (let i = 1; i < evts.length; i++) {
        switch (evts[i].getType()) {
          case GameEvent.Type.CHALLENGE_BONUS:
            turn.score = `${turn.score}+${evts[i].getBonus()}`;
            break;
          case GameEvent.Type.END_RACK_PENALTY:
            turn.score = `${turn.score}-${evts[i].getLostScore()}`;
            break;
          case GameEvent.Type.END_RACK_PTS:
            turn.score = `${turn.score}+${evts[i].getEndRackPoints()}`;
            break;
          case GameEvent.Type.TIME_PENALTY:
            turn.score = `${turn.score}-${evts[i].getLostScore()}`;
            break;
        }
        turn.cumulative = evts[i].getCumulative();
      }
    }
    return turn;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [evts]);

  return (
    <>
      <div className="turn">
        <PlayerAvatar player={memoizedTurn.player} />
        <div className="coords-time">
          <strong>{memoizedTurn.coords}</strong> <br />
          {memoizedTurn.timeRemaining}
        </div>
        <div className="play">
          <strong>{memoizedTurn.play}</strong> <br />
          {memoizedTurn.rack}
        </div>
        <div className="scores">
          {`${memoizedTurn.oldScore}+${memoizedTurn.score}`} <br />
          <strong>{memoizedTurn.cumulative}</strong>
        </div>
      </div>
    </>
  );
};

export const ScoreCard = (props: Props) => {
  const el = useRef<HTMLDivElement>(null);
  useEffect(() => {
    el.current?.scrollIntoView({ block: 'end', behavior: 'smooth' });
  }, [props.turns]);
  return (
    <Card
      className="score-card"
      title={`Turn ${props.turns.length + 1}`}
      // eslint-disable-next-line jsx-a11y/anchor-is-valid
      extra={<a href="#">Notepad</a>}
    >
      {props.turns.map((t, idx) => (
        <Turn
          turn={t}
          board={props.board}
          key={`t_${idx + 0}`}
          playing={props.playing}
          username={props.username}
        />
      ))}
      {props.currentTurn.getEventsList().length ? (
        <Turn
          turn={props.currentTurn}
          board={props.board}
          playing={props.playing}
          username={props.username}
        />
      ) : null}
      <div id="dummy-end" ref={el} />
    </Card>
  );
};
