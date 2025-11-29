import { create } from "@bufbuild/protobuf";
import { Action, ActionType } from "../../actions/actions";
import {
  MatchUser,
  MatchUserSchema,
  SeekRequest,
} from "../../gen/api/proto/ipc/omgseeks_pb";
import {
  GameInfoResponse,
  RatingMode,
} from "../../gen/api/proto/ipc/omgwords_pb";
import {
  ProfileUpdate,
  ProfileUpdate_Rating,
} from "../../gen/api/proto/ipc/users_pb";
import { BotTypesEnum } from "../../lobby/bots";
import { StartingRating } from "../constants";

export type SoughtGame = {
  seeker: string;
  seekerID?: string;
  lexicon: string;
  initialTimeSecs: number;
  incrementSecs: number;
  maxOvertimeMinutes: number;
  challengeRule: number;
  userRating: string;
  rated: boolean;
  seekID: string;
  playerVsBot: boolean;
  botType: BotTypesEnum;
  variant: string;
  minRatingRange: number;
  maxRatingRange: number;
  gameMode: number;
  // Only for direct match requests:
  receiver: MatchUser;
  rematchFor: string;
  tournamentID: string;
  receiverIsPermanent: boolean;
  ratingKey: string;
  requireEstablishedRating: boolean;
  onlyFollowedPlayers: boolean;
};

type playerMeta = {
  rating: string;
  displayName: string;
  uuid?: string;
};

export type ActiveGame = {
  lexicon: string;
  variant: string;
  initialTimeSecs: number;
  incrementSecs: number;
  challengeRule: number;
  rated: boolean;
  maxOvertimeMinutes: number;
  gameID: string;
  players: Array<playerMeta>;
  tournamentID: string;
  tournamentDivision: string;
  tournamentRound: number;
  tournamentGameIndex: number;
  gameMode: number; // GameMode enum value
  playerOnTurn?: number; // Index of player whose turn it is (0 or 1)
  lastUpdate?: number; // Timestamp of last move in milliseconds
  leagueSlug?: string; // League slug if this is a league game
  scores?: number[]; // Current scores [player0, player1]
};

export type LobbyState = {
  soughtGames: Array<SoughtGame>;
  matchRequests: Array<SoughtGame>;
  correspondenceSeeks: Array<SoughtGame>; // Pending correspondence match requests (incoming + outgoing)
  // + Other things in the lobby here that have state.
  activeGames: Array<ActiveGame>;
  correspondenceGames: Array<ActiveGame>;
  profile: {
    ratings: {
      [k: string]: ProfileUpdate_Rating;
    };
  };
  lobbyFilterByLexicon: string | null;
};

export const SeekRequestToSoughtGame = (
  req: SeekRequest,
): SoughtGame | null => {
  const gameReq = req.gameRequest;
  const user = req.user;
  if (!gameReq || !user) {
    return null;
  }

  let receivingUser = create(MatchUserSchema, {});
  let rematchFor = "";
  let tournamentID = "";
  if (req.receiverIsPermanent) {
    console.log("ismatchrequest");
    receivingUser = req.receivingUser ?? receivingUser;
    rematchFor = req.rematchFor;
    tournamentID = req.tournamentId;
  }

  return {
    seeker: user.displayName,
    seekerID: user.userId,
    userRating: user.relevantRating,
    lexicon: gameReq.lexicon,
    initialTimeSecs: gameReq.initialTimeSeconds,
    challengeRule: gameReq.challengeRule,
    seekID: gameReq.requestId,
    rated: gameReq.ratingMode === RatingMode.RATED,
    minRatingRange: req.minimumRatingRange,
    maxRatingRange: req.maximumRatingRange,
    maxOvertimeMinutes: gameReq.maxOvertimeMinutes,
    receiver: receivingUser,
    rematchFor,
    incrementSecs: gameReq.incrementSeconds,
    playerVsBot: gameReq.playerVsBot,
    tournamentID,
    variant: gameReq.rules?.variantName || "",
    ratingKey: req.ratingKey,
    receiverIsPermanent: req.receiverIsPermanent,
    gameMode: gameReq.gameMode ?? 0,
    requireEstablishedRating: req.requireEstablishedRating,
    onlyFollowedPlayers: req.onlyFollowedPlayers,
    // this is inconsequential as bot match requests are never shown
    // to the user. change if this becomes the case some day.
    botType: 0,
  };
};

export const GameInfoResponseToActiveGame = (
  gi: GameInfoResponse,
): ActiveGame | null => {
  const users = gi.players;
  const gameReq = gi.gameRequest;
  const players = users.map((um) => ({
    rating: um.rating,
    displayName: um.nickname,
    uuid: um.userId,
  }));

  if (!gameReq) {
    return null;
  }
  let variant = gameReq.rules?.variantName;
  if (!variant) {
    variant = "classic";
  }

  // Convert Timestamp to milliseconds
  const lastUpdate = gi.lastUpdate
    ? Number(gi.lastUpdate.seconds) * 1000 +
      Math.floor(Number(gi.lastUpdate.nanos) / 1000000)
    : undefined;

  return {
    players,
    lexicon: gameReq.lexicon,
    variant,
    initialTimeSecs: gameReq.initialTimeSeconds,
    challengeRule: gameReq.challengeRule,
    rated: gameReq.ratingMode === RatingMode.RATED,
    maxOvertimeMinutes: gameReq.maxOvertimeMinutes,
    gameID: gi.gameId,
    incrementSecs: gameReq.incrementSeconds,
    tournamentID: gi.tournamentId,
    tournamentDivision: gi.tournamentDivision,
    tournamentRound: gi.tournamentRound,
    tournamentGameIndex: gi.tournamentGameIndex,
    gameMode: gameReq.gameMode ?? 0, // 0 = REAL_TIME
    playerOnTurn: gi.playerOnTurn,
    lastUpdate,
    leagueSlug: gi.leagueSlug || undefined,
    scores: gi.scores.length >= 2 ? [...gi.scores] : undefined,
  };
};

export const matchesRatingFormula = (
  sg: SoughtGame,
  ratings: { [k: string]: ProfileUpdate_Rating },
) => {
  const ratingKey = sg.ratingKey;
  // Note that accidentally, if sg.userRating ends with a `?`, parseInt still
  // works:
  const seekerRating = parseInt(sg.userRating, 10);

  const receiverRating = ratings[ratingKey];
  // If this rating doesn't exist, then the user has never played this variant
  // before, so their starting rating is the default starting rating.
  const receiverRatingValue =
    receiverRating?.rating || parseInt(StartingRating, 10);

  // minRatingRange should be negative for this to work:
  const minRating = seekerRating + sg.minRatingRange;
  const maxRating = seekerRating + sg.maxRatingRange;
  return receiverRatingValue >= minRating && receiverRatingValue <= maxRating;
};

export const hasEstablishedRating = (
  ratings: { [k: string]: ProfileUpdate_Rating },
  ratingKey: string,
): boolean => {
  const rating = ratings[ratingKey];
  if (!rating) return false;
  // RatingDeviationConfidence = MinimumRatingDeviation(60) + 30 = 90
  return rating.deviation <= 90;
};

export function LobbyReducer(state: LobbyState, action: Action): LobbyState {
  switch (action.actionType) {
    case ActionType.AddSoughtGame: {
      const soughtGame = action.payload as SoughtGame;

      if (!soughtGame.receiverIsPermanent) {
        // Open seek
        const existingSoughtGames = state.soughtGames.filter((sg) => {
          return sg.seekID !== soughtGame.seekID;
        });

        // If it's a correspondence open seek, add to correspondenceSeeks as well
        if (soughtGame.gameMode === 1) {
          const existingCorrespondenceSeeks = state.correspondenceSeeks.filter(
            (sg) => {
              return sg.seekID !== soughtGame.seekID;
            },
          );
          return {
            ...state,
            soughtGames: [...existingSoughtGames, soughtGame],
            correspondenceSeeks: [...existingCorrespondenceSeeks, soughtGame],
          };
        }

        return {
          ...state,
          soughtGames: [...existingSoughtGames, soughtGame],
        };
      } else {
        // Match request (receiverIsPermanent = true)
        const existingMatchRequests = state.matchRequests.filter((sg) => {
          return sg.seekID !== soughtGame.seekID;
        });

        // If it's a correspondence match, ONLY add to correspondenceSeeks (not matchRequests)
        if (soughtGame.gameMode === 1) {
          const existingCorrespondenceSeeks = state.correspondenceSeeks.filter(
            (sg) => {
              return sg.seekID !== soughtGame.seekID;
            },
          );
          return {
            ...state,
            matchRequests: existingMatchRequests, // Don't add correspondence matches to matchRequests
            correspondenceSeeks: [...existingCorrespondenceSeeks, soughtGame],
          };
        }

        // Real-time match requests go to matchRequests only
        return {
          ...state,
          matchRequests: [...existingMatchRequests, soughtGame],
        };
      }
    }

    case ActionType.RemoveSoughtGame: {
      // Look for match requests and correspondence seeks too.
      const { soughtGames, matchRequests, correspondenceSeeks } = state;
      const id = action.payload as string;

      const newSought = soughtGames.filter((sg) => {
        return sg.seekID !== id && !sg.receiverIsPermanent;
      });
      const newMatch = matchRequests.filter((sg) => {
        return sg.seekID !== id && sg.receiverIsPermanent;
      });
      const newCorrespondenceSeeks = correspondenceSeeks.filter((sg) => {
        return sg.seekID !== id;
      });

      return {
        ...state,
        soughtGames: newSought,
        matchRequests: newMatch,
        correspondenceSeeks: newCorrespondenceSeeks,
      };
    }

    case ActionType.AddSoughtGames: {
      const soughtGames = action.payload as Array<SoughtGame>;
      const seeks: SoughtGame[] = [];
      const matches: SoughtGame[] = [];
      const correspondenceSeeks: SoughtGame[] = [];

      soughtGames.forEach(function (sg) {
        if (sg.receiverIsPermanent) {
          // Match request
          if (sg.gameMode === 1) {
            // Correspondence match - ONLY add to correspondenceSeeks
            correspondenceSeeks.push(sg);
          } else {
            // Real-time match - add to matchRequests
            matches.push(sg);
          }
        } else {
          // Open seek - always add to seeks array
          seeks.push(sg);
          // If correspondence, also add to correspondenceSeeks (show in both tabs)
          if (sg.gameMode === 1) {
            correspondenceSeeks.push(sg);
          }
        }
      });

      seeks.sort((a, b) => {
        return a.userRating < b.userRating ? -1 : 1;
      });
      matches.sort((a, b) => {
        return a.userRating < b.userRating ? -1 : 1;
      });
      correspondenceSeeks.sort((a, b) => {
        return a.userRating < b.userRating ? -1 : 1;
      });

      return {
        ...state,
        soughtGames: seeks,
        matchRequests: matches,
        correspondenceSeeks,
      };
    }

    case ActionType.AddActiveGames: {
      const p = action.payload as {
        activeGames: Array<ActiveGame>;
      };
      // Route games based on game mode
      const realTimeGames = p.activeGames.filter((g) => g.gameMode !== 1);
      const correspondenceGames = p.activeGames.filter((g) => g.gameMode === 1);
      return {
        ...state,
        activeGames: realTimeGames,
        correspondenceGames,
      };
    }

    case ActionType.AddActiveGame: {
      const { activeGames, correspondenceGames } = state;
      const p = action.payload as {
        activeGame: ActiveGame;
      };
      // Route to correct array based on game mode
      if (p.activeGame.gameMode === 1) {
        return {
          ...state,
          correspondenceGames: [...correspondenceGames, p.activeGame],
        };
      }
      return {
        ...state,
        activeGames: [...activeGames, p.activeGame],
      };
    }

    case ActionType.RemoveActiveGame: {
      const { activeGames, correspondenceGames } = state;
      const g = action.payload as string;

      const newActiveArr = activeGames.filter((ag) => {
        return ag.gameID !== g;
      });
      const newCorresArr = correspondenceGames.filter((ag) => {
        return ag.gameID !== g;
      });

      return {
        ...state,
        activeGames: newActiveArr,
        correspondenceGames: newCorresArr,
      };
    }

    case ActionType.AddCorrespondenceGames: {
      const p = action.payload as {
        correspondenceGames: Array<ActiveGame>;
      };
      return {
        ...state,
        correspondenceGames: p.correspondenceGames,
      };
    }

    case ActionType.AddCorrespondenceGame: {
      const { correspondenceGames } = state;
      const p = action.payload as {
        correspondenceGame: ActiveGame;
      };
      return {
        ...state,
        correspondenceGames: [...correspondenceGames, p.correspondenceGame],
      };
    }

    case ActionType.RemoveCorrespondenceGame: {
      const { correspondenceGames } = state;
      const g = action.payload as string;

      const newArr = correspondenceGames.filter((ag) => {
        return ag.gameID !== g;
      });

      return {
        ...state,
        correspondenceGames: newArr,
      };
    }

    case ActionType.UpdateCorrespondenceGame: {
      const { correspondenceGames } = state;
      const p = action.payload as {
        correspondenceGame: ActiveGame;
      };

      // Find and update the game, or add if not found
      const gameIndex = correspondenceGames.findIndex(
        (ag) => ag.gameID === p.correspondenceGame.gameID,
      );

      let newGames: ActiveGame[];
      if (gameIndex >= 0) {
        // Update existing game
        newGames = [...correspondenceGames];
        newGames[gameIndex] = p.correspondenceGame;
      } else {
        // Add new game if not found
        newGames = [...correspondenceGames, p.correspondenceGame];
      }

      return {
        ...state,
        correspondenceGames: newGames,
      };
    }

    case ActionType.SetCorrespondenceSeeks: {
      const p = action.payload as {
        correspondenceSeeks: Array<SoughtGame>;
      };
      return {
        ...state,
        correspondenceSeeks: p.correspondenceSeeks,
      };
    }

    case ActionType.UpdateProfile: {
      const { profile } = state;
      const p = action.payload as ProfileUpdate;
      const ratings: { [k: string]: ProfileUpdate_Rating } = {};
      for (const [k, v] of Object.entries(p.ratings)) {
        ratings[k] = v;
      }

      console.log("got ratings", ratings);
      return {
        ...state,
        profile: {
          ...profile,
          ratings,
        },
      };
    }

    case ActionType.setLobbyFilterByLexicon: {
      const newLexiconFilter = action.payload as string | null;
      return {
        ...state,
        lobbyFilterByLexicon: newLexiconFilter,
      };
    }
  }
  throw new Error(`unhandled action type ${action.actionType}`);
}
