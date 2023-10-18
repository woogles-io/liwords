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
import { getWolges, getMagpie, MagpieMoveTypes } from '../wasm/loader';
import { useMountedState } from '../utils/mounted';
import { RedoOutlined } from '@ant-design/icons';
import {
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
  EphemeralTile,
  MachineLetter,
  MachineWord,
} from '../utils/cwgame/common';
import { Unrace } from '../utils/unrace';
import { sortTiles } from '../store/constants';
import {
  GameEvent_Type,
  GameEvent_Direction,
} from '../gen/api/proto/macondo/macondo_pb';
import { GameState } from '../store/reducers/game_reducer';
import {
  Alphabet,
  machineLetterToRune,
  machineWordToRunes,
  runesToMachineWord,
} from '../constants/alphabets';
import { parseCoordinates, toFen } from '../utils/cwgame/board';
import { computeLeaveML } from '../utils/cwgame/game_event';
import { subscribe } from '../shared/pubsub';

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
  // select just a few keys.
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

type AnalyzerMove = {
  jsonKey: string;
  chosen?: boolean; // true for played, undefined for analyzer-generated moves
  valid?: boolean;
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
  winpct?: number;
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

const analyzerMoveFromJsonMove = (
  move: JsonMove,
  dim: number,
  letters: Array<MachineLetter>,
  rackNum: MachineWord,
  alphabet: Alphabet
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
    tiles: new Array<MachineLetter>(),
    isExchange: false,
  };

  switch (move.action) {
    case 'play': {
      let leaveNum = [...rackNum];
      const leaveWithGaps = [...rackNum];
      let displayMove = '';
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
            displayMove += '(';
            inParen = true;
          }
          displayMove += machineLetterToRune(
            letters[r * dim + c],
            alphabet,
            false,
            true
          );
          tilesBeingMoved.push(0); // through space
        } else {
          if (inParen) {
            displayMove += ')';
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
      if (inParen) displayMove += ')';
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
    case 'exchange': {
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
                true
              )}`
            : 'Pass',
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

const analyzerMoveFromUCGIString = (
  ucgis: string,
  dim: number,
  letters: Array<MachineLetter>,
  rackNum: MachineWord,
  alphabet: Alphabet
): AnalyzerMove => {
  /**
   * info currmove c9.ZEK sc 26 wp 70.503 wpe 4.870 eq -18.027 eqe 4.801 it 519 ig 1 ply1-scm 30.073 ply1-scd 23.176 ply1-bp 18.919 ply2-scm 19.492 ply2-scd 4.216 ply2-bp 0.000 ply3-scm 33.730 ply3-scd 23.773 ply3-bp 20.231 ply4-scm 24.286 ply4-scd 14.759 ply4-bp 6.509 ply5-scm 20.287 ply5-scd 17.196 ply5-bp 5.689
   * currmove 6g.DIPETAZ result scored valid false invalid_words WIFAY,ZGENUINE,DIPETAZ sc 57 eq 72.947
   */
  if (ucgis.startsWith('info ')) {
    ucgis = ucgis.substring(5);
  }
  const splitstr = ucgis.trim().split(' ');

  const kv: { [x: string]: string } = {};
  for (let i = 0; i < splitstr.length; i += 2) {
    kv[splitstr[i]] = splitstr[i + 1];
  }

  const jsonKey = JSON.stringify({ move: kv['currmove'] });

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
    winpct: 0.0,
    tiles: new Array<MachineLetter>(),
    isExchange: false,
  };

  if (kv['currmove'] === 'pass') {
    return {
      ...defaultRet,
      leave: rackNum,
      leaveWithGaps: rackNum,
      equity: parseFloat(kv['eq']),
      winpct: 'wp' in kv ? parseFloat(kv['wp']) : undefined,
    };
  }
  if (kv['currmove'].startsWith('ex.')) {
    let leaveNum = [...rackNum];
    const leaveWithGaps = [...rackNum];

    let tilesBeingMoved = new Array<MachineLetter>();

    const parts = kv['currmove'].split('.');
    // convert to machine letters.. only to convert back to runes. It's ok.
    const word = runesToMachineWord(parts[1], alphabet);

    for (const t of word) {
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
              true
            )}`
          : 'Pass',
      leave: leaveNum,
      leaveWithGaps,
      equity: parseFloat(kv['eq']),
      winpct: 'wp' in kv ? parseFloat(kv['wp']) : undefined,
      tiles: tilesBeingMoved,
      isExchange: true,
    };
  } else {
    // it's a tile move play
    let leaveNum = [...rackNum];
    const leaveWithGaps = [...rackNum];
    let displayMove = '';
    const tilesBeingMoved = new Array<MachineLetter>();

    const parts = kv['currmove'].split('.');
    // coordinates will be valid, since UCGI will not send invalid coordinates.
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const coords = parseCoordinates(parts[0])!;
    const vertical = !coords.horizontal;
    const row = coords.row;
    const col = coords.col;

    const rowStr = String(row + 1);
    const colStr = String.fromCharCode(col + 0x41);
    const coordinates = vertical ? `${colStr}${rowStr}` : `${rowStr}${colStr}`;

    // convert to machine letters.. only to convert back to runes. It's ok.
    const word = runesToMachineWord(parts[1], alphabet);

    // copy some of this from analyzerMoveFromJsonMove
    let r = row;
    let c = col;
    let inParen = false;
    for (const t of word) {
      if (letters[r * dim + c] !== 0) {
        // There is already a tile in the board at this position.
        if (!inParen) {
          displayMove += '(';
          inParen = true;
        }
        displayMove += machineLetterToRune(
          letters[r * dim + c],
          alphabet,
          false,
          true
        );
        tilesBeingMoved.push(0); // through space
      } else {
        if (inParen) {
          displayMove += ')';
          inParen = false;
        }
        const tileLabel = machineLetterToRune(t, alphabet, false, true);
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
    if (inParen) displayMove += ')';

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
      score: parseInt(kv['sc'], 10),
      equity: parseFloat(kv['eq']),
      winpct: 'wp' in kv ? parseFloat(kv['wp']) : undefined,
      tiles: tilesBeingMoved,
      isExchange: false,
      valid: kv['valid'] === 'true' || kv['valid'] == undefined,
      invalid_words: kv['invalid_words']?.split(',') ?? undefined,
    };
  }
};

const parseExaminableGameContext = (
  examinableGameContext: GameState,
  lexicon: string,
  variant?: string
) => {
  const { board, onturn, players, alphabet } = examinableGameContext;
  const { dim, letters } = board;
  const letterDistribution = defaultLetterDistribution(lexicon);
  // const labelToNum = labelToNumFor(letterDistribution);
  // const numToLabel = numToLabelFor(letterDistribution);

  const rackNum = sortTiles(players[onturn].currentRack, alphabet);

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
      Array.from(
        letters
          .slice(row * dim, row * dim + dim)
          // I like writing write-only code.
          .map((l) => (l & 0x80 ? -(l & 0x7f) : l))
      )
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

  const fen = toFen(board, alphabet);
  const ourRack = machineWordToRunes(rackNum, alphabet);
  const ourScore = players[onturn].score;
  const theirScore = players[1 - onturn].score;
  const cgp = `${fen} ${ourRack}/ ${ourScore}/${theirScore} 0 lex ${lexicon}; ld ${letterDistribution};`;

  return { dim, letters, rackNum, effectiveLexicon, boardObj, alphabet, cgp };
};

const AnalyzerContext = React.createContext<{
  autoMode: boolean;
  setAutoMode: React.Dispatch<React.SetStateAction<boolean>>;
  staticOnlyMode: boolean;
  setStaticOnlyMode: React.Dispatch<React.SetStateAction<boolean>>;
  cachedMoves: Array<AnalyzerMove> | null;
  examinerLoading: boolean;
  requestAnalysis: (lexicon: string, variant?: string) => void;
  showMovesForTurn: number;
  setShowMovesForTurn: (a: number) => void;
  nps: number;
}>({
  autoMode: false,
  staticOnlyMode: false,
  cachedMoves: null,
  examinerLoading: false,
  requestAnalysis: (lexicon: string, variant?: string) => {},
  showMovesForTurn: -1,
  setShowMovesForTurn: (a: number) => {},
  setAutoMode: () => {},
  setStaticOnlyMode: () => {},
  nps: 0,
});

type MovesCache = { [key: string]: Array<AnalyzerMove> | null };

const cacheKey = (turn: number, staticOnlyMode: boolean) =>
  `t${turn}-st${staticOnlyMode}`;

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
  const [staticOnlyMode, setStaticOnlyMode] = useState(false);
  const [unrace, setUnrace] = useState(new Unrace());
  const [nps, setNps] = useState(0);

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();

  const examinerId = useRef(0);
  const movesCacheRef = useRef<MovesCache>({});
  const maxThreads = Math.max(navigator.hardwareConcurrency - 1, 1);
  useEffect(() => {
    examinerId.current = (examinerId.current + 1) | 0;
    movesCacheRef.current = {};
    setUnrace(new Unrace());
  }, [examinableGameContext.gameID]);

  useEffect(() => {
    // note this is not really used for anything other than logging
    console.log('Subscribing to magpie.stdout');
    const { unsubscribe } = subscribe('magpie.stdout', (d: string) => {
      console.log('data', d);
    });

    return () => {
      console.log('unsubscribing');
      unsubscribe();
    };
  }, []);

  const requestAnalysis = useCallback(
    (lexicon, variant) => {
      const examinerIdAtStart = examinerId.current;
      const turn = examinableGameContext.turns.length;
      if (nocache) {
        movesCacheRef.current = {};
      }
      const movesCache = movesCacheRef.current;
      const ck = cacheKey(turn, staticOnlyMode);
      // null = loading. undefined = not yet requested.
      if (movesCache[ck] !== undefined) return;
      movesCache[ck] = null;

      unrace.run(async () => {
        try {
          const {
            dim,
            letters,
            rackNum,
            effectiveLexicon,
            boardObj: bareBoardObj,
            alphabet,
            cgp,
          } = parseExaminableGameContext(
            examinableGameContext,
            lexicon,
            variant
          );
          const boardObj = { ...bareBoardObj, count: 15 };
          let wolges, magpie;
          if (variant === 'wordsmog' || variant === 'classic_super') {
            // magpie doesn't yet support these variants
            wolges = await getWolges(effectiveLexicon);
          } else {
            magpie = await getMagpie(effectiveLexicon);
          }
          if (examinerIdAtStart !== examinerId.current) return;

          const boardStr = JSON.stringify(boardObj);
          if (wolges) {
            const movesStr = await wolges.analyze(boardStr);
            if (examinerIdAtStart !== examinerId.current) return;

            const movesObj = JSON.parse(movesStr) as Array<JsonMove>;
            const formattedMoves = movesObj.map((move) =>
              analyzerMoveFromJsonMove(move, dim, letters, rackNum, alphabet)
            );
            movesCache[ck] = formattedMoves;
            rerenderMoves();
          } else if (magpie) {
            let resp = '';
            if (staticOnlyMode) {
              resp = await magpie.staticEvaluation(cgp, 15);
              const formattedMoves = resp
                .split('\n')
                .filter((move: string) => move && !move.startsWith('bestmove'))
                .map((move: string) =>
                  analyzerMoveFromUCGIString(
                    move,
                    dim,
                    letters,
                    rackNum,
                    alphabet
                  )
                );
              movesCache[ck] = formattedMoves;
              rerenderMoves();
            } else {
              await magpie.processUCGICommand(`position cgp ${cgp}`);
              await magpie.processUCGICommand(
                `go sim threads ${maxThreads} plays 40 stopcondition 95 depth 5 i 5000 checkstop 500`
              );
              const interval = setInterval(async () => {
                const status = (await magpie.searchStatus()) as string;
                const lines = status.split('\n');
                const moves = [];
                for (let i = 0; i < lines.length; i++) {
                  if (lines[i].startsWith('info nps')) {
                    setNps(parseFloat(lines[i].substring(9)));
                  } else if (
                    lines[i].startsWith('bestmove') ||
                    lines[i].startsWith('bestsofar')
                  ) {
                    continue;
                  } else if (lines[i].trim() !== '') {
                    moves.push(
                      analyzerMoveFromUCGIString(
                        lines[i],
                        dim,
                        letters,
                        rackNum,
                        alphabet
                      )
                    );
                  }
                }
                movesCache[ck] = moves;
                rerenderMoves();
                if (status.includes('bestmove')) {
                  console.log('stop interval');
                  clearInterval(interval);
                }
              }, 500);
            }
            // setTimeout(() => {
            //   console.log('gonna get search status');
            //   const status = analyzerBinary.searchStatus();
            //   console.log('status', status);
            // }, 100);
          }
        } catch (e) {
          if (examinerIdAtStart === examinerId.current) {
            movesCache[ck] = [];
            rerenderMoves();
          }
          throw e;
        }
      });
    },
    [
      examinableGameContext,
      nocache,
      rerenderMoves,
      staticOnlyMode,
      unrace,
      maxThreads,
    ]
  );
  const cck = cacheKey(examinableGameContext.turns.length, staticOnlyMode);
  const cachedMoves = movesCacheRef.current[cck];
  const examinerLoading = cachedMoves === null;
  const contextValue = useMemo(
    () => ({
      autoMode,
      setAutoMode,
      staticOnlyMode,
      setStaticOnlyMode,
      cachedMoves,
      examinerLoading,
      requestAnalysis,
      showMovesForTurn,
      setShowMovesForTurn,
      nps,
    }),
    [
      autoMode,
      setAutoMode,
      staticOnlyMode,
      setStaticOnlyMode,
      cachedMoves,
      examinerLoading,
      requestAnalysis,
      showMovesForTurn,
      setShowMovesForTurn,
      nps,
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
    staticOnlyMode,
    setStaticOnlyMode,
    cachedMoves,
    examinerLoading,
    requestAnalysis,
    showMovesForTurn,
    setShowMovesForTurn,
    nps,
  } = useContext(AnalyzerContext);

  const { gameContext: examinableGameContext } =
    useExaminableGameContextStoreContext();
  const { addHandleExaminer, removeHandleExaminer } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();

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
  const toggleStaticOnlyMode = useCallback(() => {
    setStaticOnlyMode((staticOnlyMode) => !staticOnlyMode);
  }, [setStaticOnlyMode]);

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
            word: wolgesLabelsToLetter(
              evt.playedTiles,
              examinableGameContext.alphabet
            ),
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
            tiles: runesToMachineWord(
              evt.exchanged,
              examinableGameContext.alphabet
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
          effectiveLexicon,
          boardObj: bareBoardObj,
          alphabet,
          cgp,
        } = parseExaminableGameContext(examinableGameContext, lexicon, variant);
        const boardObj = { ...bareBoardObj, plays: [actualMove] };
        let wolges, magpie;

        if (variant === 'wordsmog' || variant === 'classic_super') {
          wolges = await getWolges(effectiveLexicon);
        } else {
          magpie = await getMagpie(effectiveLexicon);
        }

        if (evaluatedMoveIdAtStart !== evaluatedMoveId.current) return;

        if (wolges) {
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
              alphabet
            );
            setEvaluatedMove({
              evaluatedMoveId: evaluatedMoveIdAtStart,
              moveObj: moveObj,
              analyzerMove: {
                ...analyzerMove,
                chosen: true,
                valid: moveObj.valid,
                invalid_words: moveObj.invalid_words?.map(
                  (tiles: Array<number>) => machineWordToRunes(tiles, alphabet)
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
        } else if (magpie) {
          let moveType = MagpieMoveTypes.Play;
          let playedTiles = actualMove.word;
          if (actualMove.action === 'exchange') {
            if (actualMove.tiles?.length === 0) {
              playedTiles = [];
              moveType = MagpieMoveTypes.Pass;
            } else {
              playedTiles = actualMove.tiles;
              moveType = MagpieMoveTypes.Exchange;
            }
          }
          playedTiles = playedTiles?.map(wolgesLetterToLiwordsLetter) || [];

          const moveStr = await magpie.scorePlay(
            cgp,
            moveType,
            actualMove.down ? actualMove.idx : actualMove.lane,
            actualMove.down ? actualMove.lane : actualMove.idx,
            actualMove.down,
            playedTiles,
            computeLeaveML(playedTiles, rackNum)
          );
          console.log('scoreplay', cgp, 'ret', moveStr);
          const analyzerMove = analyzerMoveFromUCGIString(
            moveStr,
            dim,
            letters,
            rackNum,
            alphabet
          );
          const moveObj = {
            equity: analyzerMove.equity,
            action: analyzerMove.isExchange ? 'exchange' : 'play',
            valid: analyzerMove.valid,
            // doesn't matter what we put for tiles here, we're not using this value:
            tiles: new Array<number>(),
          };
          setEvaluatedMove({
            evaluatedMoveId: evaluatedMoveIdAtStart,
            moveObj: moveObj as JsonMove,
            analyzerMove: {
              ...analyzerMove,
              chosen: true,
            },
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
      console.log(
        'curMove',
        currentEvaluatedMove,
        'cachedMoves',
        JSON.stringify(cachedMoves)
      );
      for (const elt of cachedMoves) {
        if (!found) {
          console.log('trying to add', JSON.stringify(elt));
          if (currentEvaluatedMove.analyzerMove) {
            console.log('anmove', currentEvaluatedMove.analyzerMove);
            if (elt.jsonKey === currentEvaluatedMove.analyzerMove.jsonKey) {
              const combined = { ...currentEvaluatedMove.analyzerMove, ...elt };
              arr.push(combined);
              found = true;
              continue;
            }
          }
          if (currentEvaluatedMove.moveObj) {
            console.log('mobj', currentEvaluatedMove.moveObj);
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
          <td className="move-leave">
            {machineWordToRunes(m.leave, examinableGameContext.alphabet)}
          </td>
          <td className="move-equity">{(m.equity - equityBase).toFixed(2)}</td>
          <td className="move-winpct">
            {m.winpct !== undefined ? m.winpct.toFixed(2) + '%' : ''}
            {!(m.valid ?? true) && <React.Fragment>*</React.Fragment>}
          </td>
        </tr>
      )) ?? null,
    [equityBase, examinableGameContext.alphabet, moves, placeMove]
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

      <div className="analyzer-details">
        <Button className="analyzer-details-btn" type="link">
          Details
        </Button>
      </div>

      <div className="static-only">
        <p className="static-only-label">Rapid</p>
        <Switch
          checked={staticOnlyMode}
          onChange={toggleStaticOnlyMode}
          className="static-only-toggle"
          size="small"
        />
      </div>

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
          {nps !== 0 ? (
            <div>{`${(nps / 1000).toFixed(2)}k nodes/sec`}</div>
          ) : null}
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
