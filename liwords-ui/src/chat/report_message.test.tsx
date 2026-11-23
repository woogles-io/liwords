import React from "react";
import { App } from "antd";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { reportMessage } from "./report_message";

const { flashError } = vi.hoisted(() => ({ flashError: vi.fn() }));

vi.mock("../utils/hooks/connect", () => ({ flashError }));

// A button that asks to report cesar's message through the given client.
const renderReport = (client: { reportChatMessage: () => Promise<object> }) => {
  const Harness = () => {
    const { modal, message } = App.useApp();
    return (
      <button
        onClick={() =>
          reportMessage({ modal, message }, client as never, {
            id: "m-1",
            sender: "cesar",
            message: "something rude",
          })
        }
      >
        ask
      </button>
    );
  };
  render(
    <App>
      <Harness />
    </App>,
  );
  return userEvent.click(screen.getByText("ask"));
};

afterEach(() => {
  cleanup();
  flashError.mockReset();
});

it("asks first, then sends the report", async () => {
  const reportChatMessage = vi.fn(() => Promise.resolve({}));
  await renderReport({ reportChatMessage });
  await waitFor(() =>
    expect(
      document.querySelector(".ant-modal-confirm-title"),
    ).toHaveTextContent("Report this message to the moderators?"),
  );
  // the message reported, as it was sent, and where to write for more
  expect(
    document.querySelector(".ant-modal-confirm-content p"),
  ).toHaveTextContent("cesar: something rude");
  expect(
    screen.getByRole("link", { name: "conduct@woogles.io" }),
  ).toHaveAttribute("href", "mailto:conduct@woogles.io");
  expect(reportChatMessage).not.toHaveBeenCalled();
  await userEvent.click(screen.getByRole("button", { name: "Report" }));
  expect(reportChatMessage).toHaveBeenCalledWith({ messageId: "m-1" });
  expect(await screen.findByText("Report sent")).toBeInTheDocument();
  expect(flashError).not.toHaveBeenCalled();
});

it("sends nothing when canceled", async () => {
  const reportChatMessage = vi.fn(() => Promise.resolve({}));
  await renderReport({ reportChatMessage });
  await userEvent.click(await screen.findByRole("button", { name: "Cancel" }));
  // let anything the click set off run
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(reportChatMessage).not.toHaveBeenCalled();
});

it("says why a report failed", async () => {
  const refused = new Error("bad message id");
  const reportChatMessage = vi.fn(() => Promise.reject(refused));
  await renderReport({ reportChatMessage });
  await userEvent.click(await screen.findByRole("button", { name: "Report" }));
  await waitFor(() => expect(flashError).toHaveBeenCalledWith(refused));
  expect(screen.queryByText("Report sent")).toBeNull();
});
