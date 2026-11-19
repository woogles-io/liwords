import React from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TokenLinkPage } from "./token_link_page";

// The top bar needs the whole app's stores; it isn't what is tested here.
vi.mock("../navigation/topbar", () => ({ TopBar: () => null }));

const renderPage = (url: string, confirm: (token: string) => Promise<string>) =>
  render(
    <React.StrictMode>
      <MemoryRouter initialEntries={[url]}>
        <TokenLinkPage
          confirm={confirm}
          loadingText="Working..."
          successTitle="Done"
          successSubTitle={(result) => `Result: ${result}`}
          successExtra={[]}
          errorTitle="Failed"
          errorSubTitle={(message) => `Error: ${message}`}
          errorExtra={() => []}
        />
      </MemoryRouter>
    </React.StrictMode>,
  );

afterEach(cleanup);

describe("TokenLinkPage", () => {
  // Some tokens work only once, so a second call, as StrictMode would make
  // by running the effect twice, would turn a success into a failure.
  it("confirms the token once, even in StrictMode", async () => {
    const confirm = vi.fn().mockResolvedValue("new@example.com");
    renderPage("/x?token=abc", confirm);
    expect(screen.getByText("Working...")).toBeTruthy();
    expect(await screen.findByText("Result: new@example.com")).toBeTruthy();
    expect(confirm).toHaveBeenCalledTimes(1);
    expect(confirm).toHaveBeenCalledWith("abc");
  });

  it("shows why the token was refused", async () => {
    const confirm = vi
      .fn()
      .mockRejectedValue(
        new ConnectError("link expired", Code.InvalidArgument),
      );
    renderPage("/x?token=abc", confirm);
    expect(await screen.findByText("Failed")).toBeTruthy();
    expect(screen.getByText("Error: link expired")).toBeTruthy();
  });

  it("says so when the link has no token, without calling the server", () => {
    const confirm = vi.fn();
    renderPage("/x", confirm);
    expect(
      screen.getByText("Error: This link has no token in it."),
    ).toBeTruthy();
    expect(confirm).not.toHaveBeenCalled();
  });
});
