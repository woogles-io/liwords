import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { EndOfGameModal } from "./time_penalty";

afterEach(cleanup);

const players = [
  { name: "cesar", score: 412 },
  { name: "josh", score: 388 },
];

const renderModal = (onSubmit = vi.fn(), onClose = vi.fn()) => {
  const utils = render(
    <EndOfGameModal
      open
      players={players}
      onClose={onClose}
      onSubmit={onSubmit}
    />,
  );
  const setPenalty = (name: string, value: string) => {
    const input = utils.getByLabelText(`Time penalty for ${name}`);
    fireEvent.change(input, { target: { value } });
    fireEvent.blur(input);
  };
  const applyButton = () =>
    utils.getByText("Apply time penalties").closest("button")!;
  return { ...utils, setPenalty, applyButton, onSubmit, onClose };
};

it("shows each player's score", () => {
  const { getByText } = renderModal();
  expect(getByText("412")).toBeTruthy();
  expect(getByText("388")).toBeTruthy();
});

it("cannot apply with no penalty entered", () => {
  const { applyButton } = renderModal();
  expect(applyButton().disabled).toBe(true);
});

it("penalizes only the player with points entered", async () => {
  const { setPenalty, applyButton, onSubmit, onClose } = renderModal();
  setPenalty("josh", "20");
  fireEvent.click(applyButton());
  await waitFor(() => expect(onClose).toHaveBeenCalled());
  expect(onSubmit).toHaveBeenCalledTimes(1);
  expect(onSubmit).toHaveBeenCalledWith(1, 20);
});

it("penalizes both players", async () => {
  const { setPenalty, applyButton, onSubmit } = renderModal();
  setPenalty("cesar", "10");
  setPenalty("josh", "30");
  fireEvent.click(applyButton());
  await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(2));
  expect(onSubmit).toHaveBeenNthCalledWith(1, 0, 10);
  expect(onSubmit).toHaveBeenNthCalledWith(2, 1, 30);
});

it("stays open when a penalty fails to send", async () => {
  const onSubmit = vi.fn().mockRejectedValue(new Error("nope"));
  const { setPenalty, applyButton, onClose } = renderModal(onSubmit);
  setPenalty("cesar", "10");
  fireEvent.click(applyButton());
  await waitFor(() => expect(onSubmit).toHaveBeenCalled());
  expect(onClose).not.toHaveBeenCalled();
});
