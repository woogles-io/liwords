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

const renderChecker = (
  addChat: (chat: ChatEntityObj) => void,
  lexicon: string,
) =>
  renderHook(
    ({ lexicon }: { lexicon: string }) =>
      useDefinitionAndPhonyChecker({
        addChat,
        enableHoverDefine: true,
        gameContext,
        gameDone: true,
        gameID: "annogame",
        lexicon,
        variant: undefined,
      }),
    { initialProps: { lexicon } },
  );

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
});
