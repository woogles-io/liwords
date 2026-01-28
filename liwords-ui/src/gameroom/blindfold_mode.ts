import { MachineLetter, MachineWord } from "../utils/cwgame/common";
import {
  PlayState,
  GameEvent,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import {
  wordToSayString,
  say,
  parseBlindfoldCoordinates,
} from "../utils/cwgame/blindfold";
import { singularCount } from "../utils/plural";
import {
  Alphabet,
  machineLetterToRune,
  machineWordToRunes,
  runesToMachineWord,
} from "../constants/alphabets";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import {
  RawPlayerInfo,
  TileDistribution,
} from "../store/reducers/game_reducer";
import { Board } from "../utils/cwgame/board";

type BlindfoldParams = {
  key: string;
  blindfoldCommand: string;
  setBlindfoldCommand: (cmd: string) => void;
  blindfoldUseNPA: boolean;
  setBlindfoldUseNPA: (val: boolean) => void;
  isMyTurn: boolean;
  pool: TileDistribution;
  board: Board;
  turns: Array<GameEvent>;
  players: Array<RawPlayerInfo>;
  playState: number;
  p0Time: number;
  p1Time: number;
  playerMeta: Array<PlayerInfo>;
  username: string;
  exchangeAllowed: boolean;
  setCurrentMode: React.Dispatch<
    React.SetStateAction<
      | "BLANK_MODAL"
      | "DRAWING_HOTKEY"
      | "EXCHANGE_MODAL"
      | "NORMAL"
      | "BLIND"
      | "EDITING_RACK"
      | "WAITING_FOR_RACK_EDIT"
    >
  >;
  makeMove: (move: string, addl?: Array<MachineLetter>) => void;
  gameDone: boolean;
  handleNeitherShortcut: { current: (() => void) | null };
  setArrowProperties: (props: {
    row: number;
    col: number;
    horizontal: boolean;
    show: boolean;
  }) => void;
  nicknameFromEvt: (evt: GameEvent, playerMeta: Array<PlayerInfo>) => string;
  currentRack: MachineWord;
  alphabet: Alphabet;
};

export function handleBlindfoldKeydown(
  evt: React.KeyboardEvent,
  params: BlindfoldParams,
) {
  const {
    key,
    blindfoldCommand,
    setBlindfoldCommand,
    blindfoldUseNPA,
    setBlindfoldUseNPA,
    isMyTurn,
    pool,
    players,
    board,
    turns,
    playState,
    p0Time,
    p1Time,
    playerMeta,
    username,
    exchangeAllowed,
    setCurrentMode,
    makeMove,
    gameDone,
    handleNeitherShortcut,
    currentRack,
    alphabet,
  } = params;

  function PlayerScoresAndTimes(): [
    string,
    number,
    string,
    string,
    number,
    string,
  ] {
    const timepenalty = (time: number) => {
      if (time >= 0) return 0;
      const minsOvertime = Math.ceil(Math.abs(time) / 60000);
      return minsOvertime * 10;
    };

    let p0 = players[0];
    let p1 = players[1];
    let p0TimeU = p0Time;
    let p1TimeU = p1Time;

    if (playerMeta[0].userId === p1.userID) {
      [p0, p1] = [p1, p0];
      [p0TimeU, p1TimeU] = [p1Time, p0Time];
    }

    const playing = playState !== PlayState.GAME_OVER;
    const applyTimePenalty = playing;
    let p0Score = p0?.score ?? 0;
    if (applyTimePenalty) p0Score -= timepenalty(p0TimeU);
    let p1Score = p1?.score ?? 0;
    if (applyTimePenalty) p1Score -= timepenalty(p1TimeU);

    if (playerMeta[1].nickname === username) {
      return [
        "you",
        p1Score,
        playerTimeToText(p1TimeU),
        "opponent",
        p0Score,
        playerTimeToText(p0TimeU),
      ];
    }
    return [
      "you",
      p0Score,
      playerTimeToText(p0TimeU),
      "opponent",
      p1Score,
      playerTimeToText(p1TimeU),
    ];
  }

  function sayGameEvent(ge: GameEvent) {
    const type = ge.type;
    let nickname = "opponent.";
    const evtNickname = params.nicknameFromEvt(ge, playerMeta);
    if (evtNickname === username) {
      nickname = "you.";
    }
    const playedTiles = ge.playedTiles;
    const mainWord = ge.wordsFormed[0];
    let blankAwareWord = "";
    for (let i = 0; i < playedTiles.length; i++) {
      const tile = playedTiles[i];
      if (tile >= "a" && tile <= "z") {
        blankAwareWord += tile;
      } else {
        blankAwareWord += mainWord[i];
      }
    }
    if (type === GameEvent_Type.TILE_PLACEMENT_MOVE) {
      say(
        nickname + " " + wordToSayString(ge.position, blindfoldUseNPA),
        wordToSayString(blankAwareWord, blindfoldUseNPA) +
          " " +
          ge.score.toString(),
      );
    } else if (type === GameEvent_Type.PHONY_TILES_RETURNED) {
      say(nickname + " lost challenge", "");
    } else if (type === GameEvent_Type.EXCHANGE) {
      say(nickname + " exchanged " + ge.exchanged, "");
    } else if (type === GameEvent_Type.PASS) {
      say(nickname + " passed", "");
    } else if (type === GameEvent_Type.CHALLENGE) {
      say(nickname + " challenged", "");
    } else if (type === GameEvent_Type.CHALLENGE_BONUS) {
      say(nickname + " challenge bonus", "");
    } else {
      say(nickname + " 5 point challenge or outplay", "");
    }
  }

  function playerTimeToText(ms: number): string {
    const neg = ms < 0;
    const absms = Math.abs(ms);
    let totalSecs;
    if (!neg) {
      totalSecs = Math.ceil(absms / 1000);
    } else {
      totalSecs = Math.floor(absms / 1000);
    }
    const secs = totalSecs % 60;
    const mins = Math.floor(totalSecs / 60);

    let negative = "";
    if (neg) negative = "negative ";
    let minutes = "";
    if (mins) minutes = singularCount(mins, "minute", "minutes") + " and ";
    return negative + minutes + singularCount(secs, "second", "seconds");
  }

  let newBlindfoldCommand = blindfoldCommand;
  if (key === "Enter") {
    if (blindfoldCommand.toUpperCase() === "P") {
      if (turns.length < 2) {
        say("no previous play", "");
      } else {
        sayGameEvent(turns[turns.length - 2]);
      }
    } else if (blindfoldCommand.toUpperCase() === "C") {
      if (turns.length < 1) {
        say("no current play", "");
      } else {
        sayGameEvent(turns[turns.length - 1]);
      }
    } else if (blindfoldCommand.toUpperCase() === "S") {
      const [, p0Score, , , p1Score] = PlayerScoresAndTimes();
      const scoresay = `${p0Score} to ${p1Score}`;
      say(scoresay, "");
    } else if (
      blindfoldCommand.toUpperCase() === "E" &&
      exchangeAllowed &&
      !gameDone
    ) {
      evt.preventDefault();
      if (handleNeitherShortcut.current) handleNeitherShortcut.current();
      setCurrentMode("EXCHANGE_MODAL");
      setBlindfoldCommand("");
      say("exchange modal opened", "");
      return;
    } else if (blindfoldCommand.toUpperCase() === "PASS" && !gameDone) {
      makeMove("pass");
      setCurrentMode("NORMAL");
    } else if (blindfoldCommand.toUpperCase() === "CHAL" && !gameDone) {
      makeMove("challenge");
      setCurrentMode("NORMAL");
      return;
    } else if (blindfoldCommand.toUpperCase() === "T") {
      const [, , p0Time, , , p1Time] = PlayerScoresAndTimes();
      const timesay = `${p0Time} to ${p1Time}.`;
      say(timesay, "");
    } else if (blindfoldCommand.toUpperCase() === "R") {
      say(
        wordToSayString(
          machineWordToRunes(currentRack, alphabet),
          blindfoldUseNPA,
        ),
        "",
      );
    } else if (blindfoldCommand.toUpperCase() === "B") {
      const bag = { ...pool };
      for (let i = 0; i < currentRack.length; i += 1) {
        bag[currentRack[i]] -= 1;
      }
      let numTilesRemaining = 0;
      let tilesRemaining = "";
      let blankString = " ";
      for (const [key, value] of Object.entries(bag)) {
        const letter = machineLetterToRune(parseInt(key, 10), alphabet) + ". ";
        const numValue = value as number;
        if (numValue > 0) {
          numTilesRemaining += numValue;
          if (key === "0") {
            blankString = `${numValue}, blank`;
          } else {
            tilesRemaining += `${numValue}, ${letter}`;
          }
        }
      }
      say(
        `${numTilesRemaining} tiles unseen, ` +
          wordToSayString(tilesRemaining, blindfoldUseNPA) +
          blankString,
        "",
      );
    } else if (
      blindfoldCommand.charAt(0).toUpperCase() === "B" &&
      blindfoldCommand.length === 2 &&
      blindfoldCommand.charAt(1).match(/[a-z.]/i)
    ) {
      const bag = { ...pool };
      for (let i = 0; i < currentRack.length; i += 1) {
        bag[currentRack[i]] -= 1;
      }
      let tile = blindfoldCommand.charAt(1).toUpperCase();
      try {
        const letter = runesToMachineWord(tile, alphabet)[0];
        let numTiles = bag[letter];
        if (tile === ".") {
          tile = "?";
          numTiles = bag[letter];
          say(`${numTiles}, blank`, "");
        } else {
          say(wordToSayString(`${numTiles}, ${tile}`, blindfoldUseNPA), "");
        }
      } catch {
        // do nothing.
      }
    } else if (blindfoldCommand.toUpperCase() === "N") {
      setBlindfoldUseNPA(!blindfoldUseNPA);
      say(
        "NATO Phonetic Alphabet is " +
          (!blindfoldUseNPA ? " enabled." : " disabled."),
        "",
      );
    } else if (blindfoldCommand.toUpperCase() === "W") {
      if (isMyTurn) {
        say("It is your turn.", "");
      } else {
        say("It is your opponent's turn", "");
      }
    } else if (blindfoldCommand.toUpperCase() === "L") {
      say(
        "B for bag. C for current play. " +
          "E for exchange. N for NATO pronunciations. " +
          "P for the previous play. R for rack. " +
          "S for score. T for time. W for turn. " +
          "P, A, S, S, for pass. C, H, A, L, for challenge.",
        "",
      );
    } else {
      const blindfoldCoordinates = parseBlindfoldCoordinates(blindfoldCommand);
      if (blindfoldCoordinates !== undefined) {
        say(wordToSayString(blindfoldCommand, blindfoldUseNPA), "");
        const existingTile = board.letterAt(
          blindfoldCoordinates.row,
          blindfoldCoordinates.col,
        );
        if (existingTile === 0) {
          // EmptyBoardSpaceMachineLetter === 0
          params.setArrowProperties({
            row: blindfoldCoordinates.row,
            col: blindfoldCoordinates.col,
            horizontal: blindfoldCoordinates.horizontal,
            show: true,
          });
        }
      } else {
        console.log("invalid command: ", blindfoldCommand);
        say("invalid command", "");
      }
    }

    newBlindfoldCommand = "";
    setCurrentMode("NORMAL");
  } else {
    newBlindfoldCommand = blindfoldCommand + key.toUpperCase();
  }
  setBlindfoldCommand(newBlindfoldCommand);
}
