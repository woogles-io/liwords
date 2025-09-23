import { ChatMessage } from "../gen/api/proto/ipc/chat_pb";
import { TType } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { ChallengeRule } from "../gen/api/vendor/macondo/macondo_pb";
import {
  BlankMachineLetter,
  EmptyRackSpaceMachineLetter,
  MachineWord,
} from "../utils/cwgame/common";
import { Alphabet, machineLetterToRune } from "../constants/alphabets";
import { WooglesError } from "../gen/api/proto/ipc/errors_pb";

export type PlayerOrder = "p0" | "p1";

export enum ChatEntityType {
  UserChat,
  ServerMsg,
  ErrorMsg,
}

export type ChatEntityObj = {
  entityType: ChatEntityType;
  sender: string;
  message: string;
  id?: string;
  timestamp?: bigint;
  senderId?: string;
  channel: string;
};

export type PresenceEntity = {
  uuid: string;
  username: string;
  channel: string;
  anon: boolean;
  deleting: boolean;
};

export const randomID = () => {
  // Math.random should be unique because of its seeding algorithm.
  // Convert it to base 36 (numbers + letters), and grab the first 9 characters
  // after the decimal.
  return `_${Math.random().toString(36).substring(2, 11)}`;
};

export const indexToPlayerOrder = (idx: number): PlayerOrder => {
  if (!(idx >= 0 && idx <= 1)) {
    throw new Error("unexpected player index");
  }
  return `p${idx}` as PlayerOrder;
};

// number of turns in a game, this is just an estimate. See `variants.go`
const turnsPerGame = 16;

export const calculateTotalTime = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
): number => {
  return secs + maxOvertime * 60 + incrementSecs * turnsPerGame;
};

export type valueof<T> = T[keyof T];

export const isPairedMode = (type: TType) => {
  return type === TType.CHILD || type === TType.STANDARD;
};

export const isClubType = (type: TType) => {
  return type === TType.CHILD || type === TType.CLUB;
};
// See cutoffs in variants.go. XXX: Try to tie these together better.
export const timeCtrlToDisplayName = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
  overrideDisplay?: string,
) => {
  if (overrideDisplay) {
    return [overrideDisplay, overrideDisplay, "white"];
  }
  const totalTime = calculateTotalTime(secs, incrementSecs, maxOvertime);

  if (totalTime <= 2 * 60) {
    return ["Ultra-Blitz!", "ultrablitz", "magenta"];
  }
  if (totalTime <= 6 * 60) {
    return ["Blitz", "blitz", "volcano"];
  }
  if (totalTime <= 14 * 60) {
    return ["Rapid", "rapid", "gold"];
  }
  return ["Regular", "regular", "blue"];
};
export const StartingRating = "1500";
// see ToVariantKey and related functions in ratings.go
export const ratingKey = (
  secs: number,
  incrementSecs: number,
  maxOvertime: number,
  variant: string,
  lexicon: string,
) => {
  const a = timeCtrlToDisplayName(secs, incrementSecs, maxOvertime);
  const tfmt = a[1];
  let lexVariant = lexicon;
  // These are just used for the hard-coded rating keys in the profile.
  if (lexicon.startsWith("NWL")) {
    lexVariant = "NWL18";
  }
  if (lexicon.startsWith("CSW")) {
    lexVariant = "CSW19";
  }
  if (lexicon.startsWith("ECWL")) {
    lexVariant = "ECWL";
  }
  return `${lexVariant}.${variant}.${tfmt}`;
};

const initialTimeLabel = (secs: number) => {
  let initTLabel;
  switch (secs) {
    case 15:
      initTLabel = "¼";
      break;
    case 30:
      initTLabel = "½";
      break;
    case 45:
      initTLabel = "¾";
      break;
    default:
      initTLabel = `${secs / 60}`;
  }
  return initTLabel;
};

export const initTimeDiscreteScale = [
  15,
  30,
  45,
  ...Array.from(new Array(25), (_, x) => (x + 1) * 60),
  ...[30, 35, 40, 45, 50, 55, 60].map((x) => x * 60),
].map((seconds) => ({
  seconds,
  label: initialTimeLabel(seconds),
}));

export const initialTimeSecondsToSlider = (secs: number) => {
  const ret = initTimeDiscreteScale.findIndex((x) => x.seconds === secs);
  if (ret >= 0) return ret;
  throw new Error(`bad initial time: ${secs} seconds`);
};

export const initialTimeMinutesToSlider = (mins: number) =>
  initialTimeSecondsToSlider(mins * 60);

export const timeToString = (
  secs: number,
  incrementSecs: number,
  maxOvertimeMinutes: number,
  untimed?: boolean,
) => {
  if (untimed) {
    return "";
  }
  return `${initialTimeLabel(secs)}${
    maxOvertimeMinutes ? `/${maxOvertimeMinutes}` : ""
  }${incrementSecs ? `+${incrementSecs}` : ""}`;
};

export const chatMessageToChatEntity = (cm: ChatMessage): ChatEntityObj => {
  return {
    entityType: ChatEntityType.UserChat,
    id: cm.id || randomID(),
    sender: cm.username,
    message: cm.message,
    timestamp: cm.timestamp,
    senderId: cm.userId,
    channel: cm.channel,
  };
};

export const ratingToColor = (rating: string): [number, string] => {
  let ratNum;
  if (rating.endsWith("?")) {
    ratNum = parseInt(rating.substring(0, rating.length - 1), 10);
  } else {
    ratNum = parseInt(rating, 10);
  }
  const ratingCutoffs: Array<[number, string]> = [
    [2100, "pink"],
    [1900, "volcano"],
    [1700, "yellow"],
    [1500, "orange"],
    [1300, "cyan"],
    [1100, "green"],
    [900, "blue"],
    [700, "purple"],
    [500, "gold"],
    [300, "lime"],
    [100, "gray"],
  ];
  for (let r = 0; r < ratingCutoffs.length; r++) {
    if (ratNum >= ratingCutoffs[r][0]) {
      return [ratNum, ratingCutoffs[r][1]];
    }
  }
  // If you're rated under 100 you're a geek.
  return [ratNum, "geekblue"];
};

export const challRuleToStr = (n: number): string => {
  switch (n) {
    case ChallengeRule.DOUBLE:
      return "x2";
    case ChallengeRule.SINGLE:
      return "x1";
    case ChallengeRule.TRIPLE:
      return "x3";
    case ChallengeRule.FIVE_POINT:
      return "+5";
    case ChallengeRule.TEN_POINT:
      return "+10";
    case ChallengeRule.VOID:
      return "Void";
  }
  return "Unsupported";
};

export let sharedEnableAutoShuffle =
  localStorage?.getItem("enableAutoShuffle") === "true";

export const setSharedEnableAutoShuffle = (value: boolean) => {
  if (value) {
    localStorage.setItem("enableAutoShuffle", "true");
  } else {
    localStorage.removeItem("enableAutoShuffle");
  }
  sharedEnableAutoShuffle = value;
};

// To expose this and make it more ergonomic to reorder without refreshing.
export let preferredSortOrder = localStorage.getItem("tileOrder");

export const setPreferredSortOrder = (value: string) => {
  if (value) {
    localStorage.setItem("tileOrder", value);
    preferredSortOrder = value;
  } else {
    localStorage.removeItem("tileOrder");
    preferredSortOrder = null;
  }
};

export const sortTiles = (
  rack: MachineWord,
  alphabet: Alphabet,
): MachineWord => {
  const effectiveSortOrder = preferredSortOrder ?? "";
  const arr = Array.from(rack);
  const sorted = arr
    .filter((tile) => tile !== EmptyRackSpaceMachineLetter)
    .map((tile) => {
      const rune = machineLetterToRune(tile, alphabet);
      let index = effectiveSortOrder.indexOf(rune);
      if (index < 0)
        index =
          effectiveSortOrder.length + (tile === BlankMachineLetter ? 1 : 0);
      return [index, tile];
    })
    .sort(([aIndex, aTile], [bIndex, bTile]) =>
      aIndex < bIndex
        ? -1
        : aIndex > bIndex
          ? 1
          : aTile < bTile
            ? -1
            : aTile > bTile
              ? 1
              : 0,
    )
    .map((s) => s[1]);

  return sorted;
};

export const isSpanish = (lexicon: string) => lexicon.startsWith("FILE");

// Can skip error codes for now.
// 1001 - max bye placement - front end doesn't allow this
// 1002 - min gibson placement - front end doesn't allow this
// 1003 - min gibson spread - not allowed in front end
// 1004 - empty round controls - not allowed in front end
export const errorMap: Map<number, string> = new Map<number, string>([
  // Tournament errors
  [
    WooglesError.TOURNAMENT_NEGATIVE_MAX_BYE_PLACEMENT,
    "Max Bye Placement cannot be less than 1: Tournament: $1 Division: $2 Max Bye Placement you entered: $3",
  ],
  [
    WooglesError.TOURNAMENT_NEGATIVE_MIN_PLACEMENT,
    "Min Gibson Placement cannot be less than 1.",
  ],
  [
    WooglesError.TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START,
    "The tournament has already started.",
  ],
  // 1006 - elimination tournament not supported yet.
  [
    WooglesError.TOURNAMENT_DISCONTINUOUS_INITIAL_FONTES,
    "You cannot have other pairings preceding Initial Fontes.",
  ],
  // This is major tom to round controls
  [
    WooglesError.TOURNAMENT_INVALID_INITIAL_FONTES_ROUNDS,
    "Initial Fontes must have an odd number of rounds; you entered $3.",
  ],
  // 1009 - elimination tournament not supported yet.
  [
    WooglesError.TOURNAMENT_ROUND_NUMBER_OUT_OF_RANGE,
    "Round number must be between 1 and the number of rounds.",
  ],
  [WooglesError.TOURNAMENT_NONEXISTENT_PLAYER, "You must select a player."],
  // 1012 - past rounds
  [
    WooglesError.TOURNAMENT_NONAMENDMENT_PAST_RESULT,
    "Turn on Amendment to edit an already existing score.",
  ],
  [
    WooglesError.TOURNAMENT_FUTURE_NONBYE_RESULT,
    "You cannot enter scores for future rounds.",
  ],
  [
    WooglesError.TOURNAMENT_NIL_PLAYER_PAIRING,
    "The pairing did not exist: $3, $4, $6",
  ],
  [
    WooglesError.TOURNAMENT_NONOPPONENTS,
    "The players you selected did not play in this round.",
  ],
  [1016, "Mixed void and non-void game results; please double-check game."],
  [1017, "This pairing did not exist! (Division $2, Round $3, key $5)"],
  [
    1018,
    "This pairing in division $2 round $3 is corrupt and contains no games.",
  ],
  // 1019 - elimination tournament not supported yet
  // 1020 - tournament game index out of range, only for pairings that have multiple
  // numbers of games. Can revisit this later.
  // copypasta

  [1021, "Turn on Amendment to edit an already existing score."],
  [
    1022,
    "Attempted to submit an amendment for a result that does not exist (Division $2, round $3, p1 $4, p2 $5)",
  ],
  [1023, "Gibsonization indexes out of range: [0, $3], $4, $5"],
  [1024, "Unable to assign bye in round $3, division $2"],
  [1025, "Internal error assigning byes: round $3, division $2"],
  // It is unclear how to trigger a bunch of these errors, but let's have
  // some text here and the errors will get logged internally anyway.
  [1026, "Incorrect pairings length: div $2, round $3, $4, $5"],
  [1027, "Internal error assigning byes in round $3, div $2"],
  [
    1028,
    "Internal error pairing; suspended player was not removed in div $2, round $3 ($4)",
  ],
  [
    1029,
    "Internal error pairing; tournament pairing index out of range, div $2, round $3 ($4)",
  ],
  // how?
  [
    1030,
    "Internal error pairing; suspended player was paired! (div $2, round $3, player $4)",
  ],
  [
    1031,
    "Internal error pairing; a player was unable to be paired! (div $2, round $3, player $4)",
  ],
  [
    1032,
    "Unable to add player, as they are already in division $2. (player is $3)",
  ],
  // am I interpreting this error correctly? I didn't think that was a restriction:
  [1033, "A player cannot be added in the last round"],
  [
    1034,
    "Internal error; player index out of range (div $2, round $3, $4, $5)",
  ],
  [1035, "This player has already been removed from division $2."],
  [1036, "Removing this player would create an empty division."],
  // how?
  [1037, "Gibson round seems to be negative!"],
  [1038, "This round cannot be opened because round $3 is not complete."],
  [1039, "This tournament has already finished."],
  [
    1040,
    "Tournament cannot be started. Please ensure all your divisions have at least two players.",
  ],
  [
    1041,
    "Round $3 in division $2 cannot be started because round $3 is not fully paired.",
  ],
  // Should not be possible to trigger, but that's probably the case with a lot of these:
  [1042, "Ready was sent for the wrong round (div $2, round $3)."],
  [1043, "Already received a ready for this player (round $3, player $4)"],
  [1044, "Internal error setting ready: (round $3, player $4)"],
  [1045, "Internal error setting ready: player not found"],
  [1046, "No loser found for this game: $6, $7"],
  [1047, "No winner found for this game: $6, $7"],
  [1048, "There is an unpaired player in division $2, round $3: $4"],
  [1049, "Internal error pairing; pairing matrix is corrupt"],
  [
    1050,
    'Swiss pairings must not have max repeats set to zero if "Allow over max repeats" is false.',
  ],
  // Should not be possible with our UI:
  [1051, "Your tournament has zero games per round!"],
  [1052, "You must enter a name for your tournament."],
  [1053, "The tournament has not yet started."],
  [1054, "The division named $2 does not exist in this tournament."],
  // how?
  [1055, "Internal error: The division manager is nil!"],
  [
    1056,
    "This round is already over or underway. You are currently in round $3 for division $2.",
  ],
  [1057, "You cannot add a division after the tournament has started."],
  [
    1058,
    "Your division name is invalid. Please ensure it is between 1 and 24 characters.",
  ],
  [1059, "The division named $2 already exists in this tournament."],
  [1060, "You cannot remove a division after the tournament has started."],
  [
    1061,
    "You cannot remove a division that has players in it. Please remove the players first.",
  ],
  [1062, "The user with name $3 was not found in our system."],
  [1063, "An executive director already exists for this tournament."],
  [1064, "A director by that name already exists in this tournament."],
  [1065, "This tournament has no divisions."],
  [1066, "Game controls have not been set for division $2."],
  [1067, "Round $3 cannot be started, as the current round is $4."],
  [1068, "You cannot pair a round that is already open, or in the past."],
  [
    1069,
    "You cannot delete pairings for a round that is already open, or in the past.",
  ],
  [1070, "Unable to finish tournament because division $2 is not finished."],
  [1071, "Your tournament can only have exactly one executive director."],
  [1072, "You cannot remove the executive director."],
  [1073, "You must select one of the options for Suspended game result"],
  [
    WooglesError.TOURNAMENT_OPENCHECKINS_AFTER_START,
    "You cannot open check-ins after the tournament has started.",
  ],
  [
    WooglesError.TOURNAMENT_OPENREGISTRATIONS_AFTER_START,
    "You cannot open registrations after the tournament has started.",
  ],
  [
    WooglesError.TOURNAMENT_ALREADY_STARTED,
    "The tournament has already started, so you can't do that.",
  ],
  [
    WooglesError.TOURNAMENT_CANNOT_START_CHECKINS_OR_REGISTRATIONS_OPEN,
    "You cannot start a tournament with check-ins or registrations open. Please close these first.",
  ],
  [
    WooglesError.TOURNAMENT_CANNOT_REMOVE_UNCHECKED_IN_IF_CHECKINS_OPEN,
    "You cannot remove players who are not checked in while check-ins are open. Please close check-ins first.",
  ],
  // Puzzle errors
  [
    1074,
    "Your vote must have a value of -1, 0, or 1, got $3 for puzzle $2 instead.",
  ],
  [1075, "Cannot find a random puzzle ID for lexicon $2."],
  [1076, "Cannot find a puzzle for lexicon $2."],
  [1077, "Cannot find puzzle with ID $2."],
  [1078, "You do not have any previous puzzles."],
  [1079, "You do not have any previous puzzles."],
  [1080, "Cannot find puzzle with ID $1."],
  [1081, "Internal puzzle error: puzzle ID $2 not found for user $1."],
  [
    1082,
    "Internal puzzle error: puzzle attempt not found for user $1, puzzle $2.",
  ],
  [
    1083,
    "Internal puzzle error: updating attempts failed for user $1, puzzle $2.",
  ],
  [
    1084,
    "Internal puzzle error: setting puzzle vote failed for user $1, puzzle $2.",
  ],
  [1086, "Puzzle attempt does not exist for user $1 and puzzle $2."],
  [
    1087,
    "Internal error: could not find puzzle attempt for user $1 and puzzle $2.",
  ],
  [1089, "Game is no longer available."],
]);
