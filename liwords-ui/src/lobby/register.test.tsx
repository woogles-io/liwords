import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Register } from "./register";

// The top bar needs the whole app's stores; it isn't what is tested here.
vi.mock("../navigation/topbar", () => ({ TopBar: () => null }));
vi.mock("../store/store", () => ({
  useLoginStateStoreContext: () => ({ loginState: { loggedIn: false } }),
}));

const mock = vi.hoisted(() => ({ register: vi.fn(), login: vi.fn() }));
vi.mock("../utils/hooks/connect", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../utils/hooks/connect")>()),
  useClient: () => ({ register: mock.register, login: mock.login }),
}));

afterEach(cleanup);

const fill = (placeholder: string, value: string) =>
  fireEvent.change(screen.getByPlaceholderText(placeholder), {
    target: { value },
  });

const fillForm = () => {
  fill("Email", "kid@example.com");
  fill("Username", "somekid");
  fill("Password", "a long enough password");
  fill("Year", "1990");
  fill("Month", "May");
  fill("Date", "5");
  fireEvent.click(screen.getByRole("checkbox"));
};

describe("Register", () => {
  // Each attempt counts against a per-browser sign-up limit.
  it("sends one sign-up at a time, however often Continue is clicked", async () => {
    let refuse: (e: Error) => void = () => {};
    mock.register.mockImplementation(
      () => new Promise((_, reject) => (refuse = reject)),
    );
    render(
      <MemoryRouter>
        <Register />
      </MemoryRouter>,
    );
    fillForm();
    const button = screen.getByRole("button", { name: /continue/i });

    fireEvent.click(button);
    await waitFor(() => expect(mock.register).toHaveBeenCalledTimes(1));
    for (let i = 0; i < 5; i++) fireEvent.click(button);
    fireEvent.submit(button.closest("form")!);
    await new Promise((r) => setTimeout(r, 50));
    expect(mock.register).toHaveBeenCalledTimes(1);

    // Once the server answers, the person can fix the form and try again.
    refuse(new ConnectError("username taken", Code.InvalidArgument));
    expect(await screen.findByText(/username taken/)).toBeTruthy();
    fireEvent.click(button);
    await waitFor(() => expect(mock.register).toHaveBeenCalledTimes(2));
  });
});
