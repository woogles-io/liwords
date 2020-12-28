export type TournamentMetadata = {
  name: string;
  description: string;
  directors: Array<string>;
  slug: string;
  id: string;
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
  },
};
