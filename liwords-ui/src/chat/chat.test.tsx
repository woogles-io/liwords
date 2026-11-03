import React from "react";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { Chat, Props } from "./chat";
import socializeClient from "../__mocks__/socialize_client";
import { Store, useExcludedPlayersStoreContext } from "../store/store";

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
