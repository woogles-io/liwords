type player = {
  username: string;
  score: number;
  result: string;
};

export type RecentGame = {
  players: Array<player>;
  end_reason: string;
  game_id: string;
  time: number;
};

export const pageSize = 20;
