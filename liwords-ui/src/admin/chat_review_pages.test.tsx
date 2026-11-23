import React from "react";
import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { ChatMessageSchema } from "../gen/api/proto/ipc/chat_pb";
import { ChannelChats, markWords } from "./chat_review";

// one client for the page's life, as useClient's memo gives
const { getChannelChats, client } = vi.hoisted(() => {
  const getChannelChats = vi.fn();
  return { getChannelChats, client: { getChannelChats } };
});

vi.mock("../utils/hooks/connect", () => ({
  useClient: () => client,
  flashError: vi.fn(),
}));

const msg = (n: number, text = `message ${n}`) =>
  create(ChatMessageSchema, {
    id: `id${n}`,
    channel: "chat.game.g1",
    username: "cesar",
    message: text,
    timestamp: BigInt(n * 1000),
  });

afterEach(() => {
  cleanup();
  getChannelChats.mockReset();
});

it("loads the newest page, then each older page above it", async () => {
  // the newest page, then the one older than its first message
  getChannelChats.mockImplementation(({ before }: { before?: string }) =>
    Promise.resolve(
      !before
        ? { messages: [msg(3), msg(4)], hasOlder: true }
        : { messages: [msg(1), msg(2)], hasOlder: false },
    ),
  );
  render(<ChannelChats channel="chat.game.g1" />);
  fireEvent.click(screen.getByText("Load messages"));
  expect(await screen.findByText("message 4")).toBeInTheDocument();
  expect(screen.queryByText("message 2")).toBeNull();

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
});

// What an export saves: the file's text, caught where the page makes it.
const saved = () => {
  const files: Array<Blob> = [];
  vi.spyOn(window.URL, "createObjectURL").mockImplementation((b) => {
    files.push(b as Blob);
    return "blob:export";
  });
  vi.spyOn(window.URL, "revokeObjectURL").mockImplementation(() => {});
  vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  return files;
};

it("exports the whole channel, whatever is loaded", async () => {
  getChannelChats.mockImplementation(({ before }: { before?: string }) =>
    Promise.resolve(
      !before
        ? { messages: [msg(3), msg(4)], hasOlder: true }
        : { messages: [msg(1), msg(2)], hasOlder: false },
    ),
  );
  const files = saved();
  render(<ChannelChats channel="chat.game.g1" />);
  fireEvent.click(screen.getByText("Load messages"));
  await screen.findByText("message 4");
  // only the newest page is shown; the export fetches the older one itself
  fireEvent.click(screen.getByText("Export JSON"));
  await waitFor(() => expect(files).toHaveLength(1));
  const ids = JSON.parse(await files[0].text()).map(
    (m: { id: string }) => m.id,
  );
  expect(ids).toEqual(["id1", "id2", "id3", "id4"]);
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    before: "id3",
  });
  vi.restoreAllMocks();
});

it("exports every match of a word search, whatever is loaded", async () => {
  getChannelChats.mockImplementation(({ before }: { before?: string }) =>
    Promise.resolve(
      !before
        ? { messages: [msg(7, "a rude one")], hasOlder: true }
        : { messages: [msg(2, "rude too")], hasOlder: false },
    ),
  );
  const files = saved();
  render(<ChannelChats channel="chat.game.g1" containing="rude" />);
  fireEvent.click(screen.getByText("Load messages"));
  // the newest match is shown (its word marked apart), with older ones to load
  await screen.findByText("Load older");
  fireEvent.click(screen.getByText("Export CSV"));
  await waitFor(() => expect(files).toHaveLength(1));
  const csv = await files[0].text();
  expect(csv).toContain("rude too");
  expect(csv.indexOf("rude too")).toBeLessThan(csv.indexOf("a rude one"));
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    before: "id7",
    containing: "rude",
  });
  vi.restoreAllMocks();
});

it("marks the words searched for, in any case", () => {
  const { container } = render(
    <div>{markWords("You are a LOSER, a loser.", "loser")}</div>,
  );
  const marked = [...container.querySelectorAll("mark")].map(
    (m) => m.textContent,
  );
  expect(marked).toEqual(["LOSER", "loser"]);
  expect(container.textContent).toBe("You are a LOSER, a loser.");
  // a word with a regex character in it is only itself
  const { container: c2 } = render(
    <div>{markWords("100% (sure)", "% (s")}</div>,
  );
  expect(c2.querySelector("mark")?.textContent).toBe("% (s");
});

it("shows the matches, and the messages around one on Context", async () => {
  getChannelChats.mockImplementation(
    (req: { containing?: string; around?: string; after?: string }) =>
      Promise.resolve(
        req.containing
          ? {
              messages: [msg(3, "you are a loser")],
              hasOlder: false,
              hasNewer: false,
            }
          : req.around
            ? {
                messages: [msg(2), msg(3, "you are a loser"), msg(4)],
                hasOlder: true,
                hasNewer: true,
              }
            : { messages: [msg(5)], hasOlder: false, hasNewer: false },
      ),
  );
  render(<ChannelChats channel="chat.game.g1" containing="loser" />);
  fireEvent.click(screen.getByText("Load messages"));
  expect(await screen.findByText("loser")).toBeInTheDocument();
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    containing: "loser",
  });
  expect(screen.queryByText("message 2")).toBeNull();

  fireEvent.click(screen.getByText("Context"));
  expect(await screen.findByText("message 2")).toBeInTheDocument();
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    around: "id3",
  });
  expect(screen.getByText("message 4")).toBeInTheDocument();
  // newer ones from there, below
  fireEvent.click(screen.getByText("Load newer"));
  expect(await screen.findByText("message 5")).toBeInTheDocument();
  expect(getChannelChats).toHaveBeenLastCalledWith({
    channel: "chat.game.g1",
    after: "id4",
  });
  const shown = screen.getAllByText(/^message \d$/).map((el) => el.textContent);
  expect(shown).toEqual(["message 2", "message 4", "message 5"]);
  expect(screen.queryByText("Context")).toBeNull();

  fireEvent.click(screen.getByText("Back to the matches"));
  await waitFor(() => expect(screen.queryByText("message 2")).toBeNull());
  expect(screen.getByText("Context")).toBeInTheDocument();
});
