import React from "react";
import { cleanup, fireEvent, render } from "@testing-library/react";
import GameControls, { Props } from "./game_controls";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";

const mockedUsedNavigate = vi.fn();

vi.mock("react-router", () => ({
  ...(vi.importActual("react-router") as object),
  useNavigate: () => mockedUsedNavigate,
}));

function renderGameControls(props: Partial<Props> = {}) {
  const dummyFunction = () => {
    return;
  };
  const defaultProps: Props = {
    challengeRule: ChallengeRule.FIVE_POINT,
    exchangeAllowed: false,
    isExamining: false,
    finalPassOrChallenge: false,
    myTurn: false,
    observer: false,
    lexicon: "dummy",
    allowAnalysis: true,
    setHandleChallengeShortcut: dummyFunction,
    setHandleNeitherShortcut: dummyFunction,
    setHandlePassShortcut: dummyFunction,
    showExchangeModal: dummyFunction,
    onExportGCG: dummyFunction,
    onPass: dummyFunction,
    onResign: dummyFunction,
    onRecall: dummyFunction,
    onChallenge: dummyFunction,
    onCommit: dummyFunction,
    onExamine: dummyFunction,
    onRematch: dummyFunction,
    onRequestAbort: dummyFunction,
    onNudge: dummyFunction,
    showNudge: false,
    showAbort: false,
    gameEndControls: false,
    showRematch: false,
  };
  return render(<GameControls {...defaultProps} {...props} />);
}

afterEach(cleanup);

it("fires clicks on rematch only once", async () => {
  const onRematch = vi.fn();
  const { findByTestId } = renderGameControls({
    gameEndControls: true,
    onRematch: onRematch,
    showRematch: true,
  });
  const rematchButton = await findByTestId("rematch-button");
  expect(rematchButton).toBeVisible();
  fireEvent.click(rematchButton);
  fireEvent.click(rematchButton);
  expect(onRematch).toHaveBeenCalledTimes(1);
});
