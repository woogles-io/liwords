import { act, cleanup, render } from "@testing-library/react";
import { BoardPanel } from "./board_panel";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { CrosswordGameGridLayout } from "../constants/board_layout";
import { Board } from "../utils/cwgame/board";
import { PlayerInfoSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { BrowserRouter } from "react-router";
import { waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";

function renderBoardPanel() {
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
