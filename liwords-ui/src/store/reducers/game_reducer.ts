import { Board } from '../../utils/cwgame/board';
import {
  PlayerInfo,
  GameTurn,
  GameEvent,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Action, ActionType } from '../../actions/actions';
import { ServerGameplayEvent } from '../../gen/api/proto/game_service_pb';

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

export type GameState = {
  board: Board;
  // The initial tile distribution:
  tileDistribution: TileDistribution;
  // players are in order of who went first:
  players: Array<FullPlayerInfo>;
  // The unseen tiles to the user (bag and opp's tiles)
  pool: TileDistribution;
  // Array of every turn taken so far.
  turns: Array<GameTurn>;
  // The index of the onturn player in `players`.
  onturn: number;
  currentTurn: GameTurn;
  lastEvent: GameEvent | null;
  gameID: string;
};

export const startingGameState = (
  tileDistribution: TileDistribution,
  players: Array<FullPlayerInfo>
): GameState => {
  const gs = {
    board: new Board(),
    tileDistribution: tileDistribution,
    pool: { ...tileDistribution },
    turns: new Array<GameTurn>(),
    players: players,
    onturn: 0,
    currentTurn: new GameTurn(),
    lastEvent: null,
    gameID: '',
  };
  return gs;
};

// const clonePb = (msg: GameEvent | GameTurn) => {
//   return (typeof msg).deserializeBinary(msg.serializeBinary());
// };

const newGameState = (state: GameState, evt: GameEvent): GameState => {
  // Variables to pass down anew:
  let turns = state.turns;
  let currentTurn;
  let board = state.board;

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
      placeOnBoard(board, evt);
    }
  }
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
      const evt = sge.getEvent();
      return newGameState(state, evt!);
    }
  }
};
