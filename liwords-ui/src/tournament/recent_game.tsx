type player = {
  username: string;
  score: number;
  result: string;
};

export type RecentGame = {
  players: Array<player>;
  endReason: string;
  gameId: string;
  time: bigint;
  round: number;
};

export const pageSize = 20;
