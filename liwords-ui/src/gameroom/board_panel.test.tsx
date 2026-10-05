import { cleanup, render } from "@testing-library/react";
import { BoardPanel } from "./board_panel";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { CrosswordGameGridLayout } from "../constants/board_layout";
import { Board } from "../utils/cwgame/board";
import { PlayerInfoSchema } from "../gen/api/proto/ipc/omgwords_pb";
import { StandardEnglishAlphabet } from "../constants/alphabets";
import { BrowserRouter } from "react-router";
import { waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  GameEventSchema,
  GameEvent_Type,
} from "../gen/api/proto/vendored/macondo/macondo_pb";
import { App } from "antd";

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: () => ({ data: undefined }),
}));

function renderBoardPanel(
  extra: Partial<React.ComponentProps<typeof BoardPanel>> = {},
) {
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
        {...extra}
      />
    </BrowserRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

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

const oppExchange = [
  create(GameEventSchema, {
    type: GameEvent_Type.EXCHANGE,
    playerIndex: 1,
    exchanged: "ABCD",
  }),
];

function spyOnMessages() {
  const info = vi.fn();
  const api = {
    message: { info, error: vi.fn(), success: vi.fn(), warning: vi.fn() },
    notification: { info: vi.fn(), error: vi.fn(), open: vi.fn() },
    modal: {},
  };
  vi.spyOn(App, "useApp").mockReturnValue(
    api as unknown as ReturnType<typeof App.useApp>,
  );
  return info;
}

it("announces an opponent's exchange in a live game", () => {
  const info = spyOnMessages();
  renderBoardPanel({ events: oppExchange });
  expect(info).toHaveBeenCalledWith(
    expect.objectContaining({ content: "opp exchanged ABCD" }),
    3,
    undefined,
  );
});

it("does not announce the last move in an annotated game", () => {
  const info = spyOnMessages();
  renderBoardPanel({ events: oppExchange, annotated: true });
  expect(info).not.toHaveBeenCalledWith(
    expect.objectContaining({ content: expect.stringContaining("exchanged") }),
    expect.anything(),
    undefined,
  );
});
