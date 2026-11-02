import React, { useEffect } from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { ChatEntity } from "./chat_entity";
import { ChatEntityType } from "../store/constants";
import { Store, useExcludedPlayersStoreContext } from "../store/store";

window.RUNTIME_CONFIGURATION = {};

// A chat entity draws nothing until the blocked players are known.
const BlocksKnown = (props: { children: React.ReactNode }) => {
  const { setExcludedPlayersFetched } = useExcludedPlayersStoreContext();
  useEffect(() => {
    setExcludedPlayersFetched(true);
  }, [setExcludedPlayersFetched]);
  return <React.Fragment>{props.children}</React.Fragment>;
};

afterEach(() => {
  cleanup();
});

it("shows a line the server posted in a channel as a server message", async () => {
  const { findByText, container } = render(
    <Store>
      <BlocksKnown>
        <ChatEntity
          entityType={ChatEntityType.UserChat}
          sender=""
          senderId=""
          channel="chat.game.abc"
          msgID="1793281202205-0"
          message="Play was challenged off the board! Returned tiles: AB"
          highlight={false}
        />
      </BlocksKnown>
    </Store>,
  );
  const line = await findByText(
    "Play was challenged off the board! Returned tiles: AB",
  );
  expect(line.className).toBe("server-message");
  // no avatar, name, or timestamp
  expect(container.querySelector(".chat-entity")).toBeNull();
});
