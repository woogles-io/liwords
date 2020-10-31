import React, { ReactNode, useEffect, useMemo, useRef } from 'react';
import { useMountedState } from '../utils/mounted';
import { Card } from 'antd';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { Board } from '../utils/cwgame/board';
import { PlayerAvatar } from '../shared/player_avatar';
import { millisToTimeStr } from '../store/timer_controller';
import { Blank } from '../utils/cwgame/common';
import { tilePlacementEventDisplay } from '../utils/cwgame/game_event';
import { PlayerMetadata } from './game_info';
import { Turn, gameEventsToTurns } from '../store/reducers/turns';
import { PoolFormatType } from '../constants/pool_formats';
import { Notepad } from './notepad';
const screenSizes = require('../base.scss');

type Props = {
  playing: boolean;
  username: string;
  events: Array<GameEvent>;
  board: Board;
  poolFormat: PoolFormatType;
  playerMeta: Array<PlayerMetadata>;
};

type turnProps = {
  playerMeta: Array<PlayerMetadata>;
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
  bonus: number;
  endRackPts: number;
  lostScore: number;
};

const sortBlanksLast = (rack: string) => {
  let letters = '';
  let blanks = '';
  for (const tile of rack) {
    if (tile === Blank) {
      blanks += tile;
    } else {
      letters += tile;
    }
  }
  return letters + blanks;
};

const displaySummary = (evt: GameEvent, board: Board) => {
  // Handle just a subset of the possible moves here. These may be modified
  // later on.
  switch (evt.getType()) {
    case GameEvent.Type.EXCHANGE:
      return (
        <span className="exchanged">-{sortBlanksLast(evt.getExchanged())}</span>
      );

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

    let oldScore;
    if (evts[0].getLostScore()) {
      oldScore = evts[0].getCumulative() + evts[0].getLostScore();
    } else if (evts[0].getEndRackPoints()) {
      oldScore = evts[0].getCumulative() - evts[0].getEndRackPoints();
    } else {
      oldScore = evts[0].getCumulative() - evts[0].getScore();
    }
    let timeRemaining = '';
    if (
      evts[0].getType() !== GameEvent.Type.END_RACK_PTS &&
      evts[0].getType() !== GameEvent.Type.END_RACK_PENALTY
    ) {
      timeRemaining = millisToTimeStr(evts[0].getMillisRemaining(), false);
    }
    const turn = {
      player: props.playerMeta.find(
        (playerMeta) => playerMeta.nickname === evts[0].getNickname()
      ) ?? {
        nickname: evts[0].getNickname(),
        // XXX: FIX THIS. avatar url should be set.
        full_name: '',
        avatar_url: '',
      },
      coords: evts[0].getPosition(),
      timeRemaining: timeRemaining,
      rack: evts[0].getRack(),
      play: displaySummary(evts[0], props.board),
      score: `${evts[0].getScore()}`,
      lostScore: evts[0].getLostScore(),
      moveType: displayType(evts[0]),
      cumulative: evts[0].getCumulative(),
      bonus: evts[0].getBonus(),
      endRackPts: evts[0].getEndRackPoints(),
      oldScore: oldScore,
    };
    if (evts.length === 1) {
      turn.rack = sortBlanksLast(turn.rack);
      return turn;
    }
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
    } else {
      if (evts[1].getType() === GameEvent.Type.CHALLENGE_BONUS) {
        turn.cumulative = evts[1].getCumulative();
        turn.play = (
          <>
            <span className="challenge unsuccessful">Challenge!</span>
            <span className="main-word">
              {displaySummary(evts[0], props.board)}
            </span>
          </>
        );
        turn.rack = `Play is valid ${sortBlanksLast(evts[0].getRack())}`;
      }
      // Otherwise, just add/subtract as needed.
      for (let i = 1; i < evts.length; i++) {
        switch (evts[i].getType()) {
          case GameEvent.Type.CHALLENGE_BONUS:
            turn.score = `${turn.score} + ${evts[i].getBonus()}`;
            break;
          case GameEvent.Type.END_RACK_PTS:
            turn.score = `${turn.score}+${evts[i].getEndRackPoints()}`;
            break;
        }
        turn.cumulative = evts[i].getCumulative();
      }
    }
    return turn;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.playerMeta, props.turn]);

  let scoreChange;
  if (memoizedTurn.lostScore > 0) {
    scoreChange = `${memoizedTurn.oldScore} - ${memoizedTurn.lostScore}`;
  } else if (memoizedTurn.endRackPts > 0) {
    scoreChange = `${memoizedTurn.oldScore} + ${memoizedTurn.endRackPts}`;
  } else {
    scoreChange = `${memoizedTurn.oldScore} + ${memoizedTurn.score}`;
  }

  return (
    <>
      <div className="turn">
        <PlayerAvatar player={memoizedTurn.player} withTooltip />
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
          <p className="score-change">{scoreChange}</p>
          <p className="cumulative">{memoizedTurn.cumulative}</p>
        </div>
      </div>
    </>
  );
};

export const ScoreCard = React.memo((props: Props) => {
  const { useState } = useMountedState();

  const el = useRef<HTMLDivElement>(null);
  const [cardHeight, setCardHeight] = useState(0);
  const [notepadVisible, setNotepadVisible] = useState(false);
  const resizeListener = () => {
    const currentEl = el.current;
    const vw = Math.max(
      document.documentElement.clientWidth || 0,
      window.innerWidth || 0
    );
    if (currentEl) {
      currentEl.scrollTop = currentEl.scrollHeight || 0;
      const boardHeight = document.getElementById('board-container')
        ?.clientHeight;
      const poolTop = document.getElementById('pool')?.clientHeight || 0;
      const playerCardTop =
        document.getElementById('player-cards-vertical')?.clientHeight || 0;
      const navHeight = document.getElementById('main-nav')?.clientHeight || 0;
      if (boardHeight && vw > parseInt(screenSizes.screenSizeTablet, 10)) {
        setCardHeight(
          boardHeight -
            currentEl?.getBoundingClientRect().top -
            window.pageYOffset -
            poolTop -
            playerCardTop -
            15 +
            navHeight
        );
      } else {
        setCardHeight(0);
      }
    }
  };
  useEffect(() => {
    if (props.events.length) {
      resizeListener();
    }
  }, [props.events, props.poolFormat]);
  useEffect(() => {
    window.addEventListener('resize', resizeListener);
    return () => {
      window.removeEventListener('resize', resizeListener);
    };
  }, []);

  const turns = gameEventsToTurns(props.events);
  const cardStyle = cardHeight
    ? {
        maxHeight: cardHeight,
        minHeight: cardHeight,
      }
    : undefined;
  const notepadStyle = cardHeight
    ? {
        height: cardHeight - 60,
        display: notepadVisible ? 'block' : 'none',
      }
    : undefined;
  const title = notepadVisible ? 'Notepad' : `Turn ${turns.length + 1}`;
  const extra = notepadVisible ? 'View Scorecard' : 'View Notepad';
  return (
    <Card
      className="score-card"
      title={title}
      // eslint-disable-next-line jsx-a11y/anchor-is-valid
      extra={
        <button
          className="link"
          onClick={() => {
            if (notepadVisible) {
              setNotepadVisible(false);
            } else {
              setNotepadVisible(true);
            }
          }}
        >
          {extra}
        </button>
      }
    >
      <div ref={el} style={cardStyle}>
        <Notepad style={notepadStyle} />
        {!notepadVisible
          ? turns.map((t, idx) =>
              t.length === 0 ? null : (
                <ScorecardTurn
                  turn={t}
                  board={props.board}
                  key={`t_${idx + 0}`}
                  playerMeta={props.playerMeta}
                  playing={props.playing}
                  username={props.username}
                />
              )
            )
          : null}
      </div>
    </Card>
  );
});
