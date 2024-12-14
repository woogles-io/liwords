import { act, cleanup, render } from "@testing-library/react";
import { BoardPanel } from "./board_panel";
import { ChallengeRule } from "../gen/api/vendor/macondo/macondo_pb";
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

it("renders a game board panel", async () => {
  let container: HTMLElement;
  // act so that we can wait for useEffect hooks to run:
  await act(async () => {
    container = renderBoardPanel().container;
  });
  await waitFor(async () => {
    // XXX this test is actually broken. the snapshot is wrong.
    // expect(container.querySelector('.tile-p0')).not.toBeNull();
    // console.log('ok', container.querySelector('.tile-p0'));
    expect(container).toMatchSnapshot();
  });
});
