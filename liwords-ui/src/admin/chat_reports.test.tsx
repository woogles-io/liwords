import React from "react";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { ChatMessageSchema } from "../gen/api/proto/ipc/chat_pb";
import { ChatReportSchema } from "../gen/api/proto/mod_service/mod_service_pb";
import { ChatReports } from "./chat_reports";

// one client for the page's life, as useClient's memo gives
const { client } = vi.hoisted(() => ({
  client: {
    getChatReports: vi.fn(),
    handleChatReport: vi.fn(),
    reopenChatReport: vi.fn(),
    getChannelChats: vi.fn(),
  },
}));

vi.mock("../utils/hooks/connect", () => ({
  useClient: () => client,
  flashError: vi.fn(),
}));
// a name, without the menu and the stores it needs
vi.mock("../shared/usernameWithContext", () => ({
  UsernameWithContext: (props: { username: string }) => (
    <span>{props.username}</span>
  ),
}));

const report = (
  id: bigint,
  messageId: string,
  text: string,
  reporter: string,
  deleted = false,
) =>
  create(ChatReportSchema, {
    id,
    message: create(ChatMessageSchema, {
      id: messageId,
      channel: "chat.lobby",
      username: "cesar",
      userId: "cesar-id",
      message: text,
      timestamp: BigInt(1000),
    }),
    deleted,
    reporterId: `${reporter}-id`,
    reporterUsername: reporter,
    reportedAt: timestampFromDate(new Date(2000)),
  });

afterEach(() => {
  cleanup();
  client.getChatReports.mockReset();
  client.handleChatReport.mockReset();
  client.reopenChatReport.mockReset();
  client.getChannelChats.mockReset();
});

it("lists the open reports, each with its message, sender, and reporter", async () => {
  client.getChatReports.mockResolvedValue({
    reports: [
      report(BigInt(2), "m-2", "a rude whisper", "mina", true),
      report(BigInt(1), "m-1", "something rude", "jesse"),
    ],
  });
  render(<ChatReports />);
  expect(await screen.findByText("something rude")).toBeInTheDocument();
  expect(screen.getByText("a rude whisper")).toBeInTheDocument();
  expect(screen.getByText("jesse")).toBeInTheDocument();
  expect(screen.getByText("mina")).toBeInTheDocument();
  expect(screen.getAllByText("cesar")).toHaveLength(2);
  // only the deleted one says so
  expect(screen.getAllByText("deleted")).toHaveLength(1);
});

it("marks a report handled, then lists what is still open", async () => {
  client.getChatReports
    .mockResolvedValueOnce({
      reports: [report(BigInt(1), "m-1", "something rude", "jesse")],
    })
    .mockResolvedValueOnce({ reports: [] });
  client.handleChatReport.mockResolvedValue({});
  render(<ChatReports />);
  fireEvent.click(await screen.findByText("Mark handled"));
  await waitFor(() => expect(screen.queryByText("something rude")).toBeNull());
  expect(client.handleChatReport).toHaveBeenCalledWith({ reportId: BigInt(1) });
  expect(client.getChatReports).toHaveBeenCalledTimes(2);
  expect(client.getChatReports).toHaveBeenLastCalledWith({ handled: false });
});

it("lists the handled reports, with who handled each and when, to reopen", async () => {
  const handled = create(ChatReportSchema, {
    ...report(BigInt(1), "m-1", "something rude", "jesse"),
    handledById: "mina-id",
    handledByUsername: "mina",
    handledAt: timestampFromDate(new Date(Date.UTC(2026, 10, 22, 4, 5, 6))),
  });
  let open = false;
  client.getChatReports.mockImplementation(
    ({ handled: asked }: { handled: boolean }) =>
      Promise.resolve({ reports: asked === open ? [] : [handled] }),
  );
  client.reopenChatReport.mockImplementation(() => {
    open = true;
    return Promise.resolve({});
  });
  render(<ChatReports />);
  // the open list first: none open
  await waitFor(() =>
    expect(client.getChatReports).toHaveBeenCalledWith({ handled: false }),
  );
  fireEvent.click(screen.getByText("Handled"));
  expect(await screen.findByText("something rude")).toBeInTheDocument();
  expect(screen.getByText("mina")).toBeInTheDocument();
  expect(screen.getByText("2026-11-22T04:05:06.000Z")).toBeInTheDocument();
  expect(screen.queryByText("Mark handled")).toBeNull();

  fireEvent.click(screen.getByText("Reopen"));
  await waitFor(() => expect(screen.queryByText("something rude")).toBeNull());
  expect(client.reopenChatReport).toHaveBeenCalledWith({ reportId: BigInt(1) });
  expect(client.getChatReports).toHaveBeenLastCalledWith({ handled: true });
});

it("opens the reported message's channel at the message", async () => {
  client.getChatReports.mockResolvedValue({
    reports: [report(BigInt(1), "m-1", "something rude", "jesse")],
  });
  client.getChannelChats.mockResolvedValue({
    messages: [],
    hasOlder: false,
    hasNewer: false,
  });
  render(<ChatReports />);
  await screen.findByText("something rude");
  fireEvent.click(screen.getByRole("button", { name: "Expand row" }));
  fireEvent.click(await screen.findByText("Load messages"));
  await waitFor(() =>
    expect(client.getChannelChats).toHaveBeenCalledWith({
      channel: "chat.lobby",
      around: "m-1",
    }),
  );
});

it("pages a long list, ten a page as the other admin lists", async () => {
  client.getChatReports.mockResolvedValue({
    reports: Array.from({ length: 12 }, (_, i) =>
      report(BigInt(i + 1), `m-${i + 1}`, `rude ${i + 1}`, "jesse"),
    ),
  });
  render(<ChatReports />);
  expect(await screen.findByText("rude 1")).toBeInTheDocument();
  expect(screen.getByText("rude 10")).toBeInTheDocument();
  expect(screen.queryByText("rude 11")).toBeNull();
  fireEvent.click(screen.getByTitle("2"));
  expect(await screen.findByText("rude 11")).toBeInTheDocument();
  expect(screen.queryByText("rude 1")).toBeNull();
});
