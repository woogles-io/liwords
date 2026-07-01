import React from "react";
import { cleanup, render } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  GameEvent_Type,
  GameEventSchema,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Board } from "../utils/cwgame/board";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { TwoColTurn, twoColScore } from "./scorecard";

afterEach(cleanup);

// twoColScore builds the two-column score text. Segments that render on their
// own line are joined with "\n"; the scorecard render site splits on it and
// inserts <br/>. See issue #1895.

const play = (score: number, cumulative: number) =>
  create(GameEventSchema, { rack: "ABDEKOR", score, cumulative });

const challengeBonus = (bonus: number, cumulative: number) =>
  create(GameEventSchema, {
    type: GameEvent_Type.CHALLENGE_BONUS,
    bonus,
    cumulative,
  });

const endRackPts = (endRackPoints: number, cumulative: number) =>
  create(GameEventSchema, {
    type: GameEvent_Type.END_RACK_PTS,
    endRackPoints,
    cumulative,
  });

const phony = (cumulative: number) =>
  create(GameEventSchema, {
    type: GameEvent_Type.PHONY_TILES_RETURNED,
    cumulative,
  });

it("shows the score for a plain play", () => {
  expect(twoColScore([play(12, 979)])).toBe("+12");
});

it("stacks a valid-challenge bonus after the play", () => {
  expect(twoColScore([play(12, 979), challengeBonus(5, 984)])).toBe("+12\n+5");
});

it("appends end-of-game rack points for a play-out", () => {
  expect(twoColScore([play(12, 979), endRackPts(2, 981)])).toBe("+12+2");
});

it("blanks the score when a phony is returned", () => {
  expect(twoColScore([play(40, 200), phony(160)])).toBe("");
});

// Regression: playing out on a word that is then challenged (and holds)
// produces play + CHALLENGE_BONUS + END_RACK_PTS in one turn. The builder must
// stay a plain string; the old code interpolated a ReactNode into a template
// literal, rendering as "[object Object]". See issue #1895.
it("renders play + challenge bonus + end rack without [object Object]", () => {
  const text = twoColScore([
    play(12, 979),
    challengeBonus(5, 984),
    endRackPts(2, 986),
  ]);
  expect(text).not.toContain("[object Object]");
  expect(text).toBe("+12\n+5+2");
});

// End-to-end render of the actual TwoColTurn component. The aggregated score
// must reach the DOM as "+12" / "+5+2" (the "\n" split into a real <br/>),
// never "[object Object]", alongside the correct cumulative and challenge
// label. This exercises the render path the unit tests above do not.
it("renders the aggregated score in the DOM without [object Object]", () => {
  const turn = {
    events: [
      create(GameEventSchema, {
        rack: "ABDEKOR",
        playedTiles: "WORD",
        position: "3C",
        row: 2,
        column: 2,
        score: 12,
        cumulative: 979,
      }),
      challengeBonus(5, 984),
      endRackPts(2, 986),
    ],
    firstEvtIdx: 0,
  };
  const { container } = render(
    <TwoColTurn
      turn={turn}
      board={new Board()}
      alphabet={StandardEnglishAlphabet}
    />,
  );
  expect(container.textContent).not.toContain("[object Object]");
  const scoreCell = container.querySelector(".two-col-score");
  expect(scoreCell?.querySelector("br")).toBeTruthy();
  expect(scoreCell?.textContent).toBe("+12+5+2");
  expect(container.querySelector(".two-col-cumulative")?.textContent).toBe(
    "986",
  );
  expect(container.textContent).toContain("Challenge! Valid");
});
