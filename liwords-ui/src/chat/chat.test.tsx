import React from "react";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { Chat, Props } from "./chat";
import socializeClient from "../__mocks__/socialize_client";
import {
  Store,
  useChatStoreContext,
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from "../store/store";
import { ActionType } from "../actions/actions";
import { chatMessageToChatEntity } from "../store/constants";

window.RUNTIME_CONFIGURATION = {};

vi.mock("../utils/hooks/connect", async () => {
  const mod = await vi.importActual("../utils/hooks/connect");
  return {
    ...mod,
    // replace some exports
    useClient: () => socializeClient,
  };
});

vi.mock("../collections/useCollectionContext", () => ({
  useCollectionContext: () => null, // Return null for tests (no collection context)
}));

function renderChat(props: Partial<Props> = {}) {
  const dummyFunction = () => {
    return;
  };
  const defaultProps: Props = {
    defaultChannel: "lobby",
    defaultDescription: "description",
    sendChat: dummyFunction,
    suppressDefault: false,
  };
  return render(<Chat {...defaultProps} {...props} />);
}

afterEach(() => {
  socializeClient.getActiveChatChannels.mockClear();
  socializeClient.getChatsForChannel.mockClear();
  cleanup();
});

it("renders the default description", async () => {
  socializeClient.getChatsForChannel.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: "other",
    defaultDescription: "testDescription",
  });
  const title = await findByTestId("description");
  expect(title).toContainHTML("testDescription");
});

it("renders an appropriate in game description", async () => {
  socializeClient.getChatsForChannel.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: "chat.game.123",
    defaultDescription: "Will",
  });
  const title = await findByTestId("description");
  expect(title).toContainHTML("Game chat with Will");
});

it("renders an appropriate gameTV description", async () => {
  socializeClient.getChatsForChannel.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: "chat.gametv.12345",
    defaultDescription: "Will vs. César",
  });
  const title = await findByTestId("description");
  expect(title).toContainHTML("Game chat for Will vs. César");
});

// A chat line shows only once the block list is known.
const BlockListKnown = () => {
  const { setExcludedPlayersFetched } = useExcludedPlayersStoreContext();
  React.useEffect(() => {
    setExcludedPlayersFetched(true);
  }, [setExcludedPlayersFetched]);
  return null;
};

it("tilts the first player's initial in the game's chat", async () => {
  socializeClient.getChatsForChannel.mockResolvedValueOnce({
    messages: [
      {
        id: "m1",
        username: "Will",
        userId: "will-id",
        channel: "chat.game.123",
        message: "good luck",
        timestamp: BigInt(Date.now()),
      },
      {
        id: "m2",
        username: "Cesar",
        userId: "cesar-id",
        channel: "chat.game.123",
        message: "you too",
        timestamp: BigInt(Date.now()),
      },
    ],
  });
  // the chat's lines live in the store
  const { findByText } = render(
    <Store>
      <BlockListKnown />
      <Chat
        defaultChannel="chat.game.123"
        defaultDescription="Cesar"
        sendChat={() => {}}
        firstPlayerID="will-id"
      />
    </Store>,
  );
  const avatarOf = async (text: string) =>
    (await findByText(text))
      .closest(".chat-entity")
      ?.querySelector<HTMLElement>(".player-avatar");
  expect((await avatarOf("good luck"))?.style.transform).toBe("rotate(-10deg)");
  expect((await avatarOf("you too"))?.style.transform).toBe("");
});

describe("the chat collapsed to its tabs and its latest line", () => {
  beforeEach(() => {
    socializeClient.getChatsForChannel.mockResolvedValue({
      data: { messages: [] },
    });
  });
  afterEach(() => {
    localStorage.removeItem("chatCollapsed");
  });
  const card = (container: HTMLElement) => container.querySelector("#chat");

  it("collapses where it may, the choice kept, and expands again", async () => {
    const { container, findByRole } = renderChat({ collapsible: true });
    fireEvent.click(await findByRole("button", { name: "Collapse the chat" }));
    expect(card(container)).toHaveClass("collapsed");
    expect(localStorage.getItem("chatCollapsed")).toBe("true");
    fireEvent.click(await findByRole("button", { name: "Expand the chat" }));
    expect(card(container)).not.toHaveClass("collapsed");
    expect(localStorage.getItem("chatCollapsed")).toBe("false");
  });

  it("stays expanded where it may not collapse, whatever was chosen", async () => {
    localStorage.setItem("chatCollapsed", "true");
    const { container, findByText, queryByRole } = renderChat({});
    await findByText("Players");
    expect(card(container)).not.toHaveClass("collapsed");
    expect(queryByRole("button", { name: /the chat$/ })).toBeNull();
  });

  it("keeps its latest line in sight", async () => {
    localStorage.setItem("chatCollapsed", "true");
    socializeClient.getChatsForChannel.mockResolvedValue({
      messages: [
        {
          id: "m1",
          username: "Will",
          userId: "will-id",
          channel: "chat.game.456",
          message: "good luck",
          timestamp: BigInt(Date.now() - 1000),
        },
        {
          id: "m2",
          username: "Cesar",
          userId: "cesar-id",
          channel: "chat.game.456",
          message: "you too",
          timestamp: BigInt(Date.now()),
        },
      ],
    });
    const { container } = render(
      <Store>
        <BlockListKnown />
        <Chat
          defaultChannel="chat.game.456"
          defaultDescription="Cesar"
          sendChat={() => {}}
          collapsible
        />
      </Store>,
    );
    await waitFor(() =>
      expect(container.querySelector(".chat-latest")?.textContent).toBe(
        "Cesar: you too",
      ),
    );
  });

  it("expands again as a tab is clicked", async () => {
    localStorage.setItem("chatCollapsed", "true");
    const { container, findByRole } = renderChat({ collapsible: true });
    expect(card(container)).toHaveClass("collapsed");
    fireEvent.click(await findByRole("tab", { name: "Chat" }));
    expect(card(container)).not.toHaveClass("collapsed");
  });
});

describe("a channel's history, a page at a time", () => {
  // UUIDv7-shaped IDs that sort as their numbers do.
  const id = (n: number) =>
    `0190a000-0000-7000-8000-${n.toString().padStart(12, "0")}`;
  const msg = (n: number, userId = "zed-id") => ({
    id: id(n),
    username: userId === "me-id" ? "Me" : "Zed",
    userId,
    channel: "chat.game.789",
    message: `message ${n}`,
    timestamp: BigInt(n),
  });

  // Logs in, then adds a live message on request.
  let addLive: (n: number) => void = () => {};
  let addLiveIn: (m: ReturnType<typeof msg>) => void = () => {};
  const Harness = () => {
    const { dispatchLoginState } = useLoginStateStoreContext();
    const { addChat } = useChatStoreContext();
    React.useEffect(() => {
      dispatchLoginState({
        actionType: ActionType.SetAuthentication,
        payload: {
          username: "Me",
          userID: "me-id",
          loggedIn: true,
          connID: "c",
        },
      });
    }, [dispatchLoginState]);
    React.useEffect(() => {
      addLive = (n: number) =>
        addChat(chatMessageToChatEntity(msg(n) as never));
      addLiveIn = (m) => addChat(chatMessageToChatEntity(m as never));
    }, [addChat]);
    return null;
  };
  const renderGame = () =>
    render(
      <Store>
        <BlockListKnown />
        <Harness />
        <Chat
          defaultChannel="chat.game.789"
          defaultDescription="Zed"
          sendChat={() => {}}
        />
      </Store>,
    );

  it("opens at a line before the first unread message", async () => {
    socializeClient.getChatsForChannel.mockResolvedValue({
      messages: [msg(1), msg(2, "me-id"), msg(3), msg(4)],
      hasOlder: false,
      hasNewer: false,
      lastReadId: id(2),
    });
    const { findByText, container } = renderGame();
    await findByText("message 4");
    const line = container.querySelector(".new-messages-line");
    expect(line?.textContent).toBe("New messages");
    expect(line?.nextElementSibling?.textContent).toContain("message 3");
    expect(socializeClient.getChatsForChannel).toHaveBeenCalledWith(
      expect.objectContaining({ aroundLastRead: true, limit: 100 }),
    );
  });

  it("holds live messages back while older history is shown", async () => {
    socializeClient.getChatsForChannel.mockResolvedValueOnce({
      messages: [msg(1), msg(2)],
      hasOlder: false,
      hasNewer: true,
      lastReadId: id(1),
    });
    const { findByText, queryByText, findByRole } = renderGame();
    await findByText("message 2");
    expect(await findByRole("button", { name: /Jump to latest/ })).toBeTruthy();
    React.act(() => addLive(9));
    expect(
      await findByRole("button", { name: /1 new message: jump to latest/ }),
    ).toBeTruthy();
    expect(queryByText("message 9")).toBeNull();

    socializeClient.getChatsForChannel.mockResolvedValueOnce({
      messages: [msg(8), msg(9)],
      hasOlder: true,
      hasNewer: false,
      lastReadId: id(1),
    });
    fireEvent.click(await findByRole("button", { name: /jump to latest/ }));
    await findByText("message 9");
    expect(socializeClient.getChatsForChannel).toHaveBeenLastCalledWith({
      channel: "chat.game.789",
      limit: 100,
    });
    await waitFor(() =>
      expect(document.querySelector(".jump-to-latest")).toBeNull(),
    );
  });

  it("counts conversations, not messages, on the Chat tab", async () => {
    // The Players tab watches its rows scroll into view.
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    );
    socializeClient.getChatsForChannel.mockResolvedValue({
      messages: [msg(1)],
      hasOlder: false,
      hasNewer: false,
      lastReadId: id(1),
    });
    const { findByText, findByRole, container } = renderGame();
    await findByText("message 1");
    fireEvent.click(await findByRole("tab", { name: "Players" }));
    const badge = () =>
      container.querySelector(".notification")?.textContent ?? "";
    const live = (n: number, channel: string, userId = "zed-id") =>
      React.act(() =>
        addLiveIn({
          ...msg(n, userId),
          channel,
          timestamp: BigInt(Date.now() + n),
        }),
      );
    live(10, "chat.game.789");
    live(11, "chat.game.789");
    await waitFor(() => expect(badge()).toBe("1"));
    live(12, "chat.pm.me-id_zed-id");
    await waitFor(() => expect(badge()).toBe("2"));
    live(13, "chat.pm.me-id_zed-id", "me-id"); // the user's own: not counted
    live(14, "chat.pm.me-id_zed-id");
    await waitFor(() => expect(badge()).toBe("2"));
    fireEvent.click(await findByRole("tab", { name: /Chat/ }));
    await waitFor(() => expect(badge()).toBe(""));
    vi.unstubAllGlobals();
  });
});
