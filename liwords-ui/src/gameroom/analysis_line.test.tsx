import { create } from "@bufbuild/protobuf";
import { act, cleanup, render, renderHook } from "@testing-library/react";
import { useEffect } from "react";
import { BrowserRouter } from "react-router";
import { afterEach, expect, it, vi } from "vitest";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { PlayerInfoSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { Store, useTentativeTileContext } from "../store/store";
import { Board } from "../utils/cwgame/board";
import { AnalyzerMove, usePlaceMoveCallback } from "./analyzer";
import { BoardPanel } from "./board_panel";

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: () => ({ data: undefined }),
}));

window.RUNTIME_CONFIGURATION = {};

afterEach(cleanup);

// A move laying tiles (wolges letters, 0 for a square played through) from
// a square onward.
const placement = (
  row: number,
  col: number,
  vertical: boolean,
  tiles: Array<number>,
): AnalyzerMove => ({
  jsonKey: "",
  displayMove: "",
  coordinates: "",
  leave: [],
  leaveWithGaps: [],
  score: 10,
  equity: 0,
  row,
  col,
  vertical,
  tiles,
  isExchange: false,
});

const pass: AnalyzerMove = {
  ...placement(0, 0, false, []),
  score: 0,
  isExchange: true,
};

// A and then B, which plays through A, on the empty board; C goes after them.
const moveA = placement(7, 7, false, [1, -2]);
const moveB = placement(6, 8, true, [3, 0, 4]);
const moveC = placement(9, 8, false, [5]);

const usePlacing = () => ({
  placeMove: usePlaceMoveCallback(),
  ...useTentativeTileContext(),
});

it("places a move in a line on the board its earlier moves make", () => {
  const { result } = renderHook(usePlacing, { wrapper: Store });
  act(() => {
    result.current.placeMove(moveC, [moveA, moveB]);
  });
  const { placedTiles, placedTilesBoard } = result.current;
  expect(Array.from(placedTiles)).toEqual([{ row: 9, col: 8, letter: 5 }]);
  expect(placedTilesBoard?.placedTiles).toBe(placedTiles);
  const letters = placedTilesBoard!.letters;
  const at = (row: number, col: number) => letters[row * 15 + col];
  // A's blank is in the game's own form.
  expect([at(7, 7), at(7, 8), at(6, 8), at(8, 8)]).toEqual([1, 0x82, 3, 4]);
  expect(letters.filter((l) => l !== 0)).toHaveLength(4);
  // B is the last played, and only where it laid a tile.
  expect(placedTilesBoard!.lastPlayedTiles).toEqual({
    R6C8: true,
    R8C8: true,
  });
  expect(placedTilesBoard!.playerOfTileAt).toEqual({
    R7C7: 0,
    R7C8: 0,
    R6C8: 1,
    R8C8: 1,
  });
});

it("keeps the last placement as the last played across a pass", () => {
  const { result } = renderHook(usePlacing, { wrapper: Store });
  act(() => {
    result.current.placeMove(moveC, [moveA, pass]);
  });
  const { placedTilesBoard } = result.current;
  expect(placedTilesBoard!.lastPlayedTiles).toEqual({
    R7C7: true,
    R7C8: true,
  });
});

it("places a move alone on the game's own board", () => {
  const { result } = renderHook(usePlacing, { wrapper: Store });
  act(() => {
    result.current.placeMove(moveB, [moveA]);
  });
  expect(result.current.placedTilesBoard).toBeDefined();
  act(() => {
    result.current.placeMove(moveA);
  });
  expect(result.current.placedTilesBoard).toBeUndefined();
});

// What a component in the store sees, kept by Probe.
let placing: ReturnType<typeof usePlacing> | undefined;
const Probe = () => {
  const current = usePlacing();
  useEffect(() => {
    placing = current;
  }, [current]);
  return null;
};

it("draws a line's earlier moves until the placed tiles change", () => {
  const { container } = render(
    <Store>
      <BrowserRouter>
        <BoardPanel
          anonymousViewer={false}
          username="cesar"
          currentRack={[]}
          events={[]}
          gameID={"abcdef"}
          challengeRule={ChallengeRule.DOUBLE}
          board={new Board()}
          sendSocketMsg={() => {}}
          sendGameplayEvent={() => {}}
          gameDone={true}
          playerMeta={[
            create(PlayerInfoSchema, { userId: "cesarid", nickname: "cesar" }),
            create(PlayerInfoSchema, { userId: "oppid", nickname: "opp" }),
          ]}
          lexicon="NWL20"
          alphabet={StandardEnglishAlphabet}
          handleAcceptRematch={() => {}}
          handleAcceptAbort={() => {}}
          vsBot={false}
        />
        <Probe />
      </BrowserRouter>
    </Store>,
  );
  const tile = (row: number, col: number) =>
    container
      .querySelector(".tiles")!
      .children[row * 15 + col].querySelector(".tile");
  act(() => {
    placing!.placeMove(moveB, [moveA]);
  });
  // A is played, and the last played; B is placed.
  for (const [row, col] of [
    [7, 7],
    [7, 8],
  ]) {
    expect(tile(row, col)?.className).toMatch(/\blast-played\b/);
    expect(tile(row, col)?.className).not.toMatch(/\btentative\b/);
    expect(tile(row, col)?.className).toMatch(/\btile-p0\b/);
  }
  for (const [row, col] of [
    [6, 8],
    [8, 8],
  ]) {
    expect(tile(row, col)?.className).toMatch(/\btentative\b/);
  }
  act(() => {
    placing!.setPlacedTiles(new Set());
  });
  expect(tile(7, 7)).toBeNull();
  expect(tile(7, 8)).toBeNull();
});
