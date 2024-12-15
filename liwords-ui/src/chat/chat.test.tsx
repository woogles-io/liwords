import React from "react";
import { cleanup, render } from "@testing-library/react";
import { Chat, Props } from "./chat";
import socializeClient from "../__mocks__/socialize_client";

window.RUNTIME_CONFIGURATION = {};

vi.mock("../utils/hooks/connect", async () => {
  const mod = await vi.importActual("../utils/hooks/connect");
  return {
    ...mod,
    // replace some exports
    useClient: () => socializeClient,
  };
});

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
