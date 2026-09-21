import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { TimePenaltyControl } from "./time_penalty";

afterEach(cleanup);

it("submits the default ten-point penalty for the first player", async () => {
  const onSubmit = vi.fn();
  const { getByTestId } = render(
    <TimePenaltyControl playerNames={["cesar", "josh"]} onSubmit={onSubmit} />,
  );
  fireEvent.click(getByTestId("time-penalty-submit"));
  await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(0, 10));
});

it("submits the entered points", async () => {
  const onSubmit = vi.fn();
  const { getByLabelText, getByTestId } = render(
    <TimePenaltyControl playerNames={["cesar", "josh"]} onSubmit={onSubmit} />,
  );
  const input = getByLabelText("Time penalty points");
  fireEvent.change(input, { target: { value: "30" } });
  fireEvent.blur(input);
  fireEvent.click(getByTestId("time-penalty-submit"));
  await waitFor(() => expect(onSubmit).toHaveBeenCalledWith(0, 30));
});
