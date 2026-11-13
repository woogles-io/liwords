// One sim run in parts, one part a worker (lab.ts), put back together: the
// parts' answers merged into the one sim's, and the candidates behind the
// leader on the merged numbers found, as the wasm prunes one sim's.

export type Stat = { mean: number; variance: number };
export type CountedStat = Stat & { n: number };

type Ply = { n: number; score: Stat; bingo: number };

export type PartCandidate = {
  play: unknown;
  challenged_off?: boolean;
  state: string;
  n: number;
  win: Stat;
  equity: Stat;
  spread?: Stat;
  leaves?: Stat;
  objective?: Stat;
  plies?: Array<Ply>;
  against?: { win: CountedStat; equity: CountedStat; objective: CountedStat };
};

export type PartAnswer = {
  iterations: number;
  stepped?: number;
  candidates: Array<PartCandidate>;
};

// Statistics of parts merged: n the sum, the mean the n-weighted mean, and
// the variance by Chan's parallel form, each part's M2 its sample variance
// times its n less 1; the merged variance the sample's, 0 under two samples.
export const mergeStats = (parts: Array<{ n: number; stat: Stat }>): Stat => {
  let n = 0;
  let mean = 0;
  let m2 = 0;
  for (const p of parts) {
    if (p.n <= 0) continue;
    const total = n + p.n;
    const d = p.stat.mean - mean;
    mean += (d * p.n) / total;
    m2 += p.stat.variance * (p.n - 1) + (d * d * n * p.n) / total;
    n = total;
  }
  return { mean, variance: n > 1 ? m2 / (n - 1) : 0 };
};

const sum = (xs: Array<number>) => xs.reduce((a, b) => a + b, 0);

const mergeCounted = (stats: Array<CountedStat>): CountedStat => ({
  n: sum(stats.map((s) => s.n)),
  ...mergeStats(stats.map((s) => ({ n: s.n, stat: s }))),
});

// Which candidate a part's answer has at a place: its play and whether it
// was challenged off. Every part is given the same candidates in the same
// order, so each answers with them alike once each has had every change.
const identity = (c: PartCandidate) =>
  JSON.stringify([c.play, !!c.challenged_off]);

// The parts' answers merged, or null while they do not yet hold the same
// candidates (a change some part has not had yet). A candidate runs if it
// runs in every part, and is paused otherwise.
export const mergeAnswers = (answers: Array<PartAnswer>): PartAnswer | null => {
  const [first] = answers;
  if (!first) return null;
  const alike = answers.every(
    (a) =>
      a.candidates.length === first.candidates.length &&
      a.candidates.every(
        (c, i) => identity(c) === identity(first.candidates[i]),
      ),
  );
  if (!alike) return null;
  return {
    iterations: sum(answers.map((a) => a.iterations)),
    stepped: sum(answers.map((a) => a.stepped ?? 0)),
    candidates: first.candidates.map((c0, i) => {
      const cs = answers.map((a) => a.candidates[i]);
      const merged = (
        key: "win" | "equity" | "spread" | "leaves" | "objective",
      ) =>
        cs.every((c) => c[key])
          ? mergeStats(cs.map((c) => ({ n: c.n, stat: c[key] as Stat })))
          : undefined;
      const plies = Math.max(0, ...cs.map((c) => c.plies?.length ?? 0));
      const against = cs.every((c) => c.against)
        ? {
            win: mergeCounted(cs.map((c) => c.against!.win)),
            equity: mergeCounted(cs.map((c) => c.against!.equity)),
            objective: mergeCounted(cs.map((c) => c.against!.objective)),
          }
        : undefined;
      const spread = merged("spread");
      const leaves = merged("leaves");
      const objective = merged("objective");
      return {
        ...c0,
        state: cs.every((c) => c.state === "running") ? "running" : "paused",
        n: sum(cs.map((c) => c.n)),
        win: merged("win")!,
        equity: merged("equity")!,
        ...(spread ? { spread } : {}),
        ...(leaves ? { leaves } : {}),
        ...(objective ? { objective } : {}),
        plies: Array.from({ length: plies }, (_, p) => {
          const ps = cs.flatMap((c) => (c.plies?.[p] ? [c.plies[p]] : []));
          const n = sum(ps.map((x) => x.n));
          return {
            n,
            score: mergeStats(ps.map((x) => ({ n: x.n, stat: x.score }))),
            bingo: n ? sum(ps.map((x) => x.bingo * x.n)) / n : 0,
          };
        }),
        ...(against ? { against } : { against: undefined }),
      };
    }),
  };
};

// The one-sided z of the stitched boundary at a sim's iterations so far, as
// wolges's move_picker::stitched_z works it out: the chance the sim ever
// prunes its best candidate behind a given other one stays at most alpha
// however long it runs.
export const stitchedZ = (iterations: number, alpha: number) =>
  1.7 *
  Math.sqrt(
    Math.log(Math.log(2 * Math.max(1, iterations))) +
      0.72 * Math.log(5.2 / alpha),
  );

type Outcome = "won" | "lost" | "open";

const outcome = (win: number, decided: number): Outcome =>
  win >= 1 - decided ? "won" : win <= decided ? "lost" : "open";

// The running candidate a page sorting by the chance to win lists first, as
// the wasm finds it: if any game is won, the won one of the highest equity;
// if every game is lost, the one of the highest equity; else the one of the
// highest chance to win. Its place, or -1 with none running.
export const leaderOf = (
  candidates: Array<PartCandidate>,
  running: (i: number) => boolean,
  decided: number,
) => {
  const at = candidates.map((_, i) => i).filter(running);
  const best = (by: (i: number) => number, among: Array<number>) =>
    among.reduce((a, b) => (by(b) > by(a) ? b : a), among[0] ?? -1);
  const equity = (i: number) => candidates[i].equity.mean;
  const won = at.filter(
    (i) => outcome(candidates[i].win.mean, decided) === "won",
  );
  if (won.length) return best(equity, won);
  if (at.every((i) => outcome(candidates[i].win.mean, decided) === "lost")) {
    return best(equity, at);
  }
  return best((i) => candidates[i].win.mean, at);
};

// The running candidates behind the leader, by their places: each compared
// with the leader on the iterations both ran (its answer's against, the
// leader's play), by the chance to win, or by equity where both games are
// won or both lost; behind when the difference is below 0 even at the top of
// its interval, z standard errors up, over two iterations or more.
export const behindLeader = (
  candidates: Array<PartCandidate>,
  lead: number,
  running: (i: number) => boolean,
  z: number,
  decided: number,
) => {
  const leadWin = outcome(candidates[lead].win.mean, decided);
  return candidates.flatMap((c, i) => {
    if (i === lead || !running(i) || !c.against) return [];
    const mine = outcome(c.win.mean, decided);
    const diff =
      mine !== "open" && mine === leadWin ? c.against.equity : c.against.win;
    const top = diff.mean + z * Math.sqrt(diff.variance / diff.n);
    return diff.n >= 2 && top < 0 ? [i] : [];
  });
};
