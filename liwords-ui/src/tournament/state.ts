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
