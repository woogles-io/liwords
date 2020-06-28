import { Board, Tile } from '../../utils/cwgame/board';
import {
  PlayerInfo,
  GameTurn,
  GameEvent,
  PlayState,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Action, ActionType } from '../../actions/actions';
import {
  ServerGameplayEvent,
  GameHistoryRefresher,
} from '../../gen/api/proto/realtime_pb';
import { Direction, isBlank, Blank } from '../../utils/cwgame/common';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';
import { PlayerOrder } from '../constants';
import { ClockController, Millis } from '../timer_controller';
import { ThroughTileMarker } from '../../utils/cwgame/game_event';

type TileDistribution = { [rune: string]: number };

export type FullPlayerInfo = {
  userID: string;
  nickname: string;
  fullName: string;
  countryFlag: string;
  rating: number; // in the relevant lexicon / game mode
  title: string;
  score: number;
  // The time remaining is not going to go here. For efficiency,
  // we will put it in its own reducer.
  // timeMillis: number;
  onturn: boolean;
  avatarUrl: string;
  currentRack: string;
};

export type ReducedPlayerInfo = {
  nickname: string;
  fullName: string;
  avatarUrl: string;
};

const initialExpandToFull = (playerList: PlayerInfo[]): FullPlayerInfo[] => {
  return playerList.map((pi, idx) => {
    return {
      userID: pi.getUserId(),
      nickname: pi.getNickname(),
      fullName: pi.getRealName(),
      countryFlag: '',
      rating: 0,
      title: '',
      score: 0,
      onturn: idx === 0,
      avatarUrl: '',
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
  players: Array<FullPlayerInfo>;
  // The unseen tiles to the user (bag and opp's tiles)
  pool: TileDistribution;
  // Array of every turn taken so far.
  turns: Array<GameTurn>;
  onturn: number; // index in players
  currentTurn: GameTurn;
  lastEvent: GameEvent | null;
  gameID: string;
  lastPlayedTiles: Array<Tile>;
  nickToPlayerOrder: { [nick: string]: PlayerOrder };
  uidToPlayerOrder: { [uid: string]: PlayerOrder };
  playState: number;
  clockController: React.MutableRefObject<ClockController | null> | null;
  onClockTick: (p: PlayerOrder, t: Millis) => void;
  onClockTimeout: (p: PlayerOrder) => void;
};

export const startingGameState = (
  tileDistribution: TileDistribution,
  players: Array<FullPlayerInfo>,
  gameID: string
): GameState => {
  const gs = {
    board: new Board(),
    tileDistribution,
    pool: { ...tileDistribution },
    turns: new Array<GameTurn>(),
    players,
    onturn: 0,
    currentTurn: new GameTurn(),
    lastEvent: null,
    gameID,
    lastPlayedTiles: new Array<Tile>(),
    nickToPlayerOrder: {},
    uidToPlayerOrder: {},
    playState: PlayState.PLAYING,
    clockController: null,
    onClockTick: () => {},
    onClockTimeout: () => {},
  };
  return gs;
};

/** newStateAfterTurns creates a new state from an old state and a "RefreshTurns"
 * game event.
 * A RefreshTurns event ALWAYS gets sent immediately after a challenge. As such,
 * these are the possibilities for what it contains:
 * If it was a valid challenge (word is phony):
 *    Contains one turn with two events: a tile placement and a tile "take back"
 *      - This overwrites the last turn, which would have just had the tile placement
 *      - The timestamp of the "take back" event is the time left for the CHALLENGER.
 *    Potentially contains an additional 2 "turns" with end game rack penalties,
 *      if the game ended because of six zeroes.
 * If it was an invalid challenge (word is good):
 *    Last turn gets overwritten no matter what, with a tile placement + a bonus score event,
 *      or just with a tile placement (if this is double or single challenge).
 *    Next turn is a whole new turn if this is double challenge (essentially a pass), or nothing
 *     if this is any other type of challenge. However, if this ends the game, there
 *     is no additional "pass turn" if double challenge.
 *    Potentially contains end game rack bonuses, if this was a challenged out-play.
 *    Or potentially contains end game rack penalties, if this was 6 consecutive zeroes
 *      (extremely rare, would likely be a contrived game with a zero-scoring last play)
 *
 *    The "Time Remaining" of the challenge event is the CHALLENGER's time left.
 */
// XXX: We are not doing this now, it's too complicated. Let's just refresh from
// GameHistory.

// const newStateAfterTurns = (
//   state: GameState,
//   gtr: GameTurnsRefresher
// ): GameState => {
//   const { lastPlayedTiles, onturn } = state;
//   const incomingTurns = gtr.getTurnsList();
//   const startingTurn = gtr.getStartingTurn();
//   const turnsCopy = [...state.turns];
//   turnsCopy.splice(
//     startingTurn,
//     turnsCopy.length - startingTurn,
//     ...incomingTurns
//   );

//   // Reset the board back to blank and place tiles on it.
//   const gs = startingGameState(
//     state.tileDistribution,
//     [...state.players],
//     state.gameID
//   );
//   pushTurns(gs, turnsCopy);
// };

const newGameState = (
  state: GameState,
  sge: ServerGameplayEvent
): GameState => {
  // Variables to pass down anew:
  let { turns, board, lastPlayedTiles, pool, onturn } = state;
  let currentTurn;
  const evt = sge.getEvent()!;
  if (
    state.lastEvent !== null &&
    evt.getNickname() !== state.lastEvent.getNickname()
  ) {
    // Create a new turn
    turns = [...state.turns, state.currentTurn];
    currentTurn = new GameTurn();
  } else {
    // Clone the pb msg. Do this since `cloneMessage` is not yet implemented
    // in typescript module.
    currentTurn = GameTurn.deserializeBinary(
      state.currentTurn.serializeBinary()
    );
  }
  // Append the event.
  currentTurn.addEvents(evt);
  switch (evt.getType()) {
    case GameEvent.Type.TILE_PLACEMENT_MOVE: {
      // Right now, this is the ONLY case in which we modify the board state.
      // the PHONY_TILES_RETURNED event is not handled; we instead refresh
      // the game from a GameTurnRefresher whenever there is a challenge event
      // placeOnBoard(evt) -- make board clone!
      board = state.board.deepCopy();
      [lastPlayedTiles, pool] = placeOnBoard(board, pool, evt);
    }
  }
  const players = [...state.players];

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
  console.log('but now it changes to', onturn);
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
    currentTurn,
    lastEvent: evt,
    lastPlayedTiles,
  };
};

const placeOnBoard = (
  board: Board,
  pool: TileDistribution,
  evt: GameEvent
): [Array<Tile>, TileDistribution] => {
  const play = evt.getPlayedTiles();
  const playedTiles = [];
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
      playedTiles.push(tile);
    } // Otherwise, we played through a letter.
  }
  return [playedTiles, newPool];
};

// pushTurns mutates the gs (GameState).
const pushTurns = (gs: GameState, turns: Array<GameTurn>) => {
  turns.forEach((turn, idx) => {
    const events = turn.getEventsList();
    // Detect challenged-off moves:
    let challengedOff = false;
    if (events.length === 2) {
      if (
        events[0].getType() === GameEvent.Type.TILE_PLACEMENT_MOVE &&
        events[1].getType() === GameEvent.Type.PHONY_TILES_RETURNED
      ) {
        challengedOff = true;
      }
    }
    if (!challengedOff) {
      events.forEach((evt) => {
        switch (evt.getType()) {
          case GameEvent.Type.TILE_PLACEMENT_MOVE:
            // eslint-disable-next-line no-param-reassign
            [gs.lastPlayedTiles, gs.pool] = placeOnBoard(
              gs.board,
              gs.pool,
              evt
            );
            break;
          default:
          //  do nothing - we only care about tile placement moves here.
        }
      });
    }
    // Push a deep clone of the turn.
    gs.turns.push(GameTurn.deserializeBinary(turn.serializeBinary()));
    // eslint-disable-next-line no-param-reassign
    gs.players[gs.onturn].score = events[events.length - 1].getCumulative();
    // eslint-disable-next-line no-param-reassign
    gs.onturn = (idx + 1) % 2;
  });
};

const stateFromHistory = (refresher: GameHistoryRefresher): GameState => {
  // XXX: Do this for now. We will eventually want to put the tile
  // distribution itself in the history protobuf.
  // if (['NWL18', 'CSW19'].includes(history!.getLexicon())) {
  //   const dist = EnglishCrosswordGameDistribution;
  // }
  const history = refresher.getHistory()!;
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
  pushTurns(gs, history.getTurnsList());

  // racks are given in the original order that the playerList came in.
  // so if we reversed the player list, we must reverse the racks.
  let racks = history.getLastKnownRacksList();
  // let timers = [refresher.getTimePlayer1(), refresher.getTimePlayer2()];
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
  return gs;
};

const setClock = (
  state: GameState,
  newState: GameState,
  sge: ServerGameplayEvent
) => {
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
  if (justPlayed === 'p0') {
    p0 = rem;
    activePlayer = 'p1';
  } else if (justPlayed === 'p1') {
    p1 = rem;
    activePlayer = 'p0';
  } else {
    throw new Error(`just played ${justPlayed} is unexpected`);
  }
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
  if (history.getSecondWentFirst()) {
    [t1, t2] = [t2, t1];
  }
  // Note that p0 and p1 correspond to the new indices (after flipping first and second
  // players, if that happened)
  const onturn = (history.getTurnsList().length % 2 === 0
    ? 'p0'
    : 'p1') as PlayerOrder;
  const clockState = {
    p0: t1,
    p1: t2,
    activePlayer: onturn,
    lastUpdate: 0,
  };

  console.log(
    'clockState will be set',
    clockState,
    history.getTurnsList().length
  );

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
};

// Here we are mixing declarative code with imperative code (needed for the timer).
// It is difficult and kind of messy. Hopefully it'll be the only place in the whole
// app where we do things like this.
export const GameReducer = (state: GameState, action: Action): GameState => {
  switch (action.actionType) {
    case ActionType.AddGameEvent: {
      // Check to make sure the game ID matches, and then hand off processing
      // to the newGameState function above.
      const sge = action.payload as ServerGameplayEvent;
      if (sge.getGameId() !== state.gameID) {
        return state; // no change
      }
      const ngs = newGameState(state, sge);

      console.log('new game state', ngs);
      // Always pass the clock ref along. Begin imperative section:
      ngs.clockController = state.clockController;
      setClock(state, ngs, sge);
      return ngs;
    }

    case ActionType.RefreshHistory: {
      const ghr = action.payload as GameHistoryRefresher;
      const newState = stateFromHistory(ghr);

      if (state.clockController !== null) {
        newState.clockController = state.clockController;
        initializeTimerController(state, newState, ghr);
      }
      // Otherwise if it is null, we have an issue, but there's no need to
      // throw an Error..
      return newState;
    }

    // case ActionType.RefreshTurns: {
    //   // Almost a history refresh, but not quite. We must edit turns in
    //   // place and add more turns.
    //   const gtr = action.payload as GameTurnsRefresher;
    //   const ngs = newStateAfterTurns(state, gtr);
    //   ngs.clockController = state.clockController;
    //   setClock(state, ngs, sge);
    //   return ngs;
    // }
  }
  // This should never be reached, but the compiler is complaining.
  throw new Error(`Unhandled action type ${action.actionType}`);
};
