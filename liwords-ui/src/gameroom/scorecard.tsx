import React, {
  ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Card, Dropdown, Tooltip } from "antd";
import { Link } from "react-router";
import {
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Board } from "../utils/cwgame/board";
import { PlayerAvatar } from "../shared/player_avatar";
import {
  millisToTimeStr,
  millisToTimeStrWithoutDays,
} from "../store/timer_controller";
import {
  nicknameFromEvt,
  tilePlacementEventDisplay,
} from "../utils/cwgame/game_event";
import { Turn, gameEventsToTurns } from "../store/reducers/turns";
import { PoolFormatType } from "../constants/pool_formats";
import { Notepad, useNotepadHasContent } from "./notepad";
import { sortTiles } from "../store/constants";
import { getVW, isTablet } from "../utils/cwgame/common";
import { Analyzer } from "./analyzer";
import { HeartFilled, HourglassOutlined } from "@ant-design/icons";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import {
  useExamineStoreContext,
  useGameContextStoreContext,
} from "../store/store";
import { TurnCommentPreview } from "./TurnCommentPreview";
import {
  Alphabet,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import { calculateHistoricalTimeBanks } from "../utils/time_bank_calculator";
import variables from "../base.module.scss";
const { screenSizeDesktop, screenSizeLaptop, screenSizeTablet } = variables;

type Props = {
  isExamining?: boolean;
  events: Array<GameEvent>;
  allEvents?: Array<GameEvent>; // full game events, used for two-col view in examination mode
  allBoard?: Board; // full final board, used for two-col view in examination mode
  board: Board;
  poolFormat: PoolFormatType;
  playerMeta: Array<PlayerInfo>;
  gameEpilog?: React.ReactElement<Element>;
  hideExtraInteractions?: boolean;
  showComments?: boolean;
  onOpenCommentsDrawer?: (turnIndex: number) => void;
  comments?: Array<GameComment>;
  editComment?: (cid: string, comment: string) => Promise<void>;
  addNewComment?: (
    gameID: string,
    eventNumber: number,
    comment: string,
  ) => Promise<void>;
  deleteComment?: (cid: string) => Promise<void>;
  isCorrespondence?: boolean;
  timeBankP0?: number; // Time bank in milliseconds for player 0
  timeBankP1?: number; // Time bank in milliseconds for player 1
};

type turnProps = {
  playerMeta: Array<PlayerInfo>;
  turn: Turn;
  board: Board;
  showComments: boolean;
  comments: Array<GameComment>;
  alphabet: Alphabet;
  turnIndex: number;
  onOpenCommentsDrawer?: (turnIndex: number) => void;
  isCorrespondence?: boolean;
  timeBankP0?: number;
  timeBankP1?: number;
  eventIndex: number; // Index in the full events array (not turn index)
};

type MoveEntityObj = {
  player: Partial<PlayerInfo>;
  coords: string;
  timeRemaining: ReactNode;
  moveType: string | ReactNode;
  rack: string;
  play: string | ReactNode;
  score: string;
  oldScore: number;
  cumulative: number;
  bonus: number;
  endRackPts: number;
  lostScore: number;
  isBingo: boolean;
  isUsingTimeBank: boolean;
};

function sortStringRack(rack: string, alphabet: Alphabet): string {
  // convert to ML for sorting
  const ml = runesToMachineWord(rack, alphabet);
  const sorted = sortTiles(ml, alphabet);
  return machineWordToRunes(sorted, alphabet, false, true);
}

const displaySummary = (evt: GameEvent, board: Board, alphabet: Alphabet) => {
  // Handle just a subset of the possible moves here. These may be modified
  // later on.
  switch (evt.type) {
    case GameEvent_Type.EXCHANGE:
      // evt.exchanged gets modified by the backend to either be a string
      // with the exchanged letters, or the number of exchanged tiles
      // (for the purposes of maintaining secrecy if you're currently in a game)
      // We must deal with these two cases. Note that this assumes that
      // tiles cannot be numbers. This is OK for now. We will have to redo
      // this behavior anyway once we move to OMGWordsEvents.
      let exchStr = "";
      if (evt.exchanged === "") {
        exchStr = `${evt.numTilesFromRack}`;
      } else {
        exchStr = sortStringRack(evt.exchanged, alphabet);
      }

      return <span className="exchanged">-{exchStr}</span>;

    case GameEvent_Type.PASS:
      return <span className="pass">Passed turn</span>;

    case GameEvent_Type.TILE_PLACEMENT_MOVE:
      return tilePlacementEventDisplay(evt, board, alphabet);
    case GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
      return <span className="challenge unsuccessful">Challenged</span>;
    case GameEvent_Type.END_RACK_PENALTY:
      return <span className="final-rack">Tiles on rack</span>;
    case GameEvent_Type.TIME_PENALTY:
      return <span className="time-penalty">Time penalty</span>;
  }
  return "";
};

const displayType = (evt: GameEvent) => {
  switch (evt.type) {
    case GameEvent_Type.EXCHANGE:
      return <span className="exchanged">EXCH</span>;
    case GameEvent_Type.CHALLENGE:
    case GameEvent_Type.CHALLENGE_BONUS:
    case GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
      return <span className="challenge">&nbsp;</span>;
    default:
      return <span className="other">&nbsp;</span>;
  }
};

// same as player_cards.tsx
const makeTimeRemainingFragment = (
  timeRemainingMillis: number,
  showTenths = true,
) => {
  const timeRemainingWithDays = millisToTimeStr(
    timeRemainingMillis,
    showTenths,
  );
  return (
    <Tooltip
      title={
        timeRemainingWithDays.includes("day")
          ? millisToTimeStrWithoutDays(timeRemainingMillis, showTenths)
          : null
      }
    >
      {timeRemainingWithDays}
    </Tooltip>
  );
};

const ScorecardTurn = (props: turnProps) => {
  const memoizedTurn: MoveEntityObj = useMemo(() => {
    // Create a base turn, and modify it accordingly. This is memoized as we
    // don't want to do this relatively expensive computation all the time.
    const evts = props.turn.events;

    let oldScore;
    if (evts[0].lostScore) {
      oldScore = evts[0].cumulative + evts[0].lostScore;
    } else if (evts[0].endRackPoints) {
      oldScore = evts[0].cumulative - evts[0].endRackPoints;
    } else {
      oldScore = evts[0].cumulative - evts[0].score;
    }
    let timeRemainingMillis = 0;
    let isUsingTimeBank = false;
    // Show time for all moves except end rack events
    if (
      evts[0].type !== GameEvent_Type.END_RACK_PTS &&
      evts[0].type !== GameEvent_Type.END_RACK_PENALTY
    ) {
      const millisRemaining = evts[0].millisRemaining;

      // If time is negative, player was using time bank
      // The time bank value will be passed from the parent component
      if (millisRemaining < 0) {
        const playerIndex = evts[0].playerIndex;
        const timeBankValue =
          playerIndex === 0 ? props.timeBankP0 : props.timeBankP1;

        if (timeBankValue && timeBankValue > 0) {
          timeRemainingMillis = timeBankValue;
          isUsingTimeBank = true;
        } else {
          // For real-time games with overtime, display the negative time
          timeRemainingMillis = millisRemaining;
        }
      } else {
        timeRemainingMillis = millisRemaining;
      }
    }

    const timeRemaining = makeTimeRemainingFragment(timeRemainingMillis, false);

    const turnNickname = nicknameFromEvt(evts[0], props.playerMeta);
    const turn = {
      player: props.playerMeta.find(
        (playerMeta) => playerMeta.nickname === turnNickname,
      ) ?? {
        nickname: turnNickname,
        // XXX: FIX THIS. avatar url should be set.
        fullName: "",
        avatarUrl: "",
      },
      coords: evts[0].position,
      timeRemaining: timeRemaining,
      rack: evts[0].rack,
      play: displaySummary(evts[0], props.board, props.alphabet),
      score: `${evts[0].score}`,
      lostScore: evts[0].lostScore,
      moveType: displayType(evts[0]),
      cumulative: evts[0].cumulative,
      bonus: evts[0].bonus,
      endRackPts: evts[0].endRackPoints,
      oldScore: oldScore,
      isBingo: evts[0].isBingo,
      isUsingTimeBank: isUsingTimeBank,
    };
    if (evts.length === 1) {
      turn.rack = sortStringRack(turn.rack, props.alphabet);
      return turn;
    }
    // Otherwise, we have to make some modifications.
    if (evts[1].type === GameEvent_Type.PHONY_TILES_RETURNED) {
      turn.score = "0";
      turn.cumulative = evts[1].cumulative;
      turn.play = (
        <>
          <span className="challenge successful">Challenge!</span>
          <span className="main-word">
            {displaySummary(evts[0], props.board, props.alphabet)}
          </span>
        </>
      );
      turn.rack = "Play is invalid";
    } else {
      if (evts[1].type === GameEvent_Type.CHALLENGE_BONUS) {
        turn.cumulative = evts[1].cumulative;
        turn.play = (
          <>
            <span className="challenge unsuccessful">Challenge!</span>
            <span className="main-word">
              {displaySummary(evts[0], props.board, props.alphabet)}
            </span>
          </>
        );
        turn.rack = `Play is valid ${sortStringRack(
          evts[0].rack,
          props.alphabet,
        )}`;
      } else {
        // Void challenge combines the end rack points.
        turn.rack = sortStringRack(turn.rack, props.alphabet);
      }
      // Otherwise, just add/subtract as needed.
      for (let i = 1; i < evts.length; i++) {
        switch (evts[i].type) {
          case GameEvent_Type.CHALLENGE_BONUS:
            turn.score = `${turn.score} +${evts[i].bonus}`;
            break;
          case GameEvent_Type.END_RACK_PTS:
            turn.score = `${turn.score} +${evts[i].endRackPoints}`;
            break;
        }
        turn.cumulative = evts[i].cumulative;
      }
    }
    return turn;
  }, [
    props.board,
    props.playerMeta,
    props.turn,
    props.alphabet,
    props.timeBankP0,
    props.timeBankP1,
  ]);

  let scoreChange;
  if (memoizedTurn.lostScore > 0) {
    scoreChange = `${memoizedTurn.oldScore} -${memoizedTurn.lostScore}`;
  } else if (memoizedTurn.endRackPts > 0) {
    scoreChange = `${memoizedTurn.oldScore} +${memoizedTurn.endRackPts}`;
  } else {
    scoreChange = `${memoizedTurn.oldScore} +${memoizedTurn.score}`;
  }

  const divProps: {
    className: string;
  } = {
    className: `turn${memoizedTurn.isBingo ? " bingo" : ""}`,
  };

  return (
    <>
      <div {...divProps}>
        <PlayerAvatar player={memoizedTurn.player} withTooltip />
        <div
          className={`coords-time${props.isCorrespondence ? " correspondence" : ""}`}
        >
          {memoizedTurn.coords ? (
            <p className="coord">{memoizedTurn.coords}</p>
          ) : (
            <p className="move-type">{memoizedTurn.moveType}</p>
          )}
          <p className="time-left">
            {memoizedTurn.timeRemaining}
            {memoizedTurn.isUsingTimeBank && (
              <HourglassOutlined className="time-bank-icon" />
            )}
          </p>
        </div>
        <div className="play">
          <p className="main-word">
            {memoizedTurn.play}
            {memoizedTurn.isBingo && <HeartFilled />}
          </p>
          <p>{memoizedTurn.rack}</p>
        </div>
        <div className="scores">
          <p className="score-change">{scoreChange}</p>
          <p className="cumulative">{memoizedTurn.cumulative}</p>
        </div>
        {props.showComments && (
          <TurnCommentPreview
            comments={props.comments}
            onExpandComments={() => {
              if (props.onOpenCommentsDrawer) {
                props.onOpenCommentsDrawer(props.turnIndex);
              }
            }}
            className={`inline-comment-bubble ${
              props.comments.length > 0 ? "invisible-placeholder" : ""
            }`}
          />
        )}
      </div>
      {props.showComments && props.comments.length > 0 && (
        <div className="turn-comment-row">
          <TurnCommentPreview
            comments={props.comments}
            onExpandComments={() => {
              if (props.onOpenCommentsDrawer) {
                props.onOpenCommentsDrawer(props.turnIndex);
              }
            }}
          />
          <div className="comment-preview-text">
            "
            {props.comments[props.comments.length - 1].comment.length > 120
              ? `${props.comments[props.comments.length - 1].comment.substring(0, 120)}...`
              : props.comments[props.comments.length - 1].comment}
            "
          </div>
        </div>
      )}
    </>
  );
};

type ScorecardViewType = "single" | "two-col";
const SCORECARD_VIEW_KEY = "scorecardView";

type TwoColTurnProps = {
  turn: Turn;
  board: Board;
  alphabet: Alphabet;
  onSeek?: () => void;
  isSelected?: boolean;
  showComments?: boolean;
  comments?: Array<GameComment>;
  onOpenCommentsDrawer?: (turnIndex: number) => void;
  turnIndex?: number;
};

const TwoColTurn = React.memo((props: TwoColTurnProps) => {
  const { turn, board, alphabet, onSeek, isSelected } = props;
  const evt = turn.events[0];

  let coords = evt.position || "";
  let play: React.ReactNode = displaySummary(evt, board, alphabet);
  let score = `${evt.score}`;
  let cumulative = evt.cumulative;

  // Handle multi-event turns (challenges, etc.)
  if (turn.events.length > 1) {
    const evt1 = turn.events[1];
    if (evt1.type === GameEvent_Type.PHONY_TILES_RETURNED) {
      score = "0";
      cumulative = evt1.cumulative;
      play = <span className="challenge successful">Challenge!</span>;
    } else {
      for (let i = 1; i < turn.events.length; i++) {
        switch (turn.events[i].type) {
          case GameEvent_Type.CHALLENGE_BONUS:
            score = `${score}+${turn.events[i].bonus}`;
            break;
          case GameEvent_Type.END_RACK_PTS:
            score = `${score}+${turn.events[i].endRackPoints}`;
            break;
        }
        cumulative = turn.events[i].cumulative;
      }
    }
  }

  const rack = sortStringRack(
    turn.events.length > 1 &&
      turn.events[1].type === GameEvent_Type.PHONY_TILES_RETURNED
      ? ""
      : evt.rack,
    alphabet,
  );

  return (
    <div
      className={`two-col-turn${isSelected ? " two-col-turn-selected" : ""}`}
    >
      <div className="two-col-coord">{coords}</div>
      <div
        className={`two-col-word${onSeek ? " two-col-word-seekable" : ""}`}
        onClick={onSeek}
      >
        {play}
      </div>
      <div className="two-col-scores">
        <div className="two-col-score">{score}</div>
        <div className="two-col-cumulative">{cumulative}</div>
      </div>
      <div className="two-col-comment-spot">
        {props.showComments && (
          <TurnCommentPreview
            comments={props.comments ?? []}
            onExpandComments={() =>
              props.onOpenCommentsDrawer?.(props.turnIndex ?? 0)
            }
            className="two-col-bubble"
          />
        )}
      </div>
      <div className="two-col-rack">{rack}</div>
    </div>
  );
});

export const ScoreCard = React.memo((props: Props) => {
  const el = useRef<HTMLDivElement>(null);
  const [cardHeight, setCardHeight] = useState(0);
  const [flipHidden, setFlipHidden] = useState(true);
  const [flipEnabled, setEnableFlip] = useState(isTablet());
  const [scorecardView, setScorecardView] = useState<ScorecardViewType>(
    () =>
      (localStorage?.getItem(SCORECARD_VIEW_KEY) as ScorecardViewType) ||
      "single",
  );
  const notepadHasContent = useNotepadHasContent();
  // Autoscroll code removed - comments now use drawer

  const turns = useMemo(() => gameEventsToTurns(props.events), [props.events]);
  const allTurns = useMemo(
    () => (props.allEvents ? gameEventsToTurns(props.allEvents) : turns),
    [props.allEvents, turns],
  );

  // Calculate time bank history for all events
  const timeBankHistory = useMemo(() => {
    if (!props.timeBankP0 && !props.timeBankP1) {
      return null;
    }
    return calculateHistoricalTimeBanks(
      props.events,
      props.timeBankP0 || 0,
      props.timeBankP1 || 0,
    );
  }, [props.events, props.timeBankP0, props.timeBankP1]);

  // Scroll to bottom when turns change (after DOM updates)
  useEffect(() => {
    const currentEl = el.current;
    if (currentEl && flipHidden && scorecardView !== "two-col") {
      // Use requestAnimationFrame to ensure DOM has updated
      requestAnimationFrame(() => {
        currentEl.scrollTop = currentEl.scrollHeight;
      });
    }
  }, [turns, flipHidden, scorecardView]);

  const toggleFlipVisibility = useCallback(() => {
    setFlipHidden((x) => !x);
  }, []);
  const resizeListener = useCallback(() => {
    const currentEl = el.current;
    if (isTablet() && !props.hideExtraInteractions) {
      setEnableFlip(true);
    } else {
      setEnableFlip(false);
      setFlipHidden(true);
    }
    if (currentEl) {
      if (scorecardView !== "two-col") {
        currentEl.scrollTop = currentEl.scrollHeight || 0;
      }
      const boardHeight =
        document.getElementById("board-container")?.clientHeight;
      const poolTop = document.getElementById("pool")?.clientHeight || 0;
      const playerCardTop =
        document.getElementById("player-cards-vertical")?.clientHeight || 0;
      const navHeight = document.getElementById("main-nav")?.clientHeight || 0;
      let offset = 0;
      if (getVW() > parseInt(screenSizeLaptop)) {
        offset = 45;
      }
      if (getVW() > parseInt(screenSizeDesktop)) {
        offset = 25;
      }
      if (boardHeight && getVW() >= parseInt(screenSizeTablet, 10)) {
        setCardHeight(
          boardHeight +
            offset -
            currentEl?.getBoundingClientRect().top -
            window.pageYOffset -
            poolTop -
            playerCardTop -
            15 +
            navHeight,
        );
      } else {
        setCardHeight(0);
      }
    }
  }, [props.hideExtraInteractions, scorecardView]);
  useEffect(() => {
    resizeListener();
  }, [props.events, props.poolFormat, resizeListener]);
  useEffect(() => {
    window.addEventListener("resize", resizeListener);
    return () => {
      window.removeEventListener("resize", resizeListener);
    };
  }, [resizeListener]);

  const cardStyle = useMemo(
    () =>
      cardHeight
        ? {
            maxHeight: cardHeight,
            minHeight: cardHeight,
          }
        : undefined,
    [cardHeight],
  );
  const notepadStyle = useMemo(
    () =>
      cardHeight
        ? {
            height: cardHeight - 24,
            display: flipHidden ? "none" : "flex",
          }
        : undefined,
    [cardHeight, flipHidden],
  );
  const analyzerStyle = useMemo(
    () =>
      cardHeight
        ? {
            height: cardHeight,
            display: flipHidden ? "none" : "flex",
          }
        : undefined,
    [cardHeight, flipHidden],
  );
  let title = `Turn ${turns.length + 1}`;
  let extra = null;
  if (flipEnabled) {
    if (props.isExamining) {
      title = !flipHidden ? "Analyzer" : `Turn ${turns.length + 1}`;
      extra = !flipHidden ? "View Scorecard" : "View Analyzer";
    } else {
      title = !flipHidden ? "Notepad" : `Turn ${turns.length + 1}`;
      extra = !flipHidden
        ? "View Scorecard"
        : `View Notepad${notepadHasContent ? " \u25cf" : ""}`;
    }
  }
  let contents = null;
  const { gameContext } = useGameContextStoreContext();
  const { handleExamineGoTo, examinedTurn } = useExamineStoreContext();

  const viewMenuItems = [
    { label: "One column", key: "single" },
    { label: "Two columns", key: "two-col" },
  ];

  const viewDropdown = (
    <Dropdown
      menu={{
        items: viewMenuItems,
        onClick: ({ key }) => {
          localStorage?.setItem(SCORECARD_VIEW_KEY, key);
          setScorecardView(key as ScorecardViewType);
        },
      }}
      trigger={["click"]}
      placement="bottomRight"
      overlayClassName="format-dropdown"
    >
      <Link to="/" onClick={(e) => e.preventDefault()}>
        Rearrange
      </Link>
    </Dropdown>
  );

  // Two-col player header (rendered outside the scroll container)
  let twoColHeader: React.ReactNode = null;
  if (flipHidden && scorecardView === "two-col") {
    const firstPlayerIdx =
      allTurns.length > 0 ? allTurns[0].events[0].playerIndex : 0;
    const leftPlayer = props.playerMeta[firstPlayerIdx];
    const rightPlayer = props.playerMeta[1 - firstPlayerIdx];
    twoColHeader = (
      <div className="two-col-header">
        <div className="two-col-player-name">{leftPlayer?.nickname || ""}</div>
        <div className="two-col-player-name">{rightPlayer?.nickname || ""}</div>
      </div>
    );
  }

  if (flipHidden) {
    if (scorecardView === "two-col") {
      // Determine which player went first (left column)
      const firstPlayerIdx =
        allTurns.length > 0 ? allTurns[0].events[0].playerIndex : 0;

      // Group turns into pairs: [leftTurn, rightTurn]
      // turns alternate players, starting with firstPlayerIdx
      const pairs: Array<
        [{ turn: Turn; idx: number } | null, { turn: Turn; idx: number } | null]
      > = [];
      let i = 0;
      while (i < allTurns.length) {
        const t = allTurns[i];
        const tPlayerIdx = t.events[0].playerIndex;
        if (tPlayerIdx === firstPlayerIdx) {
          const hasNext =
            i + 1 < allTurns.length &&
            allTurns[i + 1].events[0].playerIndex !== firstPlayerIdx;
          pairs.push([
            { turn: t, idx: i },
            hasNext ? { turn: allTurns[i + 1], idx: i + 1 } : null,
          ]);
          i += hasNext ? 2 : 1;
        } else {
          pairs.push([null, { turn: t, idx: i }]);
          i += 1;
        }
      }

      const firstPlayerForOpening = props.playerMeta[firstPlayerIdx];
      contents = (
        <>
          <div
            className={`two-col-scorecard${allTurns.length === 0 ? " two-col-scorecard-empty" : ""}`}
          >
            {allTurns.length === 0 && (
              <div className="two-col-opening-state">
                {firstPlayerForOpening?.nickname || "Player 1"} is first
              </div>
            )}
            {pairs.map(([left, right], pairIdx) => (
              <div className="two-col-row" key={pairIdx}>
                <div className="two-col-col">
                  {left ? (
                    <TwoColTurn
                      turn={left.turn}
                      board={props.allBoard ?? props.board}
                      alphabet={gameContext.alphabet}
                      onSeek={
                        props.isExamining
                          ? () => handleExamineGoTo(left.idx + 1)
                          : undefined
                      }
                      isSelected={
                        props.isExamining && examinedTurn === left.idx + 1
                      }
                      showComments={props.showComments}
                      comments={
                        props.comments?.filter(
                          (c) =>
                            c.eventNumber >= left.turn.firstEvtIdx &&
                            c.eventNumber <
                              left.turn.firstEvtIdx + left.turn.events.length,
                        ) ?? []
                      }
                      onOpenCommentsDrawer={props.onOpenCommentsDrawer}
                      turnIndex={left.idx}
                    />
                  ) : (
                    <div className="two-col-turn two-col-turn-empty" />
                  )}
                </div>
                <div className="two-col-col">
                  {right ? (
                    <TwoColTurn
                      turn={right.turn}
                      board={props.allBoard ?? props.board}
                      alphabet={gameContext.alphabet}
                      onSeek={
                        props.isExamining
                          ? () => handleExamineGoTo(right.idx + 1)
                          : undefined
                      }
                      isSelected={
                        props.isExamining && examinedTurn === right.idx + 1
                      }
                      showComments={props.showComments}
                      comments={
                        props.comments?.filter(
                          (c) =>
                            c.eventNumber >= right.turn.firstEvtIdx &&
                            c.eventNumber <
                              right.turn.firstEvtIdx + right.turn.events.length,
                        ) ?? []
                      }
                      onOpenCommentsDrawer={props.onOpenCommentsDrawer}
                      turnIndex={right.idx}
                    />
                  ) : (
                    <div className="two-col-turn two-col-turn-empty" />
                  )}
                </div>
              </div>
            ))}
          </div>
          {props.gameEpilog}
        </>
      );
    } else {
      const turnDisplay = (t: Turn, idx: number) => {
        if (t.events.length === 0) {
          return null;
        }
        const eventIdx = t.firstEvtIdx;
        const timeBankState = timeBankHistory?.get(eventIdx);

        return (
          <ScorecardTurn
            turn={t}
            board={props.board}
            key={`t_${idx + 0}`}
            playerMeta={props.playerMeta}
            showComments={props.showComments ?? false}
            comments={
              props.comments
                ? props.comments.filter(
                    (c) =>
                      c.eventNumber >= t.firstEvtIdx &&
                      c.eventNumber < t.firstEvtIdx + t.events.length,
                  )
                : []
            }
            alphabet={gameContext.alphabet}
            turnIndex={idx}
            onOpenCommentsDrawer={props.onOpenCommentsDrawer}
            isCorrespondence={props.isCorrespondence}
            timeBankP0={timeBankState?.p0TimeBank}
            timeBankP1={timeBankState?.p1TimeBank}
            eventIndex={eventIdx}
          />
        );
      };
      contents = (
        <>
          {turns.map(turnDisplay)}
          {props.gameEpilog}
        </>
      );
    }
  }

  return (
    <Card
      className={`score-card${flipHidden ? "" : " flipped"}${scorecardView === "two-col" ? " two-col-active" : ""}`}
      title={title}
      extra={
        <div className="score-card-extra">
          {viewDropdown}
          {isTablet() && extra ? (
            <button className="link" onClick={toggleFlipVisibility}>
              {extra}
            </button>
          ) : null}
        </div>
      }
    >
      {twoColHeader}
      <div ref={el} style={cardStyle}>
        {props.isExamining ? (
          <Analyzer style={analyzerStyle} />
        ) : (
          <Notepad style={notepadStyle} />
        )}
        {contents}
      </div>
    </Card>
  );
});
