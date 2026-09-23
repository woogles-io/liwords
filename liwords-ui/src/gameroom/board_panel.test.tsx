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
