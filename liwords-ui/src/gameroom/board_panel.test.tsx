import { cleanup, fireEvent, render } from "@testing-library/react";
import { BoardPanel } from "./board_panel";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { CrosswordGameGridLayout } from "../constants/board_layout";
import { Board } from "../utils/cwgame/board";
import { PlayerInfoSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { BrowserRouter } from "react-router";
import { waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: () => ({ data: undefined }),
}));

function renderBoardPanel(boardEditingMode = false) {
  const dummyFunction = () => {};

  const rack = [0, 1, 5, 9, 14, 19, 20];
  const board = new Board(CrosswordGameGridLayout);
  const playerInfo = [
    create(PlayerInfoSchema, {
      userId: "cesarid",
      nickname: "cesar",
      fullName: "cesar richards",
    }),
    create(PlayerInfoSchema, {
      userId: "oppid",
      nickname: "opp",
      fullName: "opp mcOppface",
    }),
  ];
  return render(
    <BrowserRouter>
      <BoardPanel
        anonymousViewer={false}
        username="cesar"
        currentRack={rack}
        events={[]}
        gameID={"abcdef"}
        challengeRule={ChallengeRule.DOUBLE}
        board={board}
        sendSocketMsg={() => {}}
        sendGameplayEvent={() => {}}
        gameDone={false}
        playerMeta={playerInfo}
        lexicon="NWL20"
        alphabet={StandardEnglishAlphabet}
        handleAcceptRematch={dummyFunction}
        handleAcceptAbort={dummyFunction}
        vsBot={false}
        boardEditingMode={boardEditingMode}
      />
    </BrowserRouter>,
  );
}

afterEach(cleanup);

// skip because snapshot comparison isn't working anymore. it's failing due
// to the auto-generated CSS classes with antdesign.
it.skip("renders a game board panel", async () => {
  // Simplify by combining the rendering and waiting in one step
  const { container } = renderBoardPanel();

  // Wait for any async effects to complete
  await waitFor(() => {
    // Optional: Add a specific condition that indicates the component is fully rendered
    expect(container.querySelector(".board-container")).toBeInTheDocument();
  });

  // Take a single snapshot after the component is stable
  expect(container).toMatchSnapshot();
});

it("opens the rack editor on Space in board editing mode", () => {
  const { container } = renderBoardPanel(true);
  expect(container.querySelector("input.rack")).toBeNull();
  const panel = container.querySelector(".board-container")!;
  fireEvent.keyDown(panel, { key: " " });
  const input = container.querySelector("input.rack") as HTMLInputElement;
  expect(input).toBeTruthy();
  // No previous turn, so the editor opens empty.
  expect(input.value).toBe("");
});

it("does not open the rack editor on Space outside the editor", () => {
  const { container } = renderBoardPanel();
  fireEvent.keyDown(container.querySelector(".board-container")!, {
    key: " ",
  });
  expect(container.querySelector("input.rack")).toBeNull();
});

it("keeps Space advancing the placement arrow in board editing mode", () => {
  const { container } = renderBoardPanel(true);
  const spaces = () => Array.from(container.querySelectorAll(".board-space"));
  fireEvent.click(spaces()[0]);
  const selectedIndex = () =>
    spaces().findIndex((el) => el.classList.contains("selected"));
  expect(selectedIndex()).toBe(0);
  fireEvent.keyDown(container.querySelector(".board-container")!, {
    key: " ",
  });
  expect(container.querySelector("input.rack")).toBeNull();
  expect(selectedIndex()).toBeGreaterThan(0);
});

// Records each spin the board starts, as [from, to] in degrees.
const mockSpins = (playState: AnimationPlayState) => {
  const spins: Array<[string, string]> = [];
  const animate = vi.fn((frames: Array<{ transform: string }>) => {
    spins.push([frames[0].transform, frames[1].transform]);
    return { cancel: vi.fn(), playState } as unknown as Animation;
  });
  Element.prototype.animate = animate as unknown as Element["animate"];
  return spins;
};

it("flips the board clockwise each way on the 6 key", () => {
  const spins = mockSpins("finished");
  const { container } = renderBoardPanel();
  const board = () => container.querySelector(".board")!;
  const press6 = () =>
    fireEvent.keyDown(container.querySelector(".board-container")!, {
      key: "6",
    });
  // Starts upright, with no spin on load.
  expect(board().classList.contains("board-flipped")).toBe(false);
  expect(spins).toEqual([]);
  press6();
  expect(board().classList.contains("board-flipped")).toBe(true);
  expect(spins).toEqual([["rotate(0deg)", "rotate(180deg)"]]);
  press6();
  expect(board().classList.contains("board-flipped")).toBe(false);
  expect(spins[1]).toEqual(["rotate(180deg)", "rotate(360deg)"]);
});

it("carries a flip that arrives mid-spin on round, clockwise", () => {
  const spins = mockSpins("running");
  const { container } = renderBoardPanel();
  const press6 = () =>
    fireEvent.keyDown(container.querySelector(".board-container")!, {
      key: "6",
    });
  press6();
  // 70 degrees into the turn down when the second flip arrives.
  const rad = (70 * Math.PI) / 180;
  const real = window.getComputedStyle;
  vi.spyOn(window, "getComputedStyle").mockImplementation((el) => {
    const style = real(el);
    if (!(el as Element).classList.contains("board")) return style;
    return {
      ...style,
      transform: `matrix(${Math.cos(rad)}, ${Math.sin(rad)}, ${-Math.sin(rad)}, ${Math.cos(rad)}, 0, 0)`,
    } as CSSStyleDeclaration;
  });
  press6();
  vi.restoreAllMocks();
  const [from, to] = spins[1].map((t) => parseFloat(t.slice(7)));
  expect(from).toBeCloseTo(70);
  expect(to).toBeCloseTo(360);
});
