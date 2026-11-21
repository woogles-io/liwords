import React from "react";
import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { ChatMessageSchema } from "../gen/api/proto/ipc/chat_pb";
import { ChannelChats } from "./chat_review";

// one client for the page's life, as useClient's memo gives
const { getChannelChats, client } = vi.hoisted(() => {
  const getChannelChats = vi.fn();
  return { getChannelChats, client: { getChannelChats } };
});

vi.mock("../utils/hooks/connect", () => ({
  useClient: () => client,
  flashError: vi.fn(),
}));

const msg = (n: number) =>
  create(ChatMessageSchema, {
    id: `id${n}`,
    channel: "chat.game.g1",
    username: "cesar",
    message: `message ${n}`,
    timestamp: BigInt(n * 1000),
  });

afterEach(() => {
  cleanup();
  getChannelChats.mockReset();
});

it("loads the newest page, then each older page above it", async () => {
  // the newest page, then the one older than its first message
  getChannelChats.mockImplementation(({ before }: { before: string }) =>
    Promise.resolve(
      before === ""
        ? { messages: [msg(3), msg(4)], hasOlder: true }
        : { messages: [msg(1), msg(2)], hasOlder: false },
    ),
  );
  render(<ChannelChats channel="chat.game.g1" />);
  fireEvent.click(screen.getByText("Load messages"));
  expect(await screen.findByText("message 4")).toBeInTheDocument();
  expect(screen.queryByText("message 2")).toBeNull();
  expect(
    screen.getByText(/older messages are not loaded yet/i),
  ).toBeInTheDocument();

  fireEvent.click(screen.getByText("Load older"));
  expect(await screen.findByText("message 1")).toBeInTheDocument();
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    before: "id3",
  });
  const shown = screen.getAllByText(/^message \d$/).map((el) => el.textContent);
  expect(shown).toEqual(["message 1", "message 2", "message 3", "message 4"]);
  // all of it is loaded
  expect(screen.queryByText("Load older")).toBeNull();
  expect(screen.queryByText(/older messages are not loaded yet/i)).toBeNull();
});
