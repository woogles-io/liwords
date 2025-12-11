import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button, Card, Switch } from "antd";
import { BulbOutlined } from "@ant-design/icons";
import { defaultLetterDistribution } from "../lobby/sought_game_interactions";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { getLeaveKey, getLexiconKey, getWolges } from "../wasm/loader";
import { RedoOutlined } from "@ant-design/icons";
import {
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
  EphemeralTile,
  MachineLetter,
  MachineWord,
} from "../utils/cwgame/common";
import { Unrace } from "../utils/unrace";
import { sortTiles } from "../store/constants";
import {
  GameEvent_Type,
  GameEvent_Direction,
} from "../gen/api/vendor/macondo/macondo_pb";
import { GameState } from "../store/reducers/game_reducer";
import {
  Alphabet,
  machineLetterToRune,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";

type AnalyzerProps = {
  includeCard?: boolean;
  style?: React.CSSProperties;
};

type JsonMove =
  | {
      equity: number;
      action: "exchange";
      tiles: Array<number>;
      valid?: boolean;
      invalid_words?: Array<Array<number>>;
    }
  | {
      equity: number;
      action: "play";
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
    case "exchange": {
      return JSON.stringify(
        ["action", "tiles"].reduce(
          (h: { [key: string]: unknown }, k: string) => {
            h[k] = (v as { [key: string]: unknown })[k];
            return h;
          },
          {},
        ),
      );
    }
    case "play": {
      return JSON.stringify(
        ["action", "down", "lane", "idx", "word"].reduce(
          (h: { [key: string]: unknown }, k: string) => {
            h[k] = (v as { [key: string]: unknown })[k];
            return h;
          },
          {},
        ),
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
  leave: MachineWord;
  leaveWithGaps: MachineWord;
  score: number;
  equity: number;
  row: number;
  col: number;
  vertical: boolean;
  tiles: MachineWord;
  isExchange: boolean;
};

const wolgesLetterToLiwordsLetter = (i: number) => {
  if (i < 0) {
    // wolges-wasm encodes blanks as negative numbers. Convert to our internal
    // format.
    i = -i | 0x80;
  }
  return i;
};

const liwordsLetterToWolgesLetter = (i: number) => {
  if ((i & 0x80) > 0) {
    // This is a blank. Convert to a wolges blank.
    return -(i & 0x7f);
  }
  return i;
};

const wolgesLabelsToLetter = (runes: string, alphabet: Alphabet) => {
  const resp = runesToMachineWord(runes, alphabet);
  return resp.map(liwordsLetterToWolgesLetter);
};

const wolgesLetterToLabel = (i: number, alphabet: Alphabet) => {
  i = wolgesLetterToLiwordsLetter(i);
  return machineLetterToRune(i, alphabet, false, true);
};

export const analyzerMoveFromJsonMove = (
  move: JsonMove,
  dim: number,
  letters: Array<MachineLetter>,
  rackNum: MachineWord,
  alphabet: Alphabet,
): AnalyzerMove => {
  const jsonKey = jsonMoveToKey(move);
  const defaultRet = {
    jsonKey,
    displayMove: "",
    coordinates: "",
    // always leave out leave
    vertical: false,
    col: 0,
    row: 0,
    score: 0,
    equity: 0.0,
    tiles: new Array<MachineLetter>(),
    isExchange: false,
  };

  switch (move.action) {
    case "play": {
      let leaveNum = [...rackNum];
      const leaveWithGaps = [...rackNum];
      let displayMove = "";
      const tilesBeingMoved = new Array<MachineLetter>();
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
            displayMove += "(";
            inParen = true;
          }
          displayMove += machineLetterToRune(
            letters[r * dim + c],
            alphabet,
            false,
            true,
          );
          tilesBeingMoved.push(0); // through space
        } else {
          if (inParen) {
            displayMove += ")";
            inParen = false;
          }
          const tileLabel = wolgesLetterToLabel(t, alphabet);
          displayMove += tileLabel;
          tilesBeingMoved.push(t);
          // When t is negative, consume blank tile from rack.
          const usedTileIndex = leaveNum.lastIndexOf(Math.max(t, 0));
          if (usedTileIndex >= 0) {
            leaveWithGaps[usedTileIndex] = EmptyRackSpaceMachineLetter;
            leaveNum[usedTileIndex] = EmptyRackSpaceMachineLetter;
          }
        }
        if (vertical) ++r;
        else ++c;
      }
      if (inParen) displayMove += ")";
      // sortTiles takes out the gaps:
      leaveNum = sortTiles(leaveNum, alphabet);
      return {
        jsonKey,
        displayMove,
        coordinates,
        leave: leaveNum,
        leaveWithGaps,
        vertical,
        col,
        row,
        score: move.score,
        equity: move.equity,
        tiles: tilesBeingMoved,
        isExchange: false,
      };
    }
    case "exchange": {
      let leaveNum = [...rackNum];
      const leaveWithGaps = [...rackNum];

      let tilesBeingMoved = new Array<MachineLetter>();
      for (const t of move.tiles) {
        tilesBeingMoved.push(t);
        const usedTileIndex = leaveNum.lastIndexOf(t);
        if (usedTileIndex >= 0) {
          leaveWithGaps[usedTileIndex] = EmptyRackSpaceMachineLetter;
          leaveNum[usedTileIndex] = EmptyRackSpaceMachineLetter;
        }
      }
      tilesBeingMoved = sortTiles(tilesBeingMoved, alphabet);
      leaveNum = sortTiles(leaveNum, alphabet);

      return {
        ...defaultRet,
        displayMove:
          tilesBeingMoved.length > 0
            ? `Exch. ${machineWordToRunes(
                tilesBeingMoved,
                alphabet,
                false,
                true,
              )}`
            : "Pass",
        leave: leaveNum,
        leaveWithGaps,
        equity: move.equity,
        tiles: tilesBeingMoved,
        isExchange: true,
      };
    }
    default: {
      const leaveNum = [...rackNum];

      return {
        ...defaultRet,
        leave: leaveNum,
        leaveWithGaps: leaveNum,
      };
    }
  }
};

const parseExaminableGameContext = (
  examinableGameContext: GameState,
  lexicon: string,
  variant?: string,
) => {
  const {
    board: { dim, letters },
    onturn,
    players,
    alphabet,
  } = examinableGameContext;

  const letterDistribution = defaultLetterDistribution(lexicon);
  // const labelToNum = labelToNumFor(letterDistribution);
  // const numToLabel = numToLabelFor(letterDistribution);

  const rackNum = sortTiles(players[onturn].currentRack, alphabet);

  let loadableKey = lexicon;
  let rules = "CrosswordGame";
  if (variant === "wordsmog" || variant === "wordsmog_super") {
    rules = "WordSmog";
    loadableKey += ".WordSmog";
  }
  if (variant === "classic_super" || variant === "wordsmog_super") {
    // only english and catalan supported.
    rules += "Super";
    loadableKey = `super-${loadableKey}`;
  }
  if (letterDistribution !== "english") {
    rules += `/${letterDistribution}`;
  }
  const boardObj = {
    rack: rackNum,
    board: Array.from(new Array(dim), (_, row) =>
      Array.from(
        letters
          .slice(row * dim, row * dim + dim)
          // I like writing write-only code.
          .map((l) => (l & 0x80 ? -(l & 0x7f) : l)),
      ),
    ),
    lexicon: getLexiconKey(loadableKey),
    leave: getLeaveKey(loadableKey),
    rules,
  };

  return {
    dim,
    letters,
    rackNum,
    loadableKey,
    boardObj,
    alphabet,
  };
};

type CachedAnalyzerMoves = {
  jsonKey: string;
  analyzerMoves: Array<AnalyzerMove> | null;
};

const AnalyzerContext = React.createContext<{
  autoMode: boolean;
  setAutoMode: React.Dispatch<React.SetStateAction<boolean>>;
  cachedMoves: Array<AnalyzerMove> | null | undefined;
  examinerLoading: boolean;
  requestAnalysis: () => void;
  showMovesForTurn: number;
  setShowMovesForTurn: (a: number) => void;
  lexicon: string;
  variant?: string;
}>({
  autoMode: false,
  cachedMoves: null,
  examinerLoading: false,
  requestAnalysis: () => {},
  showMovesForTurn: -1,
  setShowMovesForTurn: (a: number) => {},
  setAutoMode: () => {},
  lexicon: "",
  variant: undefined,
});

type AnalyzerContextProviderProps = {
  children: React.ReactNode;
  lexicon: string;
  variant?: string;
};

export const AnalyzerContextProvider = (
  props: AnalyzerContextProviderProps,
) => {
  const { children, lexicon, variant } = props;
  const [, setMovesCacheId] = useState(0);
  const rerenderMoves = useCallback(
    () => setMovesCacheId((n) => (n + 1) | 0),
    [],
  );
  const [showMovesForTurn, setShowMovesForTurn] = useState(-1);
  const [autoMode, setAutoMode] = useState(false);
  const [unrace, setUnrace] = useState(new Unrace());

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const examinerId = useRef(0);
  const movesCacheRef = useRef<Array<CachedAnalyzerMoves>>([]);
  useEffect(() => {
    examinerId.current = (examinerId.current + 1) | 0;
    movesCacheRef.current = [];
    setUnrace(new Unrace());
  }, [examinableGameContext.gameID]);

  const parsedEgc = React.useMemo(() => {
    try {
      return parseExaminableGameContext(
        examinableGameContext,
        lexicon,
        variant,
      );
    } catch {
      return null;
    }
  }, [examinableGameContext, lexicon, variant]);
  const boardJsonKey = React.useMemo(
    () => JSON.stringify(parsedEgc?.boardObj),
    [parsedEgc],
  );
  const requestAnalysis = useCallback(() => {
    if (!parsedEgc) return;
    const examinerIdAtStart = examinerId.current;
    const turn = examinableGameContext.turns.length;
    const movesCache = movesCacheRef.current;
    // [boardJsonKey, null] = loading. undefined = not yet requested.
    // phrased this way so that in future it's possible for movesCache[turn] to be null (as opposed to undefined).
    if (
      (movesCache[turn] &&
        (movesCache[turn].jsonKey === boardJsonKey
          ? movesCache[turn].analyzerMoves
          : undefined)) !== undefined
    )
      return;
    movesCache[turn] = {
      jsonKey: boardJsonKey,
      analyzerMoves: null,
    };

    unrace.run(async () => {
      try {
        const {
          dim,
          letters,
          rackNum,
          loadableKey,
          boardObj: bareBoardObj,
          alphabet,
        } = parsedEgc;
        const boardObj = { ...bareBoardObj, count: 15 };

        const wolges = await getWolges(loadableKey);
        if (
          examinerIdAtStart !== examinerId.current ||
          movesCache[turn]?.jsonKey !== boardJsonKey
        )
          return;

        const boardStr = JSON.stringify(boardObj);
        const movesStr = await wolges.analyze(boardStr);
        if (
          examinerIdAtStart !== examinerId.current ||
          movesCache[turn]?.jsonKey !== boardJsonKey
        )
          return;
        const movesObj = JSON.parse(movesStr) as Array<JsonMove>;

        const formattedMoves = movesObj.map((move) =>
          analyzerMoveFromJsonMove(move, dim, letters, rackNum, alphabet),
        );
        movesCache[turn] = {
          jsonKey: boardJsonKey,
          analyzerMoves: formattedMoves,
        };
        rerenderMoves();
      } catch (e) {
        if (examinerIdAtStart === examinerId.current) {
          movesCache[turn] = {
            jsonKey: boardJsonKey,
            analyzerMoves: [],
          };
          rerenderMoves();
        }
        throw e;
      }
    });
  }, [examinableGameContext, rerenderMoves, unrace, boardJsonKey, parsedEgc]);

  const cachedMovesThisTurn =
    movesCacheRef.current[examinableGameContext.turns.length];
  const cachedMoves =
    cachedMovesThisTurn &&
    (cachedMovesThisTurn.jsonKey === boardJsonKey
      ? cachedMovesThisTurn.analyzerMoves
      : undefined);
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
      lexicon,
      variant,
    }),
    [
      autoMode,
      setAutoMode,
      cachedMoves,
      examinerLoading,
      requestAnalysis,
      showMovesForTurn,
      setShowMovesForTurn,
      lexicon,
      variant,
    ],
  );

  return <AnalyzerContext.Provider value={contextValue} children={children} />;
};

export const usePlaceMoveCallback = () => {
  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const {
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
    setPendingExchangeTiles,
  } = useTentativeTileContext();

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
          while (letters[row * dim + col] !== EmptyBoardSpaceMachineLetter) {
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
        if (t !== 0) {
          newPlacedTiles.add({
            row,
            col,
            letter: wolgesLetterToLiwordsLetter(t),
          });
        }
        if (vertical) ++row;
        else ++col;
      }
      setDisplayedRack(move.leaveWithGaps);
      setPlacedTiles(newPlacedTiles);
      setPlacedTilesTempScore(move.score);
      // Set pending exchange tiles if this is an exchange move
      setPendingExchangeTiles(move.isExchange ? move.tiles : null);
    },
    [
      examinableGameContext,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
      setPendingExchangeTiles,
    ],
  );

  return placeMove;
};

export const Analyzer = React.memo((props: AnalyzerProps) => {
  const {
    autoMode,
    setAutoMode,
    cachedMoves,
    examinerLoading,
    requestAnalysis,
    showMovesForTurn,
    setShowMovesForTurn,
    lexicon,
    variant,
  } = useContext(AnalyzerContext);

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { addHandleExaminer, removeHandleExaminer } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();

  const placeMove = usePlaceMoveCallback();

  const handleExaminer = useCallback(() => {
    setShowMovesForTurn(examinableGameContext.turns.length);
    requestAnalysis();
  }, [
    examinableGameContext.turns.length,
    requestAnalysis,
    setShowMovesForTurn,
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
            action: "play",
            down,
            lane: down ? evt.column : evt.row,
            idx: down ? evt.row : evt.column,
            word: wolgesLabelsToLetter(
              evt.playedTiles,
              examinableGameContext.alphabet,
            ),
            score: evt.score,
          };
        }
        case GameEvent_Type.PHONY_TILES_RETURNED: {
          return null;
        }
        case GameEvent_Type.PASS: {
          return { action: "exchange", tiles: [] };
        }
        case GameEvent_Type.EXCHANGE: {
          return {
            action: "exchange",
            tiles: runesToMachineWord(
              evt.exchanged,
              examinableGameContext.alphabet,
            ),
          };
        }
      }
    }
    return null;
  }, [actualEvent, examinableGameContext.alphabet]);
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
          loadableKey,
          boardObj: bareBoardObj,
          alphabet,
        } = parseExaminableGameContext(examinableGameContext, lexicon, variant);
        const boardObj = { ...bareBoardObj, plays: [actualMove] };

        const wolges = await getWolges(loadableKey);
        if (evaluatedMoveIdAtStart !== evaluatedMoveId.current) return;

        const boardStr = JSON.stringify(boardObj);
        const movesStr = await wolges.play_score(boardStr);
        if (evaluatedMoveIdAtStart !== evaluatedMoveId.current) return;
        const movesObj = JSON.parse(movesStr);
        const moveObj = movesObj[0];

        if (moveObj.result === "scored") {
          const analyzerMove = analyzerMoveFromJsonMove(
            moveObj,
            dim,
            letters,
            rackNum,
            alphabet,
          );
          setEvaluatedMove({
            evaluatedMoveId: evaluatedMoveIdAtStart,
            moveObj: moveObj,
            analyzerMove: {
              ...analyzerMove,
              chosen: true,
              valid: moveObj.valid,
              invalid_words: moveObj.invalid_words?.map(
                (tiles: Array<number>) => machineWordToRunes(tiles, alphabet),
              ),
            },
          });
        } else {
          console.error("invalid move", moveObj);
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
    () => localStorage.getItem("enableShowEquityLoss") === "true",
    [],
  );
  const equityBase = React.useMemo(
    () =>
      showEquityLoss ? (moves?.find((x) => x.valid ?? true)?.equity ?? 0) : 0,
    [moves, showEquityLoss],
  );
  const renderAnalyzerMoves = useMemo(
    () =>
      moves?.map((m: AnalyzerMove, idx) => (
        <tr
          key={idx}
          onClick={() => {
            placeMove(m);
          }}
          {...((m.chosen ?? false) && { className: "move-chosen" })}
        >
          <td className="move-coords">{m.coordinates}</td>
          <td className="move">
            {m.displayMove}
            {m.invalid_words && m.invalid_words.length > 0 && (
              <React.Fragment>
                <br />(
                {m.invalid_words.map((word, idx) => (
                  <React.Fragment key={idx}>
                    {idx > 0 && ", "}
                    {word}*
                  </React.Fragment>
                ))}
                )
              </React.Fragment>
            )}
          </td>
          <td className="move-score">{m.score}</td>
          <td className="move-leave">
            {machineWordToRunes(m.leave, examinableGameContext.alphabet)}
          </td>
          <td className="move-equity">
            {(m.equity - equityBase).toFixed(2)}
            {!(m.valid ?? true) && <React.Fragment>*</React.Fragment>}
          </td>
        </tr>
      )) ?? null,
    [equityBase, examinableGameContext.alphabet, moves, placeMove],
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
        tabIndex={-1} /* enable Analyze shortcuts on clicking card title */
      >
        {analyzerContainer}
      </Card>
    );
  }
  return analyzerContainer;
});
