type tourneytypes = 'STANDARD' | 'CLUB' | 'CLUB_SESSION';

export type TournamentMetadata = {
  name: string;
  description: string;
  directors: Array<string>;
  slug: string;
  id: string;
  type: tourneytypes;
  divisions: Array<string>;
};

export type TournamentState = {
  metadata: TournamentMetadata;
  // standings, pairings, etc. more stuff here to come.
};

export const defaultTournamentState = {
  metadata: {
    name: '',
    description: '',
    directors: new Array<string>(),
    slug: '',
    id: '',
    type: 'STANDARD' as tourneytypes,
    divisions: new Array<string>(),
  },
};
export enum TourneyStatus {
  PRETOURNEY,
  ROUND_BYE,
  ROUND_OPEN,
  ROUND_READY,
  ROUND_OPPONENT_WAITING,
  ROUND_GAME_DEFAULT,
  ROUND_GAME_OPPONENTDISCO,
  ROUND_GAME_FINISHED,
  ROUND_FORFEIT,
  POSTTOURNEY,
}
export type CompetitorState = {
  isRegistered: boolean;
  division?: string;
  status?: TourneyStatus;
  currentRound: number; // Should be the 1 based user facing round
};

export const defaultCompetitorState = {
  isRegistered: false,
  currentRound: 0,
};
