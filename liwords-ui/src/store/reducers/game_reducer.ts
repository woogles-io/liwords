import { Board, Tile } from '../../utils/cwgame/board';
import {
  PlayerInfo,
  GameTurn,
  GameEvent,
  GameHistory,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Action, ActionType } from '../../actions/actions';
import {
  ServerGameplayEvent,
  GameHistoryRefresher,
} from '../../gen/api/proto/game_service_pb';
import { Direction, isBlank, Blank } from '../../utils/cwgame/common';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';

type TileDistribution = { [rune: string]: number };

export type FullPlayerInfo = {
  nickname: string;
  fullName: string;
  countryFlag: string;
  rating: number; // in the relevant lexicon / game mode
  title: string;
  score: number;
  // The time remaining is not going to go here. For efficiency,
  // we will put it in its own reducer.
  // timeRemainingMsec: number;
  onturn: boolean;
  avatarUrl: string;
  currentRack: string;
};

const initialExpandToFull = (playerList: PlayerInfo[]): FullPlayerInfo[] => {
  return playerList.map((pi, idx) => {
    return {
      nickname: pi.getNickname(),
      fullName: pi.getRealName(),
      countryFlag: '',
      rating: 0,
      title: '',
      score: 0,
      onturn: idx === 0,
      avatarUrl: '',
      currentRack: '',
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
  };
  return gs;
};

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
      // the game from history whenever there is a challenge event
      // (XXX: This isn't ideal, but the performance impact should be minimal)
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
    if (rune !== '.') {
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

const stateFromHistory = (history: GameHistory): GameState => {
  // XXX: Do this for now. We will eventually want to put the tile
  // distribution itself in the history protobuf.
  // if (['NWL18', 'CSW19'].includes(history!.getLexicon())) {
  //   const dist = EnglishCrosswordGameDistribution;
  // }
  let playerList = history.getPlayersList();
  const flipPlayers = history.getFlipPlayers();
  // If flipPlayers is on, we want to flip the players in the playerList.
  // The backend doesn't do this because of Reasons.
  if (flipPlayers) {
    playerList = [...playerList].reverse();
  }

  const gs = startingGameState(
    EnglishCrosswordGameDistribution,
    initialExpandToFull(playerList),
    history!.getUid()
  );
  history.getTurnsList().forEach((turn, idx) => {
    const events = turn.getEventsList();
    // Skip challenged-off moves:
    if (events.length === 2) {
      if (
        events[0].getType() === GameEvent.Type.TILE_PLACEMENT_MOVE &&
        events[1].getType() === GameEvent.Type.PHONY_TILES_RETURNED
      ) {
        // do not process at all
        return;
      }
    }
    events.forEach((evt) => {
      switch (evt.getType()) {
        case GameEvent.Type.TILE_PLACEMENT_MOVE:
          [gs.lastPlayedTiles, gs.pool] = placeOnBoard(gs.board, gs.pool, evt);
          break;
        default:
        //  do nothing - we only care about tile placement moves here.
      }
    });
    gs.players[gs.onturn].score = events[events.length - 1].getCumulative();
    gs.onturn = (idx + 1) % 2;
  });

  let racks = history.getLastKnownRacksList();
  if (flipPlayers) {
    racks = [...racks].reverse();
  }
  // Assign racks. Remember that the player listed first goes first.
  [gs.players[0].currentRack, gs.players[1].currentRack] = racks;
  gs.players[gs.onturn].onturn = true;
  gs.players[1 - gs.onturn].onturn = false;
  return gs;
};

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
      return ngs;
    }

    case ActionType.RefreshHistory: {
      const ghr = action.payload as GameHistoryRefresher;
      const history = ghr.getHistory();
      return stateFromHistory(history!);
    }
  }
  // This should never be reached, but the compiler is complaining.
  throw new Error(`Unhandled action type ${action.actionType}`);
};
