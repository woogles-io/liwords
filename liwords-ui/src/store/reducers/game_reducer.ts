import { Board } from '../../utils/cwgame/board';
import {
  PlayerInfo,
  GameEvent,
  PlayState,
  GameHistory,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Action, ActionType } from '../../actions/actions';
import {
  ServerGameplayEvent,
  GameHistoryRefresher,
  GameEndedEvent,
} from '../../gen/api/proto/realtime/realtime_pb';
import {
  Direction,
  isBlank,
  Blank,
  PlayedTiles,
} from '../../utils/cwgame/common';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';
import { PlayerOrder } from '../constants';
import { ClockController, Millis } from '../timer_controller';
import { ThroughTileMarker } from '../../utils/cwgame/game_event';

type TileDistribution = { [rune: string]: number };

export type RawPlayerInfo = {
  userID: string;
  score: number;
  // The time remaining is not going to go here. For efficiency,
  // we will put it in its own reducer.
  // timeMillis: number;
  onturn: boolean;
  currentRack: string;
};

const initialExpandToFull = (playerList: PlayerInfo[]): RawPlayerInfo[] => {
  return playerList.map((pi, idx) => {
    return {
      userID: pi.getUserId(),
      score: 0,
      onturn: idx === 0,
      currentRack: '',
      // timeMillis: 0,
    };
  });
};

export type GameState = {
  board: Board;
  // The initial tile distribution:
  tileDistribution: TileDistribution;
  // players are always in order of who went first:
  players: Array<RawPlayerInfo>;
  // The unseen tiles to the user (bag and opp's tiles)
  pool: TileDistribution;
  // Array of every turn taken so far.
  onturn: number; // index in players
  turns: Array<GameEvent>;
  gameID: string;
  lastPlayedTiles: PlayedTiles;
  nickToPlayerOrder: { [nick: string]: PlayerOrder };
  uidToPlayerOrder: { [uid: string]: PlayerOrder };
  playState: number;
  clockController: React.MutableRefObject<ClockController | null> | null;
  onClockTick: (p: PlayerOrder, t: Millis) => void;
  onClockTimeout: (p: PlayerOrder) => void;
};

export const startingGameState = (
  tileDistribution: TileDistribution,
  players: Array<RawPlayerInfo>,
  gameID: string
): GameState => {
  const gs = {
    board: new Board(),
    tileDistribution,
    pool: { ...tileDistribution },
    turns: new Array<GameEvent>(),
    players,
    onturn: 0,
    gameID,
    lastPlayedTiles: {},
    nickToPlayerOrder: {},
    uidToPlayerOrder: {},
    playState: PlayState.GAME_OVER,
    clockController: null,
    onClockTick: () => {},
    onClockTimeout: () => {},
  };
  return gs;
};

const onturnFromEvt = (state: GameState, evt: GameEvent) => {
  const po = state.nickToPlayerOrder[evt.getNickname()];
  let onturn;
  if (po === 'p0') {
    onturn = 0;
  } else if (po === 'p1') {
    onturn = 1;
  } else {
    throw new Error(
      `unexpected player order; nick:${evt.getNickname()}, ntpo:${JSON.stringify(
        state.nickToPlayerOrder
      )} `
    );
  }
  return onturn;
};

const clonePlayers = (players: Array<RawPlayerInfo>) => {
  const pclone = new Array<RawPlayerInfo>();
  players.forEach((p) => {
    pclone.push({ ...p });
  });
  return pclone;
};

const newGameState = (
  state: GameState,
  sge: ServerGameplayEvent
): GameState => {
  // Variables to pass down anew:
  let { board, lastPlayedTiles, pool } = state;
  const turns = [...state.turns];
  // let currentTurn;
  const evt = sge.getEvent()!;

  // Append the event.
  turns.push(GameEvent.deserializeBinary(evt.serializeBinary()));
  const players = clonePlayers(state.players);

  // onturn should be set to the player that came with the event.
  let onturn = onturnFromEvt(state, evt);
  switch (evt.getType()) {
    case GameEvent.Type.TILE_PLACEMENT_MOVE: {
      board = state.board.deepCopy();
      [lastPlayedTiles, pool] = placeOnBoard(board, pool, evt);
      break;
    }
    case GameEvent.Type.PHONY_TILES_RETURNED: {
      board = state.board.deepCopy();
      // Unplace the move BEFORE this one.
      const toUnplace = turns[turns.length - 2];
      pool = unplaceOnBoard(board, pool, toUnplace);
      // Set the user's rack back to what it used to be.
      players[onturn].currentRack = toUnplace.getRack();
      break;
    }
  }

  if (
    evt.getType() === GameEvent.Type.TILE_PLACEMENT_MOVE ||
    evt.getType() === GameEvent.Type.EXCHANGE
  ) {
    players[onturn].currentRack = sge.getNewRack();
  }

  players[onturn].score = evt.getCumulative();
  players[onturn].onturn = false;
  players[1 - onturn].onturn = true;
  onturn = 1 - onturn;
  return {
    // These never change:
    tileDistribution: state.tileDistribution,
    gameID: state.gameID,
    nickToPlayerOrder: state.nickToPlayerOrder,
    uidToPlayerOrder: state.uidToPlayerOrder,
    playState: sge.getPlaying(),
    clockController: state.clockController,
    onClockTick: state.onClockTick,
    onClockTimeout: state.onClockTimeout,
    // Potential changes:
    board,
    pool,
    turns,
    players,
    onturn,
    lastPlayedTiles,
  };
};

const placeOnBoard = (
  board: Board,
  pool: TileDistribution,
  evt: GameEvent
): [PlayedTiles, TileDistribution] => {
  const play = evt.getPlayedTiles();
  const playedTiles: PlayedTiles = {};
  const newPool = { ...pool };
  for (let i = 0; i < play.length; i += 1) {
    const rune = play[i];
    const row =
      evt.getDirection() === Direction.Vertical
        ? evt.getRow() + i
        : evt.getRow();
    const col =
      evt.getDirection() === Direction.Horizontal
        ? evt.getColumn() + i
        : evt.getColumn();
    const tile = { row, col, rune };
    if (rune !== ThroughTileMarker) {
      board.addTile(tile);
      if (isBlank(tile.rune)) {
        newPool[Blank] -= 1;
      } else {
        newPool[tile.rune] -= 1;
      }
      playedTiles[`R${row}C${col}`] = true;
    } // Otherwise, we played through a letter.
  }
  return [playedTiles, newPool];
};

const unplaceOnBoard = (
  board: Board,
  pool: TileDistribution,
  evt: GameEvent
): TileDistribution => {
  const play = evt.getPlayedTiles();
  const newPool = { ...pool };
  for (let i = 0; i < play.length; i += 1) {
    const rune = play[i];
    const row =
      evt.getDirection() === Direction.Vertical
        ? evt.getRow() + i
        : evt.getRow();
    const col =
      evt.getDirection() === Direction.Horizontal
        ? evt.getColumn() + i
        : evt.getColumn();
    const tile = { row, col, rune };
    if (rune !== ThroughTileMarker) {
      // Remove the tile from the board and place it back in the pool.
      board.removeTile(tile);
      if (isBlank(tile.rune)) {
        newPool[Blank] += 1;
      } else {
        newPool[tile.rune] += 1;
      }
    }
  }
  return newPool;
};

// pushTurns mutates the gs (GameState).
export const pushTurns = (gs: GameState, events: Array<GameEvent>) => {
  events.forEach((evt, idx) => {
    // We only care about placement and unplacement events here:
    switch (evt.getType()) {
      case GameEvent.Type.TILE_PLACEMENT_MOVE:
        // eslint-disable-next-line no-param-reassign
        [gs.lastPlayedTiles, gs.pool] = placeOnBoard(gs.board, gs.pool, evt);
        break;
      case GameEvent.Type.PHONY_TILES_RETURNED: {
        // Unplace the move BEFORE this one.
        const toUnplace = events[idx - 1];
        // eslint-disable-next-line no-param-reassign
        gs.pool = unplaceOnBoard(gs.board, gs.pool, toUnplace);
        // Set the user's rack back to what it used to be.
        break;
      }
    }

    // Push a deep clone of the turn.
    gs.turns.push(GameEvent.deserializeBinary(evt.serializeBinary()));
    // determine turn from event.
    const onturn = onturnFromEvt(gs, evt);
    // eslint-disable-next-line no-param-reassign
    gs.players[onturn].score = events[idx].getCumulative();
    // eslint-disable-next-line no-param-reassign
    gs.onturn = (onturn + 1) % 2;
  });
};

const stateFromHistory = (history: GameHistory): GameState => {
  // XXX: Do this for now. We will eventually want to put the tile
  // distribution itself in the history protobuf.
  // if (['NWL18', 'CSW19'].includes(history!.getLexicon())) {
  //   const dist = EnglishCrosswordGameDistribution;
  // }
  let playerList = history.getPlayersList();
  const flipPlayers = history.getSecondWentFirst();
  // If flipPlayers is on, we want to flip the players in the playerList.
  // The backend doesn't do this because of Reasons.
  if (flipPlayers) {
    playerList = [...playerList].reverse();
  }

  const nickToPlayerOrder = {
    [playerList[0].getNickname()]: 'p0' as PlayerOrder,
    [playerList[1].getNickname()]: 'p1' as PlayerOrder,
  };

  const uidToPlayerOrder = {
    [playerList[0].getUserId()]: 'p0' as PlayerOrder,
    [playerList[1].getUserId()]: 'p1' as PlayerOrder,
  };

  const gs = startingGameState(
    EnglishCrosswordGameDistribution,
    initialExpandToFull(playerList),
    history!.getUid()
  );
  gs.nickToPlayerOrder = nickToPlayerOrder;
  gs.uidToPlayerOrder = uidToPlayerOrder;
  pushTurns(gs, history.getEventsList());
  // racks are given in the original order that the playerList came in.
  // so if we reversed the player list, we must reverse the racks.
  let racks = history.getLastKnownRacksList();
  if (flipPlayers) {
    racks = [...racks].reverse();
    // timers = timers.reverse();
  }
  // Assign racks. Remember that the player listed first goes first.
  [gs.players[0].currentRack, gs.players[1].currentRack] = racks;
  // [gs.players[0].timeMillis, gs.players[1].timeMillis] = timers;
  gs.players[gs.onturn].onturn = true;
  gs.players[1 - gs.onturn].onturn = false;
  gs.playState = history.getPlayState();

  console.log('gs id', gs.gameID);
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
  const rem = sge.getTimeRemaining(); // time remaining for the player who just played
  const evt = sge.getEvent()!;
  const justPlayed = newState.nickToPlayerOrder[evt.getNickname()];
  let { p0, p1 } = newState.clockController.current.times;
  let activePlayer;
  let flipTimeRemaining = false;
  console.log('just played', justPlayed, evt.getNickname());
  console.log('player times are currently', p0, p1, 'from evt:', rem);

  if (
    evt.getType() === GameEvent.Type.CHALLENGE_BONUS ||
    evt.getType() === GameEvent.Type.PHONY_TILES_RETURNED
  ) {
    // For these particular two events, the time remaining is for the CHALLENGER.
    // Therefore, it's not the time remaining of the player whose nickname is
    // in the event, so we must flip the times here.
    flipTimeRemaining = true;
    console.log('flipTimeRemaining = true');
  }

  if (justPlayed === 'p0') {
    flipTimeRemaining ? (p1 = rem) : (p0 = rem);
    activePlayer = 'p1';
  } else if (justPlayed === 'p1') {
    flipTimeRemaining ? (p0 = rem) : (p1 = rem);
    activePlayer = 'p0';
  } else {
    throw new Error(`just played ${justPlayed} is unexpected`);
  }
  console.log('activePlayer is', activePlayer);

  newState.clockController.current.setClock(
    newState.playState,
    {
      p0,
      p1,
      activePlayer: activePlayer as PlayerOrder,
      lastUpdate: 0, // will get overwritten by setclock
    },
    0
  );
  // Send out a tick so the state updates right away (See store)
  newState.onClockTick(
    activePlayer as PlayerOrder,
    newState.clockController!.current.millisOf(activePlayer as PlayerOrder)
  );
};

const initializeTimerController = (
  state: GameState,
  newState: GameState,
  ghr: GameHistoryRefresher
) => {
  const history = ghr.getHistory()!;
  let [t1, t2] = [ghr.getTimePlayer1(), ghr.getTimePlayer2()];
  // Note that p0 is always first, even when "secondWentFirst", as p0 refers
  // to the order in the playerList, which always has the first player in that list
  // going first. (See flipPlayers in stateFromHistory)
  let onturn = 'p0' as PlayerOrder;
  if (history.getSecondWentFirst()) {
    [t1, t2] = [t2, t1];
  }

  // Note that p0 and p1 correspond to the new indices (after flipping first and second
  // players, if that happened)
  const evts = history.getEventsList();
  if (evts.length > 0) {
    // determine onturn from the last event.
    const lastWent = onturnFromEvt(newState, evts[evts.length - 1]);
    if (lastWent === 1) {
      onturn = 'p0' as PlayerOrder;
    } else if (lastWent === 0) {
      onturn = 'p1' as PlayerOrder;
    } else {
      throw new Error(`unexpected lastwent: ${lastWent}`);
    }
  }

  const clockState = {
    p0: t1,
    p1: t2,
    activePlayer: onturn,
    lastUpdate: 0,
  };

  if (newState.clockController!.current) {
    newState.clockController!.current.setClock(newState.playState, clockState);
  } else {
    // eslint-disable-next-line no-param-reassign
    newState.clockController!.current = new ClockController(
      clockState,
      state.onClockTimeout,
      state.onClockTick
    );
  }
  // And send out a tick right now.
  newState.onClockTick(
    onturn,
    newState.clockController!.current.millisOf(onturn)
  );
  newState.clockController!.current.setMaxOvertime(ghr.getMaxOvertimeMinutes());
};

// Here we are mixing declarative code with imperative code (needed for the timer).
// It is difficult and kind of messy. Hopefully it'll be the only place in the whole
// app where we do things like this.
export const GameReducer = (state: GameState, action: Action): GameState => {
  switch (action.actionType) {
    case ActionType.ClearHistory: {
      const gs = startingGameState(
        EnglishCrosswordGameDistribution,
        new Array<RawPlayerInfo>(),
        ''
      );
      gs.playState = PlayState.GAME_OVER;
      // Don't lose the clock controller, but pass it on until we get a
      // history refresher etc. Reset the shown time to 0.
      if (state.clockController !== null) {
        gs.clockController = state.clockController;
        if (gs.clockController!.current) {
          gs.clockController!.current.setClock(gs.playState, {
            p0: 0,
            p1: 0,
            lastUpdate: 0,
          });
        } else {
          gs.clockController!.current = new ClockController(
            { p0: 0, p1: 0, lastUpdate: 0 },
            state.onClockTimeout,
            state.onClockTick
          );
        }
      }
      return gs;
    }

    case ActionType.AddGameEvent: {
      // Check to make sure the game ID matches, and then hand off processing
      // to the newGameState function above.
      const sge = action.payload as ServerGameplayEvent;
      if (sge.getGameId() !== state.gameID) {
        return state; // no change
      }
      console.log('add game event', sge);
      const ngs = newGameState(state, sge);

      // Always pass the clock ref along. Begin imperative section:
      ngs.clockController = state.clockController;
      setClock(ngs, sge);
      return ngs;
    }

    case ActionType.RefreshHistory: {
      const ghr = action.payload as GameHistoryRefresher;
      const newState = stateFromHistory(ghr.getHistory()!);

      if (state.clockController !== null) {
        newState.clockController = state.clockController;
        initializeTimerController(state, newState, ghr);
      }
      // Otherwise if it is null, we have an issue, but there's no need to
      // throw an Error..
      return newState;
    }

    case ActionType.EndGame: {
      // If the game ends, we should set this in the store, if it hasn't
      // already been set. This can happen if it ends in an "abnormal" way
      // like a resignation or a timeout -- these aren't ServerGamePlayEvents per se.
      const gee = action.payload as GameEndedEvent;
      const newState = stateFromHistory(gee.getHistory()!);
      if (newState.clockController) {
        newState.clockController.current?.stopClock();
      }
      return newState;
    }
  }
  // This should never be reached, but the compiler is complaining.
  throw new Error(`Unhandled action type ${action.actionType}`);
};
