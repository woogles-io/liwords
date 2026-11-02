import React from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { PoolFormatType } from "../constants/pool_formats";
import { Board } from "../utils/cwgame/board";
import { ScoreCard } from "./scorecard";

vi.mock("./analyzer", () => ({ Analyzer: () => <div>the analyzer</div> }));
vi.mock("./notepad", () => ({
  Notepad: () => <div>the notepad</div>,
  useNotepadHasContent: () => false,
}));

afterEach(cleanup);

const renderScoreCard = (props: {
  isExamining?: boolean;
  hideAnalyzer?: boolean;
}) =>
  render(
    <ScoreCard
      events={[]}
      board={new Board()}
      poolFormat={PoolFormatType.Alphabet}
      playerMeta={[]}
      {...props}
    />,
  );

it("shows the analyzer while examining", () => {
  renderScoreCard({ isExamining: true });
  expect(screen.getByText("the analyzer")).toBeInTheDocument();
  expect(screen.queryByText("the notepad")).toBeNull();
});

it("shows the notepad instead while a player examines their game in play", () => {
  renderScoreCard({ isExamining: true, hideAnalyzer: true });
  expect(screen.getByText("the notepad")).toBeInTheDocument();
  expect(screen.queryByText("the analyzer")).toBeNull();
});
