import React, { ReactNode, useEffect, useMemo, useRef } from 'react';
import { Card } from 'antd';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { Board } from '../utils/cwgame/board';
import { PlayerAvatar } from '../shared/player_avatar';
import { millisToTimeStr } from '../store/timer_controller';
import { tilePlacementEventDisplay } from '../utils/cwgame/game_event';
import { PlayerMetadata } from './game_info';
import { Turn, gameEventsToTurns } from '../store/reducers/turns';

type Props = {
  playing: boolean;
  username: string;
  events: Array<GameEvent>;
  board: Board;
  playerMeta: Array<PlayerMetadata>;
};

type turnProps = {
  playing: boolean;
  username: string;
  turn: Turn;
  board: Board;
};

type MoveEntityObj = {
  player: Partial<PlayerMetadata>;
  coords: string;
  timeRemaining: string;
  moveType: string | ReactNode;
  rack: string;
  play: string | ReactNode;
  score: string;
  oldScore: number;
  cumulative: number;
  lostScore: number;
};

const displaySummary = (evt: GameEvent, board: Board) => {
  // Handle just a subset of the possible moves here. These may be modified
  // later on.
  switch (evt.getType()) {
    case GameEvent.Type.EXCHANGE:
      return <span className="exchanged">-{evt.getExchanged()}</span>;

    case GameEvent.Type.PASS:
      return <span className="pass">Passed turn</span>;

    case GameEvent.Type.TILE_PLACEMENT_MOVE:
      return tilePlacementEventDisplay(evt, board);
    case GameEvent.Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
      return <span className="challenge unsuccessful">Challenged</span>;
    case GameEvent.Type.END_RACK_PENALTY:
      return <span className="final-rack">Tiles on rack</span>;
    case GameEvent.Type.TIME_PENALTY:
      return <span className="time-penalty">Time penalty</span>;
  }
  return '';
};

const displayType = (evt: GameEvent) => {
  switch (evt.getType()) {
    case GameEvent.Type.EXCHANGE:
      return <span className="exchanged">EXCH</span>;
    case GameEvent.Type.CHALLENGE:
    case GameEvent.Type.CHALLENGE_BONUS:
    case GameEvent.Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
      return <span className="challenge">&nbsp;</span>;
    default:
      return <span className="other">&nbsp;</span>;
  }
};

const ScorecardTurn = (props: turnProps) => {
  const memoizedTurn: MoveEntityObj = useMemo(() => {
    // Create a base turn, and modify it accordingly. This is memoized as we
    // don't want to do this relatively expensive computation all the time.
    const evts = props.turn;
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
      lostScore: evts[0].getLostScore(),
      moveType: displayType(evts[0]),
      cumulative: evts[0].getCumulative(),
      oldScore: evts[0].getLostScore()
        ? evts[0].getCumulative() + evts[0].getLostScore()
        : evts[0].getCumulative() - evts[0].getScore(),
    };
    if (evts.length === 1) {
      return turn;
    }
    console.log(evts[1].getType());
    // Otherwise, we have to make some modifications.
    if (evts[1].getType() === GameEvent.Type.PHONY_TILES_RETURNED) {
      turn.score = '0';
      turn.cumulative = evts[1].getCumulative();
      turn.play = (
        <>
          <span className="challenge successful">Challenge!</span>
          <span className="main-word">
            {displaySummary(evts[0], props.board)}
          </span>
        </>
      );
      turn.rack = 'Play is invalid';
    } else if (evts[1].getType() === GameEvent.Type.CHALLENGE_BONUS) {
      turn.score = '0';
      turn.cumulative = evts[1].getCumulative();
      turn.play = (
        <>
          <span className="challenge unsuccessful">Challenge!</span>
          <span className="main-word">
            {displaySummary(evts[0], props.board)}
          </span>
        </>
      );
      turn.rack = `Play is valid ${evts[0].getRack()}`;
    } else {
      // Otherwise, just add/subtract as needed.
      for (let i = 1; i < evts.length; i++) {
        switch (evts[i].getType()) {
          case GameEvent.Type.CHALLENGE_BONUS:
            turn.score = `${turn.score}+${evts[i].getBonus()}`;
            break;
          // case GameEvent.Type.END_RACK_PENALTY:
          //   turn.score = `${turn.score}-${evts[i].getLostScore()}`;
          //   break;
          case GameEvent.Type.END_RACK_PTS:
            turn.score = `${turn.score}+${evts[i].getEndRackPoints()}`;
            break;
          // case GameEvent.Type.TIME_PENALTY:
          //   turn.score = `${turn.score}-${evts[i].getLostScore()}`;
          //   break;
        }
        turn.cumulative = evts[i].getCumulative();
      }
    }
    return turn;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.turn]);
  return (
    <>
      <div className="turn">
        <PlayerAvatar player={memoizedTurn.player} />
        <div className="coords-time">
          {memoizedTurn.coords ? (
            <p className="coord">{memoizedTurn.coords}</p>
          ) : (
            <p className="move-type">{memoizedTurn.moveType}</p>
          )}
          <p className="time-left">{memoizedTurn.timeRemaining}</p>
        </div>
        <div className="play">
          <p className="main-word">{memoizedTurn.play}</p>
          <p>{memoizedTurn.rack}</p>
        </div>
        <div className="scores">
          <p className="score-change">
            {memoizedTurn.lostScore > 0
              ? `${memoizedTurn.oldScore} - ${memoizedTurn.lostScore}`
              : `${memoizedTurn.oldScore} + ${memoizedTurn.score}`}
          </p>
          <p className="cumulative">{memoizedTurn.cumulative}</p>
        </div>
      </div>
    </>
  );
};

export const ScoreCard = React.memo((props: Props) => {
  const el = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const currentEl = el.current;

    if (currentEl) {
      currentEl.scrollTop = currentEl.scrollHeight || 0;
    }
  }, [props.events]);

  const turns = gameEventsToTurns(props.events);

  return (
    <Card
      className="score-card"
      title={`Turn ${turns.length + 1}`}
      // eslint-disable-next-line jsx-a11y/anchor-is-valid
      // extra={<a href="#">Show notepad</a>}
    >
      <div ref={el}>
        {turns.map((t, idx) =>
          t.length === 0 ? null : (
            <ScorecardTurn
              turn={t}
              board={props.board}
              key={`t_${idx + 0}`}
              playing={props.playing}
              username={props.username}
            />
          )
        )}
      </div>
    </Card>
  );
});
