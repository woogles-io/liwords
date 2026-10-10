import { create } from "@bufbuild/protobuf";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GameEventSchema } from "../../gen/api/proto/vendored/macondo/macondo_pb";
import type { ChatEntityObj } from "../../store/constants";
import type { GameState } from "../../store/reducers/game_reducer";
import { useDefinitionAndPhonyChecker } from "./definitions";

// The hook's only network call. An unknown lexicon is an error, the way the
// real service answers one, so a request made before the lexicon is known
// cannot stand in for the one made after.
const mock = vi.hoisted(() => ({ defineWords: vi.fn() }));

vi.mock("./connect", () => ({
  useClient: () => ({ defineWords: mock.defineWords }),
}));

const PHONY = "UNCENSOR";

// Only turns are read off the game context.
const gameContext = {
  turns: [
    create(GameEventSchema, { wordsFormed: ["KOOLAHS", "FETTS"] }),
    create(GameEventSchema, { wordsFormed: [PHONY] }),
  ],
} as unknown as GameState;

type CheckerProps = {
  chatGeneration?: number;
  inGameChat?: boolean;
  lexicon: string;
};

const renderChecker = (
  addChat: (chat: ChatEntityObj) => void,
  lexicon: string,
) => {
  const initialProps: CheckerProps = { lexicon };
  return renderHook(
    ({ chatGeneration, inGameChat, lexicon }: CheckerProps) =>
      useDefinitionAndPhonyChecker({
        addChat,
        chatGeneration,
        enableHoverDefine: true,
        inGameChat,
        gameContext,
        gameDone: true,
        gameID: "annogame",
        lexicon,
        variant: undefined,
      }),
    { initialProps },
  );
};

describe("the phony checker", () => {
  beforeEach(() => {
    mock.defineWords.mockReset();
    mock.defineWords.mockImplementation(
      async ({ lexicon, words }: { lexicon: string; words: string[] }) => {
        if (!lexicon) throw new Error("unknown lexicon");
        return {
          results: Object.fromEntries(
            words.map((word) => [word, { v: word !== PHONY, d: "" }]),
          ),
        };
      },
    );
  });

  it("reports phonies when the lexicon is known from the start", async () => {
    const addChat = vi.fn();
    renderChecker(addChat, "CSW24");
    await waitFor(() => expect(addChat).toHaveBeenCalled());
    expect(addChat.mock.calls[0][0].message).toContain(`${PHONY}*`);
  });

  it("reports phonies when the lexicon arrives after the game", async () => {
    // The metadata carrying the lexicon can land after the game document, which
    // is what an annotated game does in production: the checker is set up
    // against an empty lexicon, then told the real one.
    const addChat = vi.fn();
    const { rerender } = renderChecker(addChat, "");
    await act(async () => {
      rerender({ lexicon: "CSW24" });
    });
    await waitFor(() => expect(addChat).toHaveBeenCalled());
    expect(addChat.mock.calls[0][0].message).toContain(`${PHONY}*`);
  });

  it("posts the report again when a channel load replaces the chat", async () => {
    // The chat's history request can land after the report is posted, and a
    // channel load replaces the whole chat list, dropping the report.
    const addChat = vi.fn();
    const { rerender } = renderChecker(addChat, "CSW24");
    await waitFor(() => expect(addChat).toHaveBeenCalledTimes(1));
    const { id, message } = addChat.mock.calls[0][0];

    // Control: with the chat left alone, the report is not repeated.
    await act(async () => {
      rerender({ lexicon: "CSW24" });
    });
    expect(addChat).toHaveBeenCalledTimes(1);

    // The same id, so the store skips it if the report survived the load.
    await act(async () => {
      rerender({ chatGeneration: 1, lexicon: "CSW24" });
    });
    expect(addChat).toHaveBeenCalledTimes(2);
    expect(addChat.mock.calls[1][0]).toMatchObject({ id, message });
    expect(id).toBeTruthy();
  });

  it("waits for the game chat before posting", async () => {
    // Before the game chat has loaded (the list may still hold the lobby's),
    // the report would land in the wrong place, or be wiped by the load.
    const addChat = vi.fn();
    const { rerender } = renderChecker(addChat, "CSW24");
    await act(async () => {
      rerender({ inGameChat: false, lexicon: "CSW24" });
    });
    await waitFor(() => expect(mock.defineWords).toHaveBeenCalled());
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });
    expect(addChat).not.toHaveBeenCalled();
    await act(async () => {
      rerender({ chatGeneration: 1, inGameChat: true, lexicon: "CSW24" });
    });
    expect(addChat).toHaveBeenCalledTimes(1);
  });

  it("posts nothing into another chat, and again on coming back", async () => {
    // A direct-message channel opened on the game page reloads the list too.
    const addChat = vi.fn();
    const { rerender } = renderChecker(addChat, "CSW24");
    await waitFor(() => expect(addChat).toHaveBeenCalledTimes(1));
    const { id } = addChat.mock.calls[0][0];
    await act(async () => {
      rerender({ chatGeneration: 1, inGameChat: false, lexicon: "CSW24" });
    });
    expect(addChat).toHaveBeenCalledTimes(1);
    await act(async () => {
      rerender({ chatGeneration: 2, inGameChat: true, lexicon: "CSW24" });
    });
    expect(addChat).toHaveBeenCalledTimes(2);
    expect(addChat.mock.calls[1][0].id).toBe(id);
  });
});
