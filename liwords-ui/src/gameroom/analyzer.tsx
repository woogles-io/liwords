import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import { Button, Card, Switch } from 'antd';
import { BulbOutlined } from '@ant-design/icons';
import { defaultLetterDistribution } from '../lobby/sought_game_interactions';
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { getWolges } from '../wasm/loader';
import { useMountedState } from '../utils/mounted';
import { RedoOutlined } from '@ant-design/icons';
import { EmptySpace, EphemeralTile } from '../utils/cwgame/common';
import { Unrace } from '../utils/unrace';
import { sortTiles } from '../store/constants';
import {
  GameEvent_Type,
  GameEvent_Direction,
} from '../gen/api/proto/macondo/macondo_pb';
import { GameState } from '../store/reducers/game_reducer';

type AnalyzerProps = {
  includeCard?: boolean;
  style?: React.CSSProperties;
  lexicon: string;
  variant?: string;
};

type JsonMove =
  | {
      equity: number;
      action: 'exchange';
      tiles: Array<number>;
      valid?: boolean;
      invalid_words?: Array<Array<number>>;
    }
  | {
      equity: number;
      action: 'play';
      down: boolean;
      lane: number;
      idx: number;
      word: Array<number>;
      score: number;
      valid?: boolean;
      invalid_words?: Array<Array<number>>;
    };

const jsonMoveToKey = (v: JsonMove) => {
  switch (v.action) {
    case 'exchange': {
      return JSON.stringify(
        ['action', 'tiles'].reduce(
          (h: { [key: string]: unknown }, k: string) => {
            h[k] = (v as { [key: string]: unknown })[k];
            return h;
          },
          {}
        )
      );
    }
    case 'play': {
      return JSON.stringify(
        ['action', 'down', 'lane', 'idx', 'word'].reduce(
          (h: { [key: string]: unknown }, k: string) => {
            h[k] = (v as { [key: string]: unknown })[k];
            return h;
          },
          {}
        )
      );
    }
    default: {
      return JSON.stringify({ invalid_object: v });
    }
  }
};

export type AnalyzerMove = {
  jsonKey: string;
  chosen?: boolean; // true for played, undefined for analyzer-generated moves
  valid?: boolean; // undefined for analyzer-generated moves
  invalid_words?: Array<string>;
  displayMove: string;
  coordinates: string;
  leave: string;
  leaveWithGaps: string;
  score: number;
  equity: number;
  row: number;
  col: number;
  vertical: boolean;
  tiles: string;
  isExchange: boolean;
};

export const analyzerMoveFromJsonMove = (
  move: JsonMove,
  dim: number,
  letters: string,
  rackNum: Array<number>,
  numToLabel: (n: number) => string
): AnalyzerMove => {
  const jsonKey = jsonMoveToKey(move);
  const defaultRet = {
    jsonKey,
    displayMove: '',
    coordinates: '',
    // always leave out leave
    vertical: false,
    col: 0,
    row: 0,
    score: 0,
    equity: 0.0,
    tiles: '',
    isExchange: false,
  };
  const makeLeaveStr = (leaveNum: Array<number>) => {
    let leaveStr = '';
    for (const t of leaveNum) {
      if (!isNaN(t)) {
        leaveStr += numToLabel(t);
      }
    }
    return leaveStr;
  };
  const addGapsToLeaveStr = (
    leaveNum: Array<number>,
    sortedLeaveStr: string
  ) => {
    let leaveStr = '';
    let r = 0;
    for (const t of leaveNum) {
      if (!isNaN(t)) {
        leaveStr += sortedLeaveStr[r++];
      } else {
        leaveStr += EmptySpace;
      }
    }
    return leaveStr;
  };
  switch (move.action) {
    case 'play': {
      const leaveNum = [...rackNum];
      let displayMove = '';
      let tilesBeingMoved = '';
      const vertical = move.down;
      const row = vertical ? move.idx : move.lane;
      const col = vertical ? move.lane : move.idx;
      const rowStr = String(row + 1);
      const colStr = String.fromCharCode(col + 0x41);
      const coordinates = vertical
        ? `${colStr}${rowStr}`
        : `${rowStr}${colStr}`;
      let r = row;
      let c = col;
      let inParen = false;
      for (const t of move.word) {
        if (t === 0) {
          if (!inParen) {
            displayMove += '(';
            inParen = true;
          }
          displayMove += letters[r * dim + c];
          tilesBeingMoved += '.';
        } else {
          if (inParen) {
            displayMove += ')';
            inParen = false;
          }
          const tileLabel = numToLabel(t);
          displayMove += tileLabel;
          tilesBeingMoved += tileLabel;
          // When t is negative, consume blank tile from rack.
          const usedTileIndex = leaveNum.lastIndexOf(Math.max(t, 0));
          if (usedTileIndex >= 0) leaveNum[usedTileIndex] = NaN;
        }
        if (vertical) ++r;
        else ++c;
      }
      if (inParen) displayMove += ')';
      const leaveStr = sortTiles(makeLeaveStr(leaveNum));
      return {
        jsonKey,
        displayMove,
        coordinates,
        leave: leaveStr,
        leaveWithGaps: addGapsToLeaveStr(leaveNum, leaveStr),
        vertical,
        col,
        row,
        score: move.score,
        equity: move.equity,
        tiles: tilesBeingMoved,
        isExchange: false,
      };
    }
    case 'exchange': {
      const leaveNum = [...rackNum];
      let tilesBeingMoved = '';
      for (const t of move.tiles) {
        const tileLabel = numToLabel(t);
        tilesBeingMoved += tileLabel;
        const usedTileIndex = leaveNum.lastIndexOf(t);
        if (usedTileIndex >= 0) leaveNum[usedTileIndex] = NaN;
      }
      tilesBeingMoved = sortTiles(tilesBeingMoved);
      const leaveStr = sortTiles(makeLeaveStr(leaveNum));
      return {
        ...defaultRet,
        displayMove: tilesBeingMoved ? `Exch. ${tilesBeingMoved}` : 'Pass',
        leave: leaveStr,
        leaveWithGaps: addGapsToLeaveStr(leaveNum, leaveStr),
        equity: move.equity,
        tiles: tilesBeingMoved,
        isExchange: true,
      };
    }
    default: {
      const leaveStr = makeLeaveStr(rackNum);
      return {
        ...defaultRet,
        leave: leaveStr,
        leaveWithGaps: addGapsToLeaveStr(rackNum, leaveStr),
      };
    }
  }
};

const parseExaminableGameContext = (
  examinableGameContext: GameState,
  lexicon: string,
  variant?: string
) => {
  const {
    board: { dim, letters },
    onturn,
    players,
  } = examinableGameContext;

  const letterDistribution = defaultLetterDistribution(lexicon);
  const labelToNum = labelToNumFor(letterDistribution);
  const numToLabel = numToLabelFor(letterDistribution);

  const rackStr = sortTiles(players[onturn].currentRack);
  const rackNum = Array.from(rackStr, labelToNum);

  let effectiveLexicon = lexicon;
  let rules = 'CrosswordGame';
  if (variant === 'wordsmog') {
    effectiveLexicon = `${lexicon}.WordSmog`;
    rules = 'WordSmog';
  } else if (variant === 'classic_super') {
    rules = 'CrosswordGameSuper';
  } else if (variant === 'wordsmog_super') {
    effectiveLexicon = `${lexicon}.WordSmog`;
    rules = 'WordSmogSuper';
  }
  if (letterDistribution !== 'english') {
    rules += `/${letterDistribution}`;
  }
  const boardObj = {
    rack: rackNum,
    board: Array.from(new Array(dim), (_, row) =>
      Array.from(letters.substr(row * dim, dim), labelToNum)
    ),
    lexicon: effectiveLexicon,
    leave:
      lexicon === 'CSW21'
        ? lexicon
        : letterDistribution === 'english' ||
          letterDistribution === 'german' ||
          letterDistribution === 'norwegian' ||
          letterDistribution === 'french' ||
          letterDistribution === 'catalan'
        ? letterDistribution
        : 'noleave',
    rules,
  };

  return { dim, letters, rackNum, effectiveLexicon, boardObj, numToLabel };
};

// Return 0 for both board's ' ' and rack's '?'.
// English-only.
const englishLabelToNum = (c: string) =>
  c >= 'A' && c <= 'Z'
    ? c.charCodeAt(0) - 0x40
    : c >= 'a' && c <= 'z'
    ? -(c.charCodeAt(0) - 0x60)
    : 0;

const GERMAN_TILES = Array.from('AÄBCDEFGHIJKLMNOÖPQRSTUÜVWXYZ');
const GERMAN_BLANK_TILES = Array.from('aäbcdefghijklmnoöpqrstuüvwxyz');

const germanLabelToNum = (c: string) => {
  let idx = GERMAN_TILES.indexOf(c);
  if (idx >= 0) return idx + 1;
  idx = GERMAN_BLANK_TILES.indexOf(c);
  if (idx >= 0) return -(idx + 1);
  return 0;
};

// note: internal wolges ordering
const NORWEGIAN_TILES = Array.from('ABCDEFGHIJKLMNOPQRSTUVWXYÜZÆÄØÖÅ');
const NORWEGIAN_BLANK_TILES = Array.from('abcdefghijklmnopqrstuvwxyüzæäøöå');

const norwegianLabelToNum = (c: string) => {
  let idx = NORWEGIAN_TILES.indexOf(c);
  if (idx >= 0) return idx + 1;
  idx = NORWEGIAN_BLANK_TILES.indexOf(c);
  if (idx >= 0) return -(idx + 1);
  return 0;
};

const labelToNumFor = (letterDistribution: string) => {
  switch (letterDistribution) {
    case 'english':
      return englishLabelToNum;
    case 'german':
      return germanLabelToNum;
    case 'norwegian':
      return norwegianLabelToNum;
    // XXX need catalan case
  }
  return englishLabelToNum;
};

// Return '?' for 0, because this is used for exchanges.
// English-only.
const englishNumToLabel = (n: number) =>
  n > 0
    ? String.fromCharCode(0x40 + n)
    : n < 0
    ? String.fromCharCode(0x60 - n)
    : '?';

const germanNumToLabel = (n: number) =>
  n > 0 ? GERMAN_TILES[n - 1] : n < 0 ? GERMAN_BLANK_TILES[-1 - n] : '?';

const norwegianNumToLabel = (n: number) =>
  n > 0 ? NORWEGIAN_TILES[n - 1] : n < 0 ? NORWEGIAN_BLANK_TILES[-1 - n] : '?';

const numToLabelFor = (letterDistribution: string) => {
  switch (letterDistribution) {
    case 'english':
      return englishNumToLabel;
    case 'german':
      return germanNumToLabel;
    case 'norwegian':
      return norwegianNumToLabel;
    // XXX need catalan case
  }
  return englishNumToLabel;
};

const AnalyzerContext = React.createContext<{
  autoMode: boolean;
  setAutoMode: React.Dispatch<React.SetStateAction<boolean>>;
  cachedMoves: Array<AnalyzerMove> | null;
  examinerLoading: boolean;
  requestAnalysis: (lexicon: string, variant?: string) => void;
  showMovesForTurn: number;
  setShowMovesForTurn: (a: number) => void;
}>({
  autoMode: false,
  cachedMoves: null,
  examinerLoading: false,
  requestAnalysis: (lexicon: string, variant?: string) => {},
  showMovesForTurn: -1,
  setShowMovesForTurn: (a: number) => {},
  setAutoMode: () => {},
});

export const AnalyzerContextProvider = ({
  children,
  nocache,
}: {
  children: React.ReactNode;
  nocache?: boolean;
}) => {
  const { useState } = useMountedState();

  const [, setMovesCacheId] = useState(0);
  const rerenderMoves = useCallback(
    () => setMovesCacheId((n) => (n + 1) | 0),
    []
  );
  const [showMovesForTurn, setShowMovesForTurn] = useState(-1);
  const [autoMode, setAutoMode] = useState(false);
  const [unrace, setUnrace] = useState(new Unrace());

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const examinerId = useRef(0);
  const movesCacheRef = useRef<Array<Array<AnalyzerMove> | null>>([]);
  useEffect(() => {
    examinerId.current = (examinerId.current + 1) | 0;
    movesCacheRef.current = [];
    setUnrace(new Unrace());
  }, [examinableGameContext.gameID]);

  const requestAnalysis = useCallback(
    (lexicon, variant) => {
      const examinerIdAtStart = examinerId.current;
      const turn = examinableGameContext.turns.length;
      if (nocache) {
        movesCacheRef.current = [];
      }
      const movesCache = movesCacheRef.current;
      // null = loading. undefined = not yet requested.
      if (movesCache[turn] !== undefined) return;
      movesCache[turn] = null;

      unrace.run(async () => {
        try {
          const {
            dim,
            letters,
            rackNum,
            effectiveLexicon,
            boardObj: bareBoardObj,
            numToLabel,
          } = parseExaminableGameContext(
            examinableGameContext,
            lexicon,
            variant
          );
          const boardObj = { ...bareBoardObj, count: 15 };

          const wolges = await getWolges(effectiveLexicon);
          if (examinerIdAtStart !== examinerId.current) return;

          const boardStr = JSON.stringify(boardObj);
          const movesStr = await wolges.analyze(boardStr);
          if (examinerIdAtStart !== examinerId.current) return;
          const movesObj = JSON.parse(movesStr) as Array<JsonMove>;

          const formattedMoves = movesObj.map((move) =>
            analyzerMoveFromJsonMove(move, dim, letters, rackNum, numToLabel)
          );
          movesCache[turn] = formattedMoves;
          rerenderMoves();
        } catch (e) {
          if (examinerIdAtStart === examinerId.current) {
            movesCache[turn] = [];
            rerenderMoves();
          }
          throw e;
        }
      });
    },
    [examinableGameContext, nocache, rerenderMoves, unrace]
  );

  const cachedMoves = movesCacheRef.current[examinableGameContext.turns.length];
  const examinerLoading = cachedMoves === null;
  const contextValue = useMemo(
    () => ({
      autoMode,
      setAutoMode,
      cachedMoves,
      examinerLoading,
      requestAnalysis,
      showMovesForTurn,
      setShowMovesForTurn,
    }),
    [
      autoMode,
      setAutoMode,
      cachedMoves,
      examinerLoading,
      requestAnalysis,
      showMovesForTurn,
      setShowMovesForTurn,
    ]
  );

  return <AnalyzerContext.Provider value={contextValue} children={children} />;
};

export const usePlaceMoveCallback = () => {
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { setDisplayedRack, setPlacedTiles, setPlacedTilesTempScore } =
    useTentativeTileContext();

  const placeMove = useCallback(
    (move: AnalyzerMove) => {
      const {
        board: { dim, letters },
      } = examinableGameContext;
      const newPlacedTiles = new Set<EphemeralTile>();
      let row = move.row;
      let col = move.col;
      let vertical = move.vertical;
      if (move.isExchange) {
        row = 0;
        col = 0;
        vertical = false;
      }
      for (const t of move.tiles) {
        if (move.isExchange) {
          while (letters[row * dim + col] !== EmptySpace) {
            ++col;
            if (col >= dim) {
              ++row;
              if (row >= dim) {
                // Cannot happen with the standard number of tiles and squares.
                row = dim - 1;
                col = dim - 1;
                break;
              }
              col = 0;
            }
          }
        }
        if (t !== '.') {
          newPlacedTiles.add({
            row,
            col,
            letter: t,
          });
        }
        if (vertical) ++row;
        else ++col;
      }
      setDisplayedRack(move.leaveWithGaps);
      setPlacedTiles(newPlacedTiles);
      setPlacedTilesTempScore(move.score);
    },
    [
      examinableGameContext,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
    ]
  );

  return placeMove;
};

export const Analyzer = React.memo((props: AnalyzerProps) => {
  const { useState } = useMountedState();
  const { lexicon, variant } = props;
  const {
    autoMode,
    setAutoMode,
    cachedMoves,
    examinerLoading,
    requestAnalysis,
    showMovesForTurn,
    setShowMovesForTurn,
  } = useContext(AnalyzerContext);

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { addHandleExaminer, removeHandleExaminer } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();

  const letterDistribution = useMemo(
    () => defaultLetterDistribution(lexicon),
    [lexicon]
  );
  const labelToNum = useMemo(
    () => labelToNumFor(letterDistribution),
    [letterDistribution]
  );

  const placeMove = usePlaceMoveCallback();

  const handleExaminer = useCallback(() => {
    setShowMovesForTurn(examinableGameContext.turns.length);
    requestAnalysis(lexicon, variant);
  }, [
    examinableGameContext.turns.length,
    lexicon,
    requestAnalysis,
    setShowMovesForTurn,
    variant,
  ]);

  const toggleAutoMode = useCallback(() => {
    setAutoMode((autoMode) => !autoMode);
  }, [setAutoMode]);
  // Let ExaminableStore activate this.
  useEffect(() => {
    addHandleExaminer(handleExaminer);
    return () => {
      removeHandleExaminer(handleExaminer);
    };
  }, [addHandleExaminer, removeHandleExaminer, handleExaminer]);

  // When at the last move, examineStoreContext.examinedTurn === Infinity.
  // To also detect new moves, we use examinableGameContext.turns.length.
  useEffect(() => {
    setShowMovesForTurn(-1);
  }, [examinableGameContext.turns.length, setShowMovesForTurn]);

  useEffect(() => {
    if (autoMode) {
      handleExaminer();
    }
  }, [autoMode, handleExaminer, showMovesForTurn]);

  const showMoves = showMovesForTurn === examinableGameContext.turns.length;
  const actualEvent = useMemo(() => {
    for (
      let i = examinableGameContext.turns.length;
      i < gameContext.turns.length;
      ++i
    ) {
      const evt = gameContext.turns[i];
      switch (evt.type) {
        case GameEvent_Type.TILE_PLACEMENT_MOVE:
        case GameEvent_Type.PHONY_TILES_RETURNED:
        case GameEvent_Type.PASS:
        case GameEvent_Type.EXCHANGE:
          return evt;
      }
    }
    return null;
  }, [gameContext, examinableGameContext]);
  const actualMove = useMemo(() => {
    const evt = actualEvent;
    if (evt) {
      switch (evt.type) {
        case GameEvent_Type.TILE_PLACEMENT_MOVE: {
          const down = evt.direction === GameEvent_Direction.VERTICAL;
          return {
            action: 'play',
            down,
            lane: down ? evt.column : evt.row,
            idx: down ? evt.row : evt.column,
            word: Array.from(evt.playedTiles, labelToNum),
            score: evt.score,
          };
        }
        case GameEvent_Type.PHONY_TILES_RETURNED: {
          return null;
        }
        case GameEvent_Type.PASS: {
          return { action: 'exchange', tiles: [] };
        }
        case GameEvent_Type.EXCHANGE: {
          return {
            action: 'exchange',
            tiles: Array.from(evt.exchanged, labelToNum),
          };
        }
      }
    }
    return null;
  }, [actualEvent, labelToNum]);
  const evaluatedMoveId = useRef(0);
  const [evaluatedMove, setEvaluatedMove] = useState<{
    evaluatedMoveId: number;
    moveObj: JsonMove | null;
    analyzerMove: AnalyzerMove | null;
  }>({
    evaluatedMoveId: -1,
    moveObj: null,
    analyzerMove: null,
  });
  useEffect(() => {
    evaluatedMoveId.current = (evaluatedMoveId.current + 1) | 0;
    const evaluatedMoveIdAtStart = evaluatedMoveId.current;
    if (actualMove) {
      (async () => {
        const {
          dim,
          letters,
          rackNum,
          effectiveLexicon,
          boardObj: bareBoardObj,
          numToLabel,
        } = parseExaminableGameContext(examinableGameContext, lexicon, variant);
        const boardObj = { ...bareBoardObj, plays: [actualMove] };

        const wolges = await getWolges(effectiveLexicon);
        if (evaluatedMoveIdAtStart !== evaluatedMoveId.current) return;

        const boardStr = JSON.stringify(boardObj);
        const movesStr = await wolges.play_score(boardStr);
        if (evaluatedMoveIdAtStart !== evaluatedMoveId.current) return;
        const movesObj = JSON.parse(movesStr);
        const moveObj = movesObj[0];

        if (moveObj.result === 'scored') {
          const analyzerMove = analyzerMoveFromJsonMove(
            moveObj,
            dim,
            letters,
            rackNum,
            numToLabel
          );
          setEvaluatedMove({
            evaluatedMoveId: evaluatedMoveIdAtStart,
            moveObj: moveObj,
            analyzerMove: {
              ...analyzerMove,
              chosen: true,
              valid: moveObj.valid,
              invalid_words: moveObj.invalid_words?.map(
                (tiles: Array<number>) => tiles.map((tile) => numToLabel(tile))
              ),
            },
          });
        } else {
          console.error('invalid move', moveObj);
          setEvaluatedMove({
            evaluatedMoveId: evaluatedMoveIdAtStart,
            moveObj: null,
            analyzerMove: null,
          });
        }
      })();
    }
  }, [actualMove, examinableGameContext, lexicon, variant]);
  const currentEvaluatedMove =
    evaluatedMove.evaluatedMoveId === evaluatedMoveId.current &&
    evaluatedMove.moveObj &&
    evaluatedMove.analyzerMove
      ? evaluatedMove
      : null;
  const moves = useMemo(() => {
    if (!showMoves) return null;
    if (cachedMoves == null) return cachedMoves;
    if (currentEvaluatedMove) {
      let found = false;
      const arr = [];
      for (const elt of cachedMoves) {
        if (!found) {
          if (currentEvaluatedMove.analyzerMove) {
            if (elt.jsonKey === currentEvaluatedMove.analyzerMove.jsonKey) {
              arr.push(currentEvaluatedMove.analyzerMove);
              found = true;
              continue;
            }
          }
          if (currentEvaluatedMove.moveObj) {
            if (elt.equity < currentEvaluatedMove.moveObj.equity) {
              // phonies may have better equity than valid plays
              if (currentEvaluatedMove.analyzerMove) {
                arr.push(currentEvaluatedMove.analyzerMove);
                found = true;
              }
            }
          }
        }
        arr.push(elt);
      }
      if (!found) {
        if (currentEvaluatedMove.analyzerMove) {
          arr.push(currentEvaluatedMove.analyzerMove);
        }
      }
      return arr;
    }
    return cachedMoves;
  }, [showMoves, cachedMoves, currentEvaluatedMove]);

  const showEquityLoss = React.useMemo(
    () => localStorage.getItem('enableShowEquityLoss') === 'true',
    []
  );
  const equityBase = React.useMemo(
    () =>
      showEquityLoss ? moves?.find((x) => x.valid ?? true)?.equity ?? 0 : 0,
    [moves, showEquityLoss]
  );
  const renderAnalyzerMoves = useMemo(
    () =>
      moves?.map((m: AnalyzerMove, idx) => (
        <tr
          key={idx}
          onClick={() => {
            placeMove(m);
          }}
          {...((m.chosen ?? false) && { className: 'move-chosen' })}
        >
          <td className="move-coords">{m.coordinates}</td>
          <td className="move">
            {m.displayMove}
            {m.invalid_words && m.invalid_words.length > 0 && (
              <React.Fragment>
                <br />(
                {m.invalid_words.map((word, idx) => (
                  <React.Fragment key={idx}>
                    {idx > 0 && ', '}
                    {word}*
                  </React.Fragment>
                ))}
                )
              </React.Fragment>
            )}
          </td>
          <td className="move-score">{m.score}</td>
          <td className="move-leave">{m.leave}</td>
          <td className="move-equity">
            {(m.equity - equityBase).toFixed(2)}
            {!(m.valid ?? true) && <React.Fragment>*</React.Fragment>}
          </td>
        </tr>
      )) ?? null,
    [equityBase, moves, placeMove]
  );
  const analyzerControls = (
    <div className="analyzer-controls">
      <Button
        className="analyze-trigger"
        shape="circle"
        icon={<BulbOutlined />}
        type="primary"
        onClick={handleExaminer}
        disabled={autoMode || examinerLoading}
      />
      <div className="auto-controls">
        <p className="auto-label">Auto</p>
        <Switch
          checked={autoMode}
          onChange={toggleAutoMode}
          className="auto-toggle"
          size="small"
        />
      </div>
    </div>
  );
  const analyzerContainer = (
    <div className="analyzer-container">
      {!examinerLoading ? (
        <div className="suggestions" style={props.style}>
          <table>
            <tbody>{renderAnalyzerMoves}</tbody>
          </table>
        </div>
      ) : (
        <div className="suggestions" style={props.style}>
          <RedoOutlined spin />
        </div>
      )}
      {!props.includeCard ? analyzerControls : null}
    </div>
  );
  if (props.includeCard) {
    return (
      <Card
        title="Analyzer"
        className="analyzer-card"
        extra={analyzerControls}
        tabIndex={-1} /* enable Examine shortcuts on clicking card title */
      >
        {analyzerContainer}
      </Card>
    );
  }
  return analyzerContainer;
});
