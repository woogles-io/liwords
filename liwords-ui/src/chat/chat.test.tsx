import React from "react";
import { cleanup, render } from "@testing-library/react";
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
