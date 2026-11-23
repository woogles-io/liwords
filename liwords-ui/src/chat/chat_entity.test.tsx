import React, { useEffect } from "react";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, expect, it, vi } from "vitest";
import { ActionType } from "../actions/actions";
import { ChatEntity } from "./chat_entity";
import { reportMessage } from "./report_message";
import { ChatEntityType } from "../store/constants";
import {
  Store,
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
} from "../store/store";

window.RUNTIME_CONFIGURATION = {};

vi.mock("./report_message", () => ({ reportMessage: vi.fn() }));

// A chat entity draws nothing until the blocked players are known.
const BlocksKnown = (props: { children: React.ReactNode }) => {
  const { setExcludedPlayersFetched } = useExcludedPlayersStoreContext();
  useEffect(() => {
    setExcludedPlayersFetched(true);
  }, [setExcludedPlayersFetched]);
  return <React.Fragment>{props.children}</React.Fragment>;
};

// Signs in as me-id.
const SignedIn = () => {
  const { dispatchLoginState } = useLoginStateStoreContext();
  useEffect(() => {
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
  return null;
};

afterEach(() => {
  cleanup();
  document.getElementById("root")?.remove();
});

it("reports another person's message from their name's menu", async () => {
  // The menu renders into #root.
  const root = document.createElement("div");
  root.id = "root";
  document.body.appendChild(root);
  render(
    <MemoryRouter>
      <Store>
        <SignedIn />
        <BlocksKnown>
          <ChatEntity
            entityType={ChatEntityType.UserChat}
            sender="cesar"
            senderId="cesar-id"
            channel="chat.game.abc"
            msgID="m-1"
            message="something rude"
            timestamp={BigInt(1000)}
            highlight={false}
          />
        </BlocksKnown>
      </Store>
    </MemoryRouter>,
  );
  await userEvent.click(await screen.findByText("cesar"));
  await userEvent.click(await screen.findByText("Report this message"));
  expect(reportMessage).toHaveBeenCalledWith(
    expect.objectContaining({
      modal: expect.anything(),
      message: expect.anything(),
    }),
    expect.anything(),
    {
      id: "m-1",
      sender: "cesar",
      message: "something rude",
    },
  );
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
