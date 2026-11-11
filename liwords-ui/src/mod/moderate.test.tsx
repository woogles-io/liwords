import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { message } from "antd";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ChangeEmail, EmailChanges } from "./moderate";

const changeEmail = vi.hoisted(() => vi.fn());
const listEmailChanges = vi.hoisted(() => vi.fn());
const flashError = vi.hoisted(() => vi.fn());
// One client for every render, as the real hook gives.
const client = vi.hoisted(() => ({ changeEmail, listEmailChanges }));

vi.mock("../utils/hooks/connect", () => ({
  flashError,
  useClient: () => client,
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

describe("EmailChanges", () => {
  beforeEach(() => {
    listEmailChanges.mockReset();
    flashError.mockReset();
  });

  it("lists each change with old, new and who", async () => {
    listEmailChanges.mockResolvedValue({
      changes: [
        {
          oldEmail: "old@example.com",
          newEmail: "new@example.com",
          changedByUsername: "admin1",
        },
      ],
    });
    render(<EmailChanges userID="uuid-1" refresh={0} />);
    expect(await screen.findByText("new@example.com")).toBeInTheDocument();
    expect(screen.getByText("old@example.com")).toBeInTheDocument();
    expect(screen.getByText("admin1")).toBeInTheDocument();
    expect(listEmailChanges).toHaveBeenCalledWith({ userId: "uuid-1" });
  });

  it("loads again on a new refresh value", async () => {
    listEmailChanges.mockResolvedValue({ changes: [] });
    const { rerender } = render(<EmailChanges userID="uuid-1" refresh={0} />);
    await waitFor(() => expect(listEmailChanges).toHaveBeenCalledTimes(1));
    rerender(<EmailChanges userID="uuid-1" refresh={1} />);
    await waitFor(() => expect(listEmailChanges).toHaveBeenCalledTimes(2));
  });

  it("shows the server's error when refused", async () => {
    const refusal = new Error("permission denied");
    listEmailChanges.mockRejectedValue(refusal);
    render(<EmailChanges userID="uuid-1" refresh={0} />);
    await waitFor(() => expect(flashError).toHaveBeenCalledWith(refusal));
  });
});
