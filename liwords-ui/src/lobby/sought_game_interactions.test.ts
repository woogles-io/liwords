import { create } from "@bufbuild/protobuf";
import { notification } from "antd";
import { afterEach, describe, expect, vi } from "vitest";
import { MatchUserSchema } from "../gen/api/proto/ipc/omgseeks_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { BotTypesEnum } from "./bots";
import { sendSeek, submitSeekFromForm } from "./sought_game_interactions";

it("tests sendSeek", () => {
  const game = {
    seeker: "",
    userRating: "",
    seekID: "",
    ratingKey: "",
    lexicon: "CSW24",
    challengeRule: 0,
    initialTimeSecs: 1200,
    incrementSecs: 0,
    rated: true,
    maxOvertimeMinutes: 1,
    receiver: create(MatchUserSchema, {
      userId: "",
      relevantRating: "",
      isAnonymous: false,
      displayName: "",
    }),
    rematchFor: "",
    playerVsBot: false,
    tournamentID: "",
    receiverIsPermanent: false,
    minRatingRange: -500,
    maxRatingRange: 500,
    botType: BotTypesEnum.MASTER,
    variant: "",
    gameMode: 0,
    requireEstablishedRating: false,
    onlyFollowedPlayers: false,
  };

  sendSeek(game, (msg: Uint8Array) => {
    console.log("Fake sending a msg: ", msg);
  });
});

describe("submitSeekFromForm", () => {
  const seek: SoughtGame = {
    seeker: "",
    userRating: "",
    seekID: "",
    ratingKey: "",
    lexicon: "CSW24",
    challengeRule: 0,
    initialTimeSecs: 1200,
    incrementSecs: 0,
    rated: false,
    maxOvertimeMinutes: 1,
    receiver: create(MatchUserSchema, {}),
    rematchFor: "",
    playerVsBot: true,
    tournamentID: "",
    receiverIsPermanent: false,
    minRatingRange: 0,
    maxRatingRange: 0,
    botType: BotTypesEnum.BEGINNER,
    variant: "classic",
    gameMode: 0,
    requireEstablishedRating: false,
    onlyFollowedPlayers: false,
  };

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("re-enables the form after a seek is sent", () => {
    vi.useFakeTimers();
    const shown = vi.spyOn(notification, "open").mockImplementation(() => {});
    const setFormDisabled = vi.fn();
    const send = vi.fn();
    submitSeekFromForm(seek, send, setFormDisabled);
    expect(send).toHaveBeenCalledWith(seek);
    expect(shown).not.toHaveBeenCalled();
    vi.advanceTimersByTime(500);
    expect(setFormDisabled).toHaveBeenCalledWith(false);
  });

  it("tells the player and re-enables the form when a seek cannot be sent", () => {
    vi.useFakeTimers();
    const shown = vi.spyOn(notification, "open").mockImplementation(() => {});
    vi.spyOn(console, "error").mockImplementation(() => {});
    const setFormDisabled = vi.fn();
    // A seek the protobuf library refuses to encode, as BeginnerBot's was
    // before the bot form sent its challenge rule.
    const unsendable = {
      ...seek,
      challengeRule: undefined,
    } as unknown as SoughtGame;
    submitSeekFromForm(
      unsendable,
      (g) => sendSeek(g, () => {}),
      setFormDisabled,
    );
    expect(shown).toHaveBeenCalledTimes(1);
    expect(shown.mock.calls[0][0].message).toBe("Error");
    expect(String(shown.mock.calls[0][0].description)).toContain(
      "challenge_rule",
    );
    expect(setFormDisabled).not.toHaveBeenCalled();
    vi.advanceTimersByTime(500);
    expect(setFormDisabled).toHaveBeenCalledWith(false);
  });
});
