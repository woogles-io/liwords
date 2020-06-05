import {
  PlayerInfo,
  GameTurn,
  GameEvent,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Direction } from './common';
import {
  GameHistoryRefresher,
  ServerGameplayEvent,
} from '../../gen/api/proto/game_service_pb';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';
import { Board, setCharAt } from './board';

export type FullPlayerInfo = {
  nickname: string;
  fullName: string;
  flag: string;
  rating: number;
  title: string;
  score: number;
  timeRemainingMsec: number;
  onturn: boolean;
  avatarUrl: string;
};

const deepCopy = (state: GameState): GameState => {
  const newState = new GameState({ ...state.tileDistribution }, [
    ...state.players,
  ]);
  newState.pool = { ...state.pool };
  newState.turns = [...state.turns];
  newState.lastPlayedLetters = { ...state.lastPlayedLetters };
  newState.scores = { ...state.scores };
  newState.timeRemaining = { ...state.timeRemaining };
  newState.onturn = state.onturn;
  newState.currentRacks = { ...state.currentRacks };
  newState.currentTurn = GameTurn.deserializeBinary(
    state.currentTurn.serializeBinary()
  );
  newState.lastEvent =
    state.lastEvent === null
      ? state.lastEvent
      : GameEvent.deserializeBinary(state.lastEvent.serializeBinary());
  newState.turnSummary = [...state.turnSummary];
  newState.gameID = state.gameID;
  newState.board = state.board.deepCopy();
  return newState;
};

export class GameState {
  // This class holds the layout of the board as well as the remaining pool.
  tileDistribution: { [rune: string]: number };

  pool: { [rune: string]: number };

  turns: Array<GameTurn>;

  lastPlayedLetters: { [tile: string]: boolean };

  players: Array<PlayerInfo>;

  scores: { [username: string]: number };

  timeRemaining: { [username: string]: number };

  onturn: number;

  currentRacks: { [username: string]: string };

  board: Board;

  currentTurn: GameTurn;

  lastEvent: GameEvent | null;

  turnSummary: Array<string>;

  gameID: string;

  constructor(
    tileDistribution: { [rune: string]: number },
    players: Array<PlayerInfo>
  ) {
    this.board = new Board();
    this.tileDistribution = tileDistribution;
    // tileDistribution should be a map of letter distributions
    // {A: 9, ?: 2, B: 2, etc}
    this.pool = { ...tileDistribution };
    this.turns = new Array<GameTurn>();
    this.lastPlayedLetters = {};
    this.players = players;

    this.onturn = 0;
    if (this.players && this.players.length) {
      this.scores = {
        [players[0].getNickname()]: 0,
        [players[1].getNickname()]: 0,
      };
      this.currentRacks = {
        [players[0].getNickname()]: '',
        [players[1].getNickname()]: '',
      };
      this.timeRemaining = {
        [players[0].getNickname()]: 0,
        [players[1].getNickname()]: 0,
      };
    } else {
      this.currentRacks = {};
      this.scores = {};
      this.timeRemaining = {};
    }
    this.currentTurn = new GameTurn();
    this.lastEvent = null;
    this.turnSummary = [];
    this.gameID = '';
  }

  /**
   * Add a letter at the row, col position. Also, remove letter from pool.
   * XXX: make dependent on board size.
   */
  addLetter(row: number, col: number, letter: string) {
    this.board.addLetter(row, col, letter);
    if (letter.toUpperCase() !== letter) {
      this.pool['?'] -= 1;
    } else {
      this.pool[letter] -= 1;
    }
  }

  setGameID(id: string) {
    this.gameID = id;
  }

  /**
   * Remove a letter from the row, col position, and add back into pool.
   */
  removeLetter(row: number, col: number, letter: string) {
    this.board.removeLetter(row, col, letter);
    if (letter.toUpperCase() !== letter) {
      if (this.pool['?']) {
        this.pool['?'] += 1;
      } else {
        this.pool['?'] = 1;
      }
    } else if (this.pool[letter]) {
      this.pool[letter] += 1;
    } else {
      this.pool[letter] = 1;
    }
  }

  /**
   * This is a utility function to remove a series of letters from the pool.
   */
  removeFromPool(rack: Array<string>) {
    for (let i = 0; i < rack.length; i += 1) {
      this.pool[rack[i]] -= 1;
    }
  }

  /**
   * Pushes a new turn for the given user.
   */
  pushNewTurn(turnRepr: GameTurn) {
    this.turns.push(turnRepr);
    const events = turnRepr.getEventsList();
    const score = events[events.length - 1].getScore();
    this.scores[events[0].getNickname()] = score;
    this.onturn = (this.onturn + 1) % 2;
  }

  /**
   * Push a new event
   */
  pushNewEvent(evt: GameEvent) {
    console.log('push new event', evt);
    if (
      this.lastEvent !== null &&
      evt.getNickname() !== this.lastEvent.getNickname()
    ) {
      // Create a new turn.
      this.turns.push(this.currentTurn);
      this.currentTurn = new GameTurn();
    }

    this.currentTurn.addEvents(evt, this.currentTurn.getEventsList().length);
    switch (evt.getType()) {
      case GameEvent.Type.TILE_PLACEMENT_MOVE:
        this.placeOnBoard(evt);
        break;
      case (GameEvent.Type.EXCHANGE, GameEvent.Type.PASS):
        // do nothing for now. I think this is right.
        break;
      case GameEvent.Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
        break;
      case GameEvent.Type.PHONY_TILES_RETURNED:
        this.unplaceOnBoard(this.lastEvent);
        break;
      // this does not add to the board but it modifies the pool.
      // if I am the player who exchanged, this should give me new tiles
      // and thus modify the pool

      // if the other player exchanged, the pool should not be modified.

      // if (evt.getUnknownExchange() > 0) {
      //   // We don't know what the new rack is. Do nothing.
      // } else {
      //   // It is likely "our" own exchange, or somehow known in another way.
      //   this.setCurrentRack(evt.getNickname(), evt.getRack());
      // }
    }
    this.scores[evt.getNickname()] = evt.getCumulative();
    console.log(
      'assigning score',
      evt.getCumulative(),
      evt.getNickname(),
      this.scores
    );
    // events always switch turns visually, although not necessarily in the game
    // for example, it is on the player on-turn to challenge; if the play comes
    // off (or stays on) this will add an event to the player who made the play.
    // XXX: this might need to change as i add more event types.
    this.onturn = (this.onturn + 1) % 2;
    this.lastEvent = evt;
  }

  placeOnBoard(evt: GameEvent) {
    this.clearLastPlayedLetters();

    let play = evt.getPlayedTiles();
    for (let i = 0; i < play.length; i += 1) {
      const letter = play[i];
      const row =
        evt.getDirection() === Direction.Vertical
          ? evt.getRow() + i
          : evt.getRow();
      const col =
        evt.getDirection() === Direction.Horizontal
          ? evt.getColumn() + i
          : evt.getColumn();
      if (letter !== '.') {
        this.addLetter(row, col, letter);
        this.setLastPlayedLetter(row, col);
      } else {
        play = setCharAt(play, i, this.board.letterAt(row, col)!);
      }
    }
    // Now push a summary of the play
    // boardState.pushNewTurn(item.nick, {
    //   pos: item.pos,
    //   summary: play,
    //   score: `+${item.score}`,
    //   cumul: item.cumul,
    //   turnIdx: idx,
    //   note: item.note,
    //   nick: item.nick,
    //   type: MoveTypesEnum.SCORING_PLAY,
    //   rack: item.rack,
    // });
  }

  unplaceOnBoard(evt: GameEvent | null) {
    if (!evt) {
      return;
    }
    this.clearLastPlayedLetters();

    const play = evt.getPlayedTiles();
    for (let i = 0; i < play.length; i += 1) {
      const letter = play[i];
      const row =
        evt.getDirection() === Direction.Vertical
          ? evt.getRow() + i
          : evt.getRow();
      const col =
        evt.getDirection() === Direction.Horizontal
          ? evt.getColumn() + i
          : evt.getColumn();
      if (letter !== '.') {
        this.removeLetter(row, col, letter);
      }
    }
  }

  clearLastPlayedLetters() {
    this.lastPlayedLetters = {};
  }

  setLastPlayedLetter(row: number, col: number) {
    this.lastPlayedLetters[`R${row}C${col}`] = true;
  }

  // XXX: make it so the backend never sends me the other player's rack
  setCurrentRack(username: string, rack: string) {
    this.currentRacks[username] = rack;
  }

  // always in seconds.
  setTimeRemaining(username: string, time: number) {
    this.timeRemaining[username] = time;
  }

  // toDict() {
  //   return {
  //     lastPlayedLetters: this.lastPlayedLetters,
  //     perPlayerTurns: this.turns,
  //     scores: this.scores,
  //     pool: this.pool,
  //     tilesLayout: tilesLayout(this.layout),
  //     latestTurn: this.latestTurn,
  //     quacklePlayerID: this.quacklePlayerID,
  //     quackleTurnNumber: this.quackleTurnNumber,
  //   };
  // }
}

export const StateFromHistoryRefresher = (
  ghr: GameHistoryRefresher
): GameState => {
  const history = ghr.getHistory();
  const playerList = history!.getPlayersList();

  const gs = new GameState(EnglishCrosswordGameDistribution, playerList);
  gs.setGameID(history!.getUid());

  // Go through game history.
  history!.getTurnsList().forEach((turn) => {
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
          gs.placeOnBoard(evt);
          break;
        default:
        // do nothing - we only care about tile placement moves here.
      }
    });

    // Set cumulative scores after every turn
    const player = events[0].getNickname();
    gs.scores[player] = events[events.length - 1].getCumulative();
  });

  const racks = history!.getLastKnownRacksList();

  console.log('racks are', racks);

  const p0nick = playerList[0].getNickname();
  const p1nick = playerList[1].getNickname();

  console.log('nicks', p0nick, p1nick);
  [gs.currentRacks[p0nick], gs.currentRacks[p1nick]] = racks;

  if (history!.getTurnsList().length % 2 === 0) {
    // because we default to the first player in the array being first
    // unless flipPlayers is set.
    gs.onturn = history!.getFlipPlayers() ? 1 : 0;
  } else {
    gs.onturn = history!.getFlipPlayers() ? 0 : 1;
  }

  gs.setTimeRemaining(p0nick, ghr.getTimePlayer1());
  gs.setTimeRemaining(p1nick, ghr.getTimePlayer2());

  console.log('then current racks are', gs.currentRacks);
  console.log('onturn', gs.onturn, gs.players[gs.onturn]);
  console.log('scores', gs.scores);
  return gs;
};

export const StateForwarder = (
  sge: ServerGameplayEvent,
  state: GameState
): GameState => {
  console.log(
    'in stateforwarder',
    sge.getGameId(),
    state.gameID,
    sge.getGameId() === state.gameID
  );
  if (sge.getGameId() !== state.gameID) {
    return state; // no change.
  }

  // Returns a new game state from a state that has had an sge applied to it.
  const evt = sge.getEvent();
  // Should benchmark this. In all likelihood it's fast enough.
  const newState = deepCopy(state);

  // Always assume it's valid, if it's not we have bigger issues.
  newState.pushNewEvent(evt!);
  if (
    evt!.getType() === GameEvent.Type.TILE_PLACEMENT_MOVE ||
    evt!.getType() === GameEvent.Type.EXCHANGE ||
    evt!.getType() === GameEvent.Type.PHONY_TILES_RETURNED
  ) {
    newState.setCurrentRack(evt!.getNickname(), sge.getNewRack());
  }
  // Otherwise, ignore it.
  newState.setTimeRemaining(evt!.getNickname(), sge.getTimeRemaining());

  console.log('returning a new state. scores', newState.scores);
  return newState;
};

export const turnSummary = (sge: ServerGameplayEvent): string => {
  const evt = sge.getEvent();
  switch (evt?.getType()) {
    case GameEvent.Type.TILE_PLACEMENT_MOVE: {
      const player = evt.getNickname();
      const move = evt.getPlayedTiles();
      const position = evt.getPosition();
      const score = evt.getScore();
      return `${player} played ${position} ${move} for ${score} points.`;
    }
    case GameEvent.Type.EXCHANGE: {
      const player = evt.getNickname();
      const move = evt.getExchanged();
      return `${player} exchanged ${move.length} tiles.`;
    }
    case GameEvent.Type.PASS:
      const player = evt.getNickname();
      return `${player} passed their turn.`;
    default:
      return `unhandled event: ${evt?.getType()}`;
  }
};

export const fullPlayerInfo = (
  playerIdx: number,
  state: GameState
): FullPlayerInfo | null => {
  const pi = state.players[playerIdx];
  if (!pi) {
    return null;
  }
  const fpi = {
    nickname: pi.getNickname(),
    fullName: pi.getRealName(),
    flag: '',
    rating: 0,
    title: '',
    score: state.scores[pi.getNickname()],
    timeRemainingMsec: state.timeRemaining[pi.getNickname()],
    onturn: state.onturn === playerIdx,
    avatarUrl: '',
  };
  return fpi;
};
