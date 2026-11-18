// One pre-endgame (two or more tiles in the bag) run in parts, one part a
// worker (lab.ts), put back together. Part k of n values only the bag's sets
// whose index is k mod n, so a play's rows in every part together are the
// whole search's rows, and its win and value are their means weighed by each
// row's deals. The best is the most win, then the most value. The merge is
// exact once every part is.

export type EndgameRow = {
  tiles: Array<number>;
  count: number;
  value: number;
  win?: number;
  drawn?: Array<unknown>;
};

export type EndgamePartCandidate = {
  play: unknown;
  win?: number;
  value: number;
  exact?: boolean;
  // the deals its rows cover, which a step's answer gives in place of the
  // rows themselves (read apart, lab.worker.ts "draws")
  deals?: number;
  draws?: Array<EndgameRow>;
};

export type EndgamePartAnswer = {
  bag: number;
  steps: number;
  nodes: number;
  exact: boolean;
  play: unknown;
  value?: number;
  win?: number;
  spread: number;
  depth: number;
  endgames?: number;
  moves?: [number, number];
  draws?: Array<EndgameRow>;
  candidates?: Array<EndgamePartCandidate>;
  part?: [number, number];
};

const sum = (xs: Array<number>) => xs.reduce((a, b) => a + b, 0);

// A play's win and value from every part, each part's weighed by the deals
// its rows cover (its own win and value being its rows' means weighed so),
// and its rows together where every part gave them; a play some part has not
// valued yet is left out.
const mergeCandidate = (
  parts: Array<EndgamePartCandidate>,
): EndgamePartCandidate => {
  const dealsOf = parts.map(
    (c) => c.deals ?? sum((c.draws ?? []).map((d) => d.count)),
  );
  const deals = sum(dealsOf);
  const weigh = (of: (c: EndgamePartCandidate) => number) =>
    deals
      ? sum(parts.map((c, k) => (dealsOf[k] ? dealsOf[k] * of(c) : 0))) / deals
      : undefined;
  return {
    play: parts[0].play,
    win: weigh((c) => c.win ?? 0),
    value: weigh((c) => c.value) ?? parts[0].value,
    exact: parts.every((c) => c.exact !== false),
    deals: deals || undefined,
    draws: parts.every((c) => c.draws)
      ? parts.flatMap((c) => c.draws!)
      : undefined,
  };
};

const byBest = (a: EndgamePartCandidate, b: EndgamePartCandidate) =>
  (b.win ?? 0) - (a.win ?? 0) || b.value - a.value;

// The parts' answers merged into the one search's. While any part lists no
// play valued, the merge has none either: no value or win, depth 0, no
// rows or candidates.
export const mergeEndgameAnswers = (
  answers: Array<EndgamePartAnswer>,
): EndgamePartAnswer => {
  const [first] = answers;
  // the lead now, the same in every part: an answer's spread less its value
  const valued = answers.find((a) => a.value !== undefined);
  const lead = valued ? valued.spread - valued.value! : first.spread;
  const totals = {
    bag: first.bag,
    steps: Math.min(...answers.map((a) => a.steps)),
    nodes: sum(answers.map((a) => a.nodes)),
    endgames: sum(answers.map((a) => a.endgames ?? 0)),
    exact: answers.every((a) => a.exact),
  };
  const moves: [number, number] | undefined = first.moves
    ? [Math.min(...answers.map((a) => a.moves?.[0] ?? 0)), first.moves[1]]
    : undefined;
  const keyed = answers.map(
    (a) =>
      new Map((a.candidates ?? []).map((c) => [JSON.stringify(c.play), c])),
  );
  const candidates =
    // a part given no set lists its plays with no rows, and may name none
    answers.every((a) => a.candidates?.length)
      ? [...keyed[0].keys()]
          .filter((k) => keyed.every((m) => m.has(k)))
          .map((k) => mergeCandidate(keyed.map((m) => m.get(k)!)))
          .sort(byBest)
      : [];
  const best = candidates[0];
  if (!best) {
    return {
      ...totals,
      exact: false,
      play: null,
      spread: lead,
      depth: 0,
      moves,
      draws: [],
      candidates: [],
    };
  }
  return {
    ...totals,
    play: best.play,
    value: best.value,
    win: best.win,
    spread: lead + best.value,
    depth: Math.min(...answers.map((a) => a.depth)),
    moves,
    draws: best.draws,
    candidates,
  };
};
