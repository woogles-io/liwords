import { Board } from "../../utils/cwgame/board";
import {
  PlayerInfo,
  GameEvent,
  PlayState,
  GameHistory,
  GameEvent_Type,
  GameEvent_Direction,
  GameEventSchema,
  PlayerInfoSchema,
} from "../../gen/api/vendor/macondo/macondo_pb";
import { Action, ActionType } from "../../actions/actions";
import {
  PlayedTiles,
  PlayerOfTiles,
  MachineLetter,
  isDesignatedBlankMachineLetter,
  MachineWord,
} from "../../utils/cwgame/common";
import { PlayerOrder } from "../constants";
import { ClockController, Millis } from "../timer_controller";
import {
  Alphabet,
  alphabetFromName,
  machineWordToRunes,
  runesToMachineWord,
  StandardEnglishAlphabet,
} from "../../constants/alphabets";
import {
  GameDocument,
  GameEndedEvent,
  GameHistoryRefresher,
  ServerGameplayEvent,
  GameEvent as OMGWordsGameEvent,
  GameEvent_Type as OMGWordsGameEventType,
  ServerOMGWordsEvent,
  GameDocumentSchema,
  ServerGameplayEventSchema,
} from "../../gen/api/proto/ipc/omgwords_pb";
import {
  CrosswordGameGridLayout,
  SuperCrosswordGameGridLayout,
} from "../../constants/board_layout";
import { clone, create } from "@bufbuild/protobuf";

export type TileDistribution = { [ml: MachineLetter]: number };

export type RawPlayerInfo = {
  userID: string;
  score: number;
  // The time remaining is not going to go here. For efficiency,
  // we will put it in its own reducer.
  // timeMillis: number;
  onturn: boolean;
  currentRack: MachineWord;
};

const initialExpandToFull = (playerList: PlayerInfo[]): RawPlayerInfo[] => {
  return playerList.map((pi, idx) => {
    return {
      userID: pi.userId,
      score: 0,
      onturn: idx === 0,
      currentRack: new Array<MachineLetter>(),
      // timeMillis: 0,
    };
  });
};

export type GameState = {
  board: Board;
  // The initial tile distribution:
  alphabet: Alphabet;
  // players are always in order of who went first:
  players: Array<RawPlayerInfo>;
  // The unseen tiles to the user (bag and opp's tiles)
  pool: TileDistribution;
  // Array of every turn taken so far.
  onturn: number; // index in players
  turns: Array<GameEvent>;
  gameID: string;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles; // not cleaned up after challenges
  nickToPlayerOrder: { [nick: string]: PlayerOrder };
  uidToPlayerOrder: { [uid: string]: PlayerOrder };
  playState: number;
  clockController: React.MutableRefObject<ClockController | null> | null;
  gameDocument: GameDocument;
  onClockTick: (p: PlayerOrder, t: Millis) => void;
  onClockTimeout: (p: PlayerOrder) => void;
  refresherCount?: number; // Track number of GameRefreshers received
  // Raw timer values from GameHistoryRefresher (for correspondence games)
  timePlayer1?: number;
  timePlayer2?: number;
};

const makePool = (alphabet: Alphabet): TileDistribution => {
  const td: TileDistribution = {};
  alphabet.letters.forEach((l, idx) => {
    td[idx] = l.count;
  });
  return td;
};

export const startingGameState = (
  alphabet: Alphabet,
  players: Array<RawPlayerInfo>,
  gameID: string,
  layout = CrosswordGameGridLayout,
): GameState => {
  const gs = {
    board: new Board(layout),
    alphabet,
    pool: makePool(alphabet),
    turns: new Array<GameEvent>(),
    players,
    onturn: 0,
    gameID,
    lastPlayedTiles: {},
    playerOfTileAt: {},
    nickToPlayerOrder: {},
    uidToPlayerOrder: {},
    playState: PlayState.GAME_OVER,
    clockController: null,
    onClockTick: () => {},
    onClockTimeout: () => {},
    gameDocument: create(GameDocumentSchema, {}),
  };
  return gs;
};

const clonePlayers = (players: Array<RawPlayerInfo>) => {
  const pclone = new Array<RawPlayerInfo>();
  players.forEach((p) => {
    pclone.push({ ...p });
  });
  return pclone;
};

const newGameStateFromGameplayEvent = (
  state: GameState,
  sge: ServerGameplayEvent,
): GameState => {
  // Variables to pass down anew:
  let { board, lastPlayedTiles, playerOfTileAt, pool } = state;
  const turns = [...state.turns];
  // let currentTurn;
  const evt = sge.event;
  if (!evt) {
    throw new Error("missing event");
  }
  // Append the event.

  turns.push(clone(GameEventSchema, evt));
  const players = clonePlayers(state.players);

  // onturn should be set to the player that came with the event.
  let onturn = evt.playerIndex;
  switch (evt.type) {
    case GameEvent_Type.TILE_PLACEMENT_MOVE: {
      board = state.board.deepCopy();
      [lastPlayedTiles, pool] = placeOnBoard(board, pool, evt, state.alphabet);
      playerOfTileAt = { ...playerOfTileAt };
      for (const k in lastPlayedTiles) {
        playerOfTileAt[k] = onturn;
      }
      break;
    }
    case GameEvent_Type.PHONY_TILES_RETURNED: {
      board = state.board.deepCopy();
      // Unplace the move BEFORE this one.
      const toUnplace = turns[turns.length - 2];
      pool = unplaceOnBoard(board, pool, toUnplace, state.alphabet);
      // Set the user's rack back to what it used to be.
      players[onturn].currentRack = runesToMachineWord(
        toUnplace.rack,
        state.alphabet,
      );
      break;
    }
  }

  if (
    evt.type === GameEvent_Type.TILE_PLACEMENT_MOVE ||
    evt.type === GameEvent_Type.EXCHANGE
  ) {
    players[onturn].currentRack = runesToMachineWord(
      sge.newRack,
      state.alphabet,
    );
  }

  players[onturn].score = evt.cumulative;
  players[onturn].onturn = false;
  players[1 - onturn].onturn = true;
  onturn = 1 - onturn;
  return {
    // These never change:
    alphabet: state.alphabet,
    gameID: state.gameID,
    nickToPlayerOrder: state.nickToPlayerOrder,
    uidToPlayerOrder: state.uidToPlayerOrder,
    playState: sge.playing,
    clockController: state.clockController,
    onClockTick: state.onClockTick,
    onClockTimeout: state.onClockTimeout,
    gameDocument: state.gameDocument,
    // Potential changes:
    board,
    pool,
    turns,
    players,
    onturn,
    lastPlayedTiles,
    playerOfTileAt,
  };
};

const placeOnBoard = (
  board: Board,
  pool: TileDistribution,
  evt: GameEvent,
  alphabet: Alphabet,
): [PlayedTiles, TileDistribution] => {
  const play = evt.playedTiles;
  const playedTiles: PlayedTiles = {};
  const newPool = { ...pool };

  const mls = runesToMachineWord(play, alphabet);
  for (let i = 0; i < mls.length; i++) {
    const ml = mls[i];
    const row =
      evt.direction === GameEvent_Direction.VERTICAL ? evt.row + i : evt.row;
    const col =
      evt.direction === GameEvent_Direction.HORIZONTAL
        ? evt.column + i
        : evt.column;
    const tile = { row, col, ml };
    if (ml !== 0 && board.letterAt(row, col) === 0) {
      board.addTile(tile);
      if (isDesignatedBlankMachineLetter(tile.ml)) {
        newPool[0] -= 1;
      } else {
        newPool[tile.ml] -= 1;
      }
      playedTiles[`R${row}C${col}`] = true;
    } // Otherwise, we played through a letter.
  }
  return [playedTiles, newPool];
};

const unplaceOnBoard = (
  board: Board,
  pool: TileDistribution,
  evt: GameEvent,
  alphabet: Alphabet,
): TileDistribution => {
  const play = evt.playedTiles;
  const newPool = { ...pool };
  const mls = runesToMachineWord(play, alphabet);
  for (let i = 0; i < mls.length; i++) {
    const ml = mls[i];
    const row =
      evt.direction === GameEvent_Direction.VERTICAL ? evt.row + i : evt.row;
    const col =
      evt.direction === GameEvent_Direction.HORIZONTAL
        ? evt.column + i
        : evt.column;
    const tile = { row, col, ml };
    if (ml !== 0 && board.letterAt(row, col) !== 0) {
      // Remove the tile from the board and place it back in the pool.
      board.removeTile(tile);
      if (isDesignatedBlankMachineLetter(tile.ml)) {
        newPool[0] += 1;
      } else {
        newPool[tile.ml] += 1;
      }
    }
  }
  return newPool;
};

// convert to a Macondo GameEvent. This function should be obsoleted eventually,
// but it will be a pain. We need a GameEvent that contains user-visible rack info.
const convertToGameEvt = (
  evt: OMGWordsGameEvent | undefined,
  alphabet: Alphabet,
): GameEvent => {
  if (!evt) {
    return create(GameEventSchema, {});
  }
  return create(GameEventSchema, {
    rack: machineWordToRunes(Array.from(evt.rack), alphabet),
    type: evt.type.valueOf(),
    cumulative: evt.cumulative,
    row: evt.row,
    column: evt.column,
    direction: evt.direction,
    position: evt.position,
    playedTiles: machineWordToRunes(
      Array.from(evt.playedTiles),
      alphabet,
      true,
    ),
    exchanged: machineWordToRunes(Array.from(evt.exchanged), alphabet),
    score: evt.score,
    bonus: evt.bonus,
    endRackPoints: evt.endRackPoints,
    lostScore: evt.lostScore,
    isBingo: evt.isBingo,
    wordsFormed: evt.wordsFormed.map((v) =>
      machineWordToRunes(Array.from(v), alphabet),
    ),
    millisRemaining: evt.millisRemaining,
    playerIndex: evt.playerIndex,
  });
};

// convert to a legacy ServerGameplayEvent. We will eventually remove this
// and have a single server event type.
const convertToServerGameplayEvent = (
  evt: ServerOMGWordsEvent,
  alphabet: Alphabet,
): ServerGameplayEvent => {
  return create(ServerGameplayEventSchema, {
    event: convertToGameEvt(evt.event, alphabet),
    gameId: evt.gameId,
    newRack: machineWordToRunes(Array.from(evt.newRack), alphabet),
    timeRemaining: evt.timeRemaining,
    playing: evt.playing.valueOf(),
    userId: evt.userId,
  });
};

// pushTurns mutates the gs (GameState).
export const pushTurns = (gs: GameState, events: Array<GameEvent>) => {
  events.forEach((evt, idx) => {
    // determine turn from event.
    const onturn = evt.playerIndex;
    // We only care about placement and unplacement events here:
    switch (evt.type) {
      case GameEvent_Type.TILE_PLACEMENT_MOVE:
        [gs.lastPlayedTiles, gs.pool] = placeOnBoard(
          gs.board,
          gs.pool,
          evt,
          gs.alphabet,
        );
        for (const k in gs.lastPlayedTiles) {
          gs.playerOfTileAt[k] = onturn;
        }
        break;
      case GameEvent_Type.PHONY_TILES_RETURNED: {
        // Unplace the move BEFORE this one.
        const toUnplace = events[idx - 1];
        gs.pool = unplaceOnBoard(gs.board, gs.pool, toUnplace, gs.alphabet);
        // Set the user's rack back to what it used to be.
        break;
      }
    }

    // Push a deep clone of the turn.
    gs.turns.push(clone(GameEventSchema, evt));
    gs.players[onturn].score = events[idx].cumulative;
    gs.onturn = (onturn + 1) % 2;
  });
};

// pushTurnsNew mutates the gs (GameState). This will supplant pushTurns
// when we move over fully to GameDocument.
const pushTurnsNew = (gs: GameState, events: Array<OMGWordsGameEvent>) => {
  events.forEach((oevt, idx) => {
    const evt = convertToGameEvt(oevt, gs.alphabet);

    const onturn = evt.playerIndex;
    // We only care about placement and unplacement events here:
    switch (evt.type) {
      case OMGWordsGameEventType.TILE_PLACEMENT_MOVE:
        [gs.lastPlayedTiles, gs.pool] = placeOnBoard(
          gs.board,
          gs.pool,
          evt,
          gs.alphabet,
        );
        for (const k in gs.lastPlayedTiles) {
          gs.playerOfTileAt[k] = onturn;
        }
        break;
      case OMGWordsGameEventType.PHONY_TILES_RETURNED: {
        // Unplace the move BEFORE this one.
        const toUnplace = convertToGameEvt(events[idx - 1], gs.alphabet);
        gs.pool = unplaceOnBoard(gs.board, gs.pool, toUnplace, gs.alphabet);
        // Set the user's rack back to what it used to be.
        break;
      }
    }
    gs.turns.push(evt);
    gs.players[onturn].score = events[idx].cumulative;
    gs.onturn = (onturn + 1) % 2;
  });
};

const stateFromHistory = (history: GameHistory): GameState => {
  const playerList = history.players;

  const nickToPlayerOrder = {
    [playerList[0].nickname]: "p0" as PlayerOrder,
    [playerList[1].nickname]: "p1" as PlayerOrder,
  };

  const uidToPlayerOrder = {
    [playerList[0].userId]: "p0" as PlayerOrder,
    [playerList[1].userId]: "p1" as PlayerOrder,
  };

  const alphabet = alphabetFromName(history.letterDistribution.toLowerCase());

  const gs = startingGameState(
    alphabet,
    initialExpandToFull(playerList),
    history.uid,
    history.boardLayout === "SuperCrosswordGame"
      ? SuperCrosswordGameGridLayout
      : CrosswordGameGridLayout,
  );
  gs.nickToPlayerOrder = nickToPlayerOrder;
  gs.uidToPlayerOrder = uidToPlayerOrder;
  pushTurns(gs, history.events);
  const racks = history.lastKnownRacks;

  // Assign racks. Remember that the player listed first goes first.
  for (let i = 0; i < gs.players.length; i++) {
    gs.players[i].currentRack = runesToMachineWord(racks[i], alphabet);
  }
  // [gs.players[0].timeMillis, gs.players[1].timeMillis] = timers;
  gs.players[gs.onturn].onturn = true;
  gs.players[1 - gs.onturn].onturn = false;
  gs.playState = history.playState;
  return gs;
};

const stateFromDocument = (gdoc: GameDocument): GameState => {
  const playerList = gdoc.players;
  // Convert from a gdoc's playerList to a macondo playerList
  const compatiblePlayerList = playerList.map((pinfo) => {
    const { $typeName: _, ...properties } = pinfo;
    void _;
    const mcpinfo = create(PlayerInfoSchema, properties);
    return mcpinfo;
  });

  const nickToPlayerOrder = {
    [playerList[0].nickname]: "p0" as PlayerOrder,
    [playerList[1].nickname]: "p1" as PlayerOrder,
  };
  const uidToPlayerOrder = {
    [playerList[0].userId]: "p0" as PlayerOrder,
    [playerList[1].userId]: "p1" as PlayerOrder,
  };

  const alphabet = alphabetFromName(gdoc.letterDistribution.toLowerCase());
  const gs = startingGameState(
    alphabet,
    initialExpandToFull(compatiblePlayerList),
    gdoc.uid,
    gdoc.boardLayout === "SuperCrosswordGame"
      ? SuperCrosswordGameGridLayout
      : CrosswordGameGridLayout,
  );
  gs.nickToPlayerOrder = nickToPlayerOrder;
  gs.uidToPlayerOrder = uidToPlayerOrder;
  pushTurnsNew(gs, gdoc.events);
  const racks = gdoc.racks;

  racks.forEach((rack, idx) => {
    gs.players[idx].currentRack = Array.from(rack);
  });
  gs.players[gdoc.playerOnTurn].onturn = true;
  gs.players[1 - gdoc.playerOnTurn].onturn = false;
  gs.playState = gdoc.playState;
  gs.gameDocument = gdoc;
  return gs;
};

const setClock = (newState: GameState, sge: ServerGameplayEvent) => {
  if (!newState.clockController) {
    return;
  }
  if (!newState.clockController.current) {
    return;
  }
  // If either of the above happened, we have an issue. But these should only
  // happen in some tests.
  // Set the clock
  const rem = sge.timeRemaining; // time remaining for the player who just played
  const timeBank = sge.timeBank; // time bank for the player who just played
  const evt = sge.event;
  if (!evt) {
    throw new Error("missing event in setclock");
  }

  const justPlayed = evt.playerIndex;

  let { p0, p1, p0TimeBank, p1TimeBank } =
    newState.clockController.current.times;
  let activePlayer;
  let flipTimeRemaining = false;

  if (
    evt.type === GameEvent_Type.CHALLENGE_BONUS ||
    evt.type === GameEvent_Type.PHONY_TILES_RETURNED
  ) {
    // For these particular two events, the time remaining is for the CHALLENGER.
    // Therefore, it's not the time remaining of the player whose nickname is
    // in the event, so we must flip the times here.
    flipTimeRemaining = true;
    console.log("flipTimeRemaining = true");
  }

  if (justPlayed === 0) {
    if (flipTimeRemaining) {
      p1 = rem;
      // Time bank also flipped - it's for the challenger (p1)
      if (timeBank !== undefined) {
        p1TimeBank = timeBank > 0 ? timeBank : undefined;
      }
    } else {
      p0 = rem;
      // Normal case - time bank for player who moved (p0)
      if (timeBank !== undefined) {
        p0TimeBank = timeBank > 0 ? timeBank : undefined;
      }
    }
    activePlayer = "p1";
  } else if (justPlayed === 1) {
    if (flipTimeRemaining) {
      p0 = rem;
      // Time bank also flipped - it's for the challenger (p0)
      if (timeBank !== undefined) {
        p0TimeBank = timeBank > 0 ? timeBank : undefined;
      }
    } else {
      p1 = rem;
      // Normal case - time bank for player who moved (p1)
      if (timeBank !== undefined) {
        p1TimeBank = timeBank > 0 ? timeBank : undefined;
      }
    }
    activePlayer = "p0";
  } else {
    throw new Error(`just played ${justPlayed} is unexpected`);
  }
  newState.clockController.current.setClock(
    newState.playState,
    {
      p0,
      p1,
      p0TimeBank,
      p1TimeBank,
      activePlayer: activePlayer as PlayerOrder,
      lastUpdate: 0, // will get overwritten by setclock
    },
    0,
  );
  // Send out a tick so the state updates right away (See store)
  newState.onClockTick(
    activePlayer as PlayerOrder,
    newState.clockController.current.millisOf(activePlayer as PlayerOrder),
  );
};

const initializeTimerController = (
  state: GameState,
  newState: GameState,
  ghr: GameHistoryRefresher,
) => {
  const history = ghr.history;
  if (!history) {
    throw new Error("missing history in initialize");
  }
  if (!newState.clockController) {
    return;
  }

  const [t1, t2] = [ghr.timePlayer1, ghr.timePlayer2];
  const [tb1, tb2] = [ghr.timeBankPlayer1, ghr.timeBankPlayer2];

  let onturn = "p0" as PlayerOrder;

  // Note that p0 and p1 correspond to the new indices
  const evts = history.events;
  if (evts.length > 0) {
    // determine onturn from the last event.
    const lastWent = evts[evts.length - 1].playerIndex;
    if (lastWent === 1) {
      onturn = "p0" as PlayerOrder;
    } else if (lastWent === 0) {
      onturn = "p1" as PlayerOrder;
    } else {
      throw new Error(`unexpected lastwent: ${lastWent}`);
    }
  }

  const clockState = {
    p0: t1,
    p1: t2,
    p0TimeBank: tb1 > 0 ? tb1 : undefined,
    p1TimeBank: tb2 > 0 ? tb2 : undefined,
    activePlayer: onturn,
    lastUpdate: 0,
  };

  if (newState.clockController.current) {
    newState.clockController.current.setClock(newState.playState, clockState);
  } else {
    newState.clockController.current = new ClockController(
      clockState,
      state.onClockTimeout,
      state.onClockTick,
    );
  }
  // And send out a tick right now.
  newState.onClockTick(
    onturn,
    newState.clockController.current.millisOf(onturn),
  );
  newState.clockController.current.setMaxOvertime(ghr.maxOvertimeMinutes);
};

// Here we are mixing declarative code with imperative code (needed for the timer).
// It is difficult and kind of messy. Hopefully it'll be the only place in the whole
// app where we do things like this.
export const GameReducer = (state: GameState, action: Action): GameState => {
  switch (action.actionType) {
    case ActionType.ClearHistory: {
      const gs = startingGameState(
        StandardEnglishAlphabet,
        new Array<RawPlayerInfo>(),
        "",
      );
      gs.playState = PlayState.GAME_OVER;
      // Don't lose the clock controller, but pass it on until we get a
      // history refresher etc. Reset the shown time to 0.
      const cmd = action.payload as string;
      if (cmd !== "noclock") {
        if (state.clockController !== null) {
          gs.clockController = state.clockController;
          if (gs.clockController.current) {
            gs.clockController.current.setClock(gs.playState, {
              p0: 0,
              p1: 0,
              lastUpdate: 0,
            });
          } else {
            gs.clockController.current = new ClockController(
              { p0: 0, p1: 0, lastUpdate: 0 },
              state.onClockTimeout,
              state.onClockTick,
            );
          }
        }
      }
      return gs;
    }

    case ActionType.AddGameEvent: {
      // Check to make sure the game ID matches, and then hand off processing
      // to the newGameState function above.
      const sge = action.payload as ServerGameplayEvent;
      if (sge.gameId !== state.gameID) {
        return state; // no change
      }
      const ngs = newGameStateFromGameplayEvent(state, sge);

      // Always pass the clock ref along. Begin imperative section:
      ngs.clockController = state.clockController;
      setClock(ngs, sge);
      return ngs;
    }

    case ActionType.AddOMGWordsEvent: {
      const evt = action.payload as ServerOMGWordsEvent;
      if (evt.gameId !== state.gameID) {
        return state;
      }
      if (!evt.event) {
        return state;
      }
      const sge = convertToServerGameplayEvent(evt, state.alphabet);
      const ngs = newGameStateFromGameplayEvent(state, sge);
      ngs.clockController = state.clockController;
      setClock(ngs, sge);
      return ngs;
    }

    case ActionType.RefreshHistory: {
      const ghr = action.payload as GameHistoryRefresher;
      const history = ghr.history;
      if (!history) {
        throw new Error("missing history in refresh");
      }

      // If we've already seen a refresher AND this one has no events,
      // ignore it as it's likely a duplicate from InitRealmInfo
      if ((state.refresherCount || 0) > 0 && history.events.length === 0) {
        console.log(
          "Ignoring subsequent empty GameRefresher - already processed",
          state.refresherCount,
          "refreshers",
        );
        return state;
      }

      const newState = stateFromHistory(history);

      // Increment the refresher count
      newState.refresherCount = (state.refresherCount || 0) + 1;

      // Store raw timer values from GameHistoryRefresher
      newState.timePlayer1 = ghr.timePlayer1;
      newState.timePlayer2 = ghr.timePlayer2;

      if (state.clockController !== null) {
        newState.clockController = state.clockController;
        initializeTimerController(state, newState, ghr);
      }
      // Otherwise if it is null, we have an issue, but there's no need to
      // throw an Error..
      return newState;
    }

    case ActionType.InitFromDocument: {
      const d = action.payload as GameDocument;
      const newState = stateFromDocument(d);
      // TODO: once we start moving all games to GameDocument, handle timer
      // here.
      // if (state.clockController !== null) {
      //   newState.clockController = state.clockController;
      //   initializeTimerController(state, newState, ghr);
      // }
      // stateFromDocument should initialize the clock controller as well.
      // Otherwise if it is null, we have an issue, but there's no need to
      // throw an Error..
      return newState;
    }

    // A GameHistory that represents a static position.
    case ActionType.SetupStaticPosition: {
      const h = action.payload as GameHistory;
      const newState = stateFromHistory(h);

      return newState;
    }

    case ActionType.EndGame: {
      // If the game ends, we should set this in the store, if it hasn't
      // already been set. This can happen if it ends in an "abnormal" way
      // like a resignation or a timeout -- these aren't ServerGamePlayEvents per se.
      const gee = action.payload as GameEndedEvent;
      const history = gee.history;
      if (!history) {
        throw new Error("missing history in end game event");
      }
      const newState = stateFromHistory(history);
      if (newState.clockController) {
        newState.clockController.current?.stopClock();
      }
      return newState;
    }

    /*
    // Should these actions maybe be in their own reducer?
    case ActionType.ReloadComments: {
      const comments = action.payload as Array<GameComment>;
      // assume comments are already sorted chronologically.
      const newState = {
        ...state,
        ...comments,
      };
      return newState;
    }

    case ActionType.AddComment: {
      const comment = action.payload as GameComment;
      // Since comments are already sorted chronologically, this
      // one should always be the newest.
      const newState = {
        ...state,
        comments: [...state.comments, comment],
      };
      return newState;
    }

    case ActionType.DeleteComment: {
      const deletedID = action.payload as string;
      const newState = {
        ...state,
        comments: state.comments.filter(
          (comment) => comment.commentId !== deletedID
        ),
      };
      return newState;
    }

    case ActionType.EditComment: {
      const editedComment = action.payload as GameComment;
      const newState = {
        ...state,
        comments: state.comments.map((cmt) => {
          if (cmt.commentId === editedComment.commentId) {
            return editedComment.clone();
          }
          return cmt.clone();
        }),
      };
      return newState;
    }
    */
  }
  // This should never be reached, but the compiler is complaining.
  throw new Error(`Unhandled action type ${action.actionType}`);
};
