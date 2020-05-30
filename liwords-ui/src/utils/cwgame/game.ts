import {
  PlayerInfo,
  GameTurn,
  GameEvent,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { Direction, EmptySpace } from './common';
import { GameHistoryRefresher } from '../../gen/api/proto/game_service_pb';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';
import { CrosswordGameGridLayout } from '../../constants/board_layout';

/* TODO: should be dependent on board dimensions in future.  */
export function blankLayout() {
  return new Array(225).fill(' ');
}

export type FullPlayerInfo = {
  nickname: string;
  fullName: string;
  flag: string;
  rating: number;
  title: string;
  score: number;
  timeRemainingSec: number;
  onturn: boolean;
  avatarUrl: string;
};

export class Board {
  private letters: Array<string>; // The letters on the board
  gridLayout: Array<string>; // the bonus squares.
  isEmpty: boolean;
  dim: number;

  constructor() {
    this.letters = blankLayout();
    this.isEmpty = true;
    this.gridLayout = CrosswordGameGridLayout;
    this.dim = this.gridLayout.length;
  }

  tilesLayout() {
    const layout = [];
    for (let j = 0; j < 15; j += 1) {
      // row by row
      const x = j * 15;
      const sl = this.letters.slice(x, x + 15);
      layout.push(sl.join(''));
    }
    return layout;
  }
  /** take in a 2D board array */
  setTileLayout(layout: Array<string>) {
    this.isEmpty = true;
    for (let row = 0; row < 15; row += 1) {
      for (let col = 0; col < 15; col += 1) {
        const letter = layout[row][col];
        if (letter !== EmptySpace) {
          this.isEmpty = false;
        }
        this.letters[row * 15 + col] = letter;
      }
    }
  }

  /**
   * Return the letter at the given row, col. Returns null if out of bounds.
   */
  letterAt(row: number, col: number) {
    if (row > this.dim - 1 || row < 0 || col > this.dim - 1 || col < 0) {
      return null;
    }
    return this.letters[row * 15 + col];
  }

  addLetter(row: number, col: number, letter: string) {
    this.letters[row * 15 + col] = letter;
    this.isEmpty = false;
  }

  removeLetter(row: number, col: number, letter: string) {
    this.letters[row * 15 + col] = ' ';
    // don't know how else to check, annoyingly
    this.isEmpty = true;
    for (let i = 0; i < 225; i++) {
      if (this.letters[i] !== ' ') {
        this.isEmpty = false;
        break;
      }
    }
  }
}

function setCharAt(str: string, index: number, chr: string) {
  if (index > str.length - 1) {
    return str;
  }
  return str.substr(0, index) + chr + str.substr(index + 1);
}

export class GameState {
  // This class holds the layout of the board as well as the remaining pool.
  tileDistribution: { [rune: string]: number };
  pool: { [rune: string]: number };
  turns: Array<GameTurn>;
  lastPlayedLetters: { [tile: string]: boolean };
  players: Array<PlayerInfo>;
  scores: { [username: string]: number };
  onturn: number;
  currentRacks: { [username: string]: string };
  board: Board;

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
    if (this.players && this.players.length) {
      this.scores = {
        [players[0].getNickname()]: 0,
        [players[1].getNickname()]: 0,
      };
    } else {
      this.scores = {};
    }
    this.onturn = 0;
    if (this.players && this.players.length) {
      this.currentRacks = {
        [players[0].getNickname()]: '',
        [players[1].getNickname()]: '',
      };
    } else {
      this.currentRacks = {};
    }
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

function trackPlay(idx: number, item: GameEvent, boardState: GameState) {
  boardState.clearLastPlayedLetters();

  let play = item.getPlayedTiles();
  for (let i = 0; i < play.length; i += 1) {
    const letter = play[i];
    const row =
      item.getDirection() === Direction.Vertical
        ? item.getRow() + i
        : item.getRow();
    const col =
      item.getDirection() === Direction.Horizontal
        ? item.getColumn() + i
        : item.getColumn();
    if (letter !== '.') {
      boardState.addLetter(row, col, letter);
      boardState.setLastPlayedLetter(row, col);
    } else {
      play = setCharAt(play, i, boardState.board.letterAt(row, col)!);
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

  return boardState;
}

export const StateFromHistoryRefresher = (
  ghr: GameHistoryRefresher
): GameState => {
  const history = ghr.getHistory();
  const playerList = history!.getPlayersList();

  const gs = new GameState(EnglishCrosswordGameDistribution, playerList);

  const racks = history!.getLastKnownRacksList();

  console.log('racks are', racks);

  const p0nick = playerList[0].getNickname();
  const p1nick = playerList[1].getNickname();

  console.log('nicks', p0nick, p1nick);
  [gs.currentRacks[p0nick], gs.currentRacks[p1nick]] = racks;
  console.log('then current racks are', gs.currentRacks);
  return gs;
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
    timeRemainingSec: 0,
    onturn: state.onturn === playerIdx,
    avatarUrl: '',
  };
  return fpi;
};
