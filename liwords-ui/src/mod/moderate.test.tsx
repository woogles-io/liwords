import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { message } from "antd";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ChangeEmail } from "./moderate";

const changeEmail = vi.hoisted(() => vi.fn());
const flashError = vi.hoisted(() => vi.fn());

vi.mock("../utils/hooks/connect", () => ({
  flashError,
  useClient: () => ({ changeEmail }),
}));

const submit = (email: string) => {
  render(<ChangeEmail userID="uuid-1" />);
  fireEvent.change(screen.getByLabelText("New email"), {
    target: { value: email },
  });
  fireEvent.click(screen.getByRole("button", { name: "Change email" }));
};

describe("ChangeEmail", () => {
  beforeEach(() => {
    changeEmail.mockReset();
    flashError.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("sends the new email for the user", async () => {
    const info = vi.spyOn(message, "info");
    changeEmail.mockResolvedValue({});
    submit("new@example.com");
    await waitFor(() =>
      expect(changeEmail).toHaveBeenCalledWith({
        userId: "uuid-1",
        email: "new@example.com",
      }),
    );
    await waitFor(() => expect(info).toHaveBeenCalledTimes(1));
    expect(flashError).not.toHaveBeenCalled();
  });

  it("shows the server's error when refused", async () => {
    const refusal = new Error("that email address is already used");
    changeEmail.mockRejectedValue(refusal);
    submit("taken@example.com");
    await waitFor(() => expect(flashError).toHaveBeenCalledWith(refusal));
  });
});
