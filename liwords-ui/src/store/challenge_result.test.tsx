import { create } from "@bufbuild/protobuf";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ServerChallengeResultEventSchema } from "../gen/api/proto/ipc/omgwords_pb";
import {
  Store,
  useChallengeResultEventStoreContext,
  useChatStoreContext,
} from "./store";

window.RUNTIME_CONFIGURATION = {};

describe("a challenge result", () => {
  it("adds no line when the server posted it in the chat", () => {
    const { result } = renderHook(
      () => ({
        ...useChatStoreContext(),
        ...useChallengeResultEventStoreContext(),
      }),
      { wrapper: Store },
    );
    act(() => {
      result.current.challengeResultEvent(
        create(ServerChallengeResultEventSchema, {
          valid: true,
          postedToChat: true,
        }),
      );
      result.current.challengeResultEvent(
        create(ServerChallengeResultEventSchema, {
          valid: false,
          returnedTiles: "AB",
          postedToChat: true,
        }),
      );
    });
    expect(result.current.chat).toEqual([]);
    // a server that does not post it still gets the line from the page
    act(() => {
      result.current.challengeResultEvent(
        create(ServerChallengeResultEventSchema, {
          valid: true,
        }),
      );
      result.current.challengeResultEvent(
        create(ServerChallengeResultEventSchema, {
          valid: false,
          returnedTiles: "AB",
        }),
      );
    });
    expect(result.current.chat.map((c) => c.message)).toEqual([
      "Challenged play was valid",
      "Play was challenged off the board! Returned tiles: AB",
    ]);
  });
});
