import { create } from "@bufbuild/protobuf";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ActionType } from "../actions/actions";
import { MatchUserSchema } from "../gen/api/proto/ipc/omgseeks_pb";
import { ChatEntityType } from "./constants";
import { SoughtGame } from "./reducers/lobby_reducer";
import {
  LOBBY_QUEUE_MS,
  Store,
  useChatStoreContext,
  useLobbyStoreContext,
} from "./store";

window.RUNTIME_CONFIGURATION = {};

const serverMsg = (id: string) => ({
  entityType: ChatEntityType.ServerMsg,
  sender: "",
  message: `message ${id}`,
  id,
  channel: "server",
});

const seek = (id: string): SoughtGame => ({
  seeker: "someone",
  lexicon: "CSW24",
  initialTimeSecs: 300,
  incrementSecs: 0,
  maxOvertimeMinutes: 1,
  challengeRule: 0,
  userRating: "1500",
  rated: false,
  seekID: id,
  playerVsBot: false,
  botType: 0,
  variant: "classic",
  minRatingRange: -500,
  maxRatingRange: 500,
  gameMode: 0,
  receiver: create(MatchUserSchema, {}),
  rematchFor: "",
  tournamentID: "",
  receiverIsPermanent: false,
  ratingKey: "",
  requireEstablishedRating: false,
  onlyFollowedPlayers: false,
});

describe("the chat store", () => {
  it("counts the times the chat list is replaced", () => {
    const { result } = renderHook(() => useChatStoreContext(), {
      wrapper: Store,
    });
    const before = result.current.chatGeneration;
    act(() => {
      result.current.addChat(serverMsg("a"));
    });
    expect(result.current.chatGeneration).toBe(before);
    act(() => {
      result.current.clearChat();
      result.current.addChats([serverMsg("b")]);
    });
    expect(result.current.chatGeneration).toBeGreaterThan(before);
    expect(result.current.chat.map((c) => c.id)).toEqual(["b"]);
    act(() => {
      result.current.setCurrentChatChannel("chat.game.abc");
    });
    expect(result.current.currentChatChannel).toBe("chat.game.abc");
  });

  it("does not add an entry whose id is already listed", () => {
    const { result } = renderHook(() => useChatStoreContext(), {
      wrapper: Store,
    });
    act(() => {
      result.current.addChat(serverMsg("a"));
      result.current.addChat(serverMsg("b"));
    });
    act(() => {
      result.current.addChat(serverMsg("a"));
    });
    expect(result.current.chat.map((c) => c.id)).toEqual(["a", "b"]);
  });
});

describe("the lobby store", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("applies queued actions together, a moment later", () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useLobbyStoreContext(), {
      wrapper: Store,
    });
    act(() => {
      result.current.queueLobbyAction({
        actionType: ActionType.AddSoughtGame,
        payload: seek("a"),
      });
      result.current.queueLobbyAction({
        actionType: ActionType.AddSoughtGame,
        payload: seek("b"),
      });
    });
    expect(result.current.lobbyContext.soughtGames).toEqual([]);
    act(() => {
      vi.advanceTimersByTime(LOBBY_QUEUE_MS);
    });
    expect(
      result.current.lobbyContext.soughtGames.map((sg) => sg.seekID),
    ).toEqual(["a", "b"]);
  });

  it("applies queued actions before a direct one", () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useLobbyStoreContext(), {
      wrapper: Store,
    });
    act(() => {
      result.current.queueLobbyAction({
        actionType: ActionType.AddSoughtGame,
        payload: seek("a"),
      });
      result.current.dispatchLobbyContext({
        actionType: ActionType.RemoveSoughtGame,
        payload: "a",
      });
    });
    expect(result.current.lobbyContext.soughtGames).toEqual([]);
    act(() => {
      vi.advanceTimersByTime(LOBBY_QUEUE_MS);
    });
    expect(result.current.lobbyContext.soughtGames).toEqual([]);
  });
});
