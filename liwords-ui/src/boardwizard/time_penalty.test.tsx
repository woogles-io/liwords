import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { EndOfGameModal, EndOfGamePlayer } from "./time_penalty";

afterEach(cleanup);

const fresh: EndOfGamePlayer[] = [
  { name: "cesar", score: 412, penalty: 0 },
  { name: "josh", score: 388, penalty: 0 },
];

const renderModal = (
  players = fresh,
  onSubmit = vi.fn(),
  onClose = vi.fn(),
) => {
  const utils = render(
    <EndOfGameModal
      open
      players={players}
      onClose={onClose}
      onSubmit={onSubmit}
    />,
  );
  const press = (name: string, times = 1) => {
    const btn = utils.getByLabelText(`Time penalty for ${name}`);
    for (let i = 0; i < times; i++) {
      fireEvent.click(btn);
    }
  };
  const applyButton = () => utils.getByText("Apply").closest("button")!;
  return { ...utils, press, applyButton, onSubmit, onClose };
};

it("shows each player's score and the winner", () => {
  const { getByText } = renderModal();
  expect(getByText("cesar")).toBeTruthy();
  expect(getByText("412")).toBeTruthy();
  expect(getByText("388")).toBeTruthy();
  expect(getByText("WINNER!")).toBeTruthy();
});

it("cannot apply with nothing changed", () => {
  const { applyButton } = renderModal();
  expect(applyButton().disabled).toBe(true);
});

it("each press deducts ten more and the button shows the next total", () => {
  const { press, getByLabelText, getByText } = renderModal();
  const btn = getByLabelText("Time penalty for josh");
  expect(btn.textContent).toBe("-10");
  press("josh", 2);
  expect(btn.textContent).toBe("-30");
  expect(getByText("368")).toBeTruthy();
});

it("sends the total for only the player whose penalty changed", async () => {
  const { press, applyButton, onSubmit, onClose } = renderModal();
  press("josh", 2);
  fireEvent.click(applyButton());
  await waitFor(() => expect(onClose).toHaveBeenCalled());
  expect(onSubmit).toHaveBeenCalledTimes(1);
  expect(onSubmit).toHaveBeenCalledWith(1, 20);
});

it("penalizes both players", async () => {
  const { press, applyButton, onSubmit } = renderModal();
  press("cesar");
  press("josh", 3);
  fireEvent.click(applyButton());
  await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(2));
  expect(onSubmit).toHaveBeenNthCalledWith(1, 0, 10);
  expect(onSubmit).toHaveBeenNthCalledWith(2, 1, 30);
});

it("starts from an existing penalty and can remove it", async () => {
  const players = [
    { name: "cesar", score: 392, penalty: 20 },
    { name: "josh", score: 400, penalty: 0 },
  ];
  const { getByLabelText, getByText, applyButton, onSubmit } =
    renderModal(players);
  expect(getByLabelText("Time penalty for cesar").textContent).toBe("-30");
  fireEvent.click(getByLabelText("Clear time penalty for cesar"));
  expect(getByText("412")).toBeTruthy();
  fireEvent.click(applyButton());
  await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(0, 0));
});

it("flips the winner when a penalty is large enough", () => {
  const { press, getAllByText } = renderModal();
  press("cesar", 3);
  expect(getAllByText("WINNER!")).toHaveLength(1);
  expect(getAllByText("382")).toHaveLength(1);
});

it("stays open when a penalty fails to send", async () => {
  const onSubmit = vi.fn().mockRejectedValue(new Error("nope"));
  const { press, applyButton, onClose } = renderModal(fresh, onSubmit);
  press("cesar");
  fireEvent.click(applyButton());
  await waitFor(() => expect(onSubmit).toHaveBeenCalled());
  expect(onClose).not.toHaveBeenCalled();
});

it("keeps its edits when the scores update, and starts afresh when remounted", () => {
  const onSubmit = vi.fn();
  const onClose = vi.fn();
  const modal = (key: number, players: EndOfGamePlayer[]) => (
    <EndOfGameModal
      key={key}
      open
      players={players}
      onClose={onClose}
      onSubmit={onSubmit}
    />
  );
  const { getByLabelText, rerender } = render(modal(1, fresh));
  const josh = () => getByLabelText("Time penalty for josh");
  fireEvent.click(josh());
  fireEvent.click(josh());
  expect(josh().textContent).toBe("-30");
  // Another penalty of 10 lands while the modal is open: the edit is kept.
  const applied = [fresh[0], { name: "josh", score: 378, penalty: 10 }];
  rerender(modal(1, applied));
  expect(josh().textContent).toBe("-30");
  // Reopened (new key): it starts from the penalty actually applied.
  rerender(modal(2, applied));
  expect(josh().textContent).toBe("-20");
});
