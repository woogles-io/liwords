import { describe, expect, it } from "vitest";
import { GameMode } from "../gen/api/proto/ipc/omgwords_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { collapseIdenticalSeeks, pickSeek } from "./seek_collapse";

const seek = (over: Partial<SoughtGame>): SoughtGame =>
  ({
    seeker: "nishans",
    seekerID: "u1",
    lexicon: "CSW24",
    initialTimeSecs: 432000,
    incrementSecs: 86400,
    maxOvertimeMinutes: 0,
    challengeRule: 5,
    userRating: "1841",
    rated: true,
    seekID: "s",
    playerVsBot: false,
    botType: 0,
    variant: "classic",
    minRatingRange: -500,
    maxRatingRange: 500,
    gameMode: GameMode.CORRESPONDENCE,
    receiver: { displayName: "", userId: "" },
    rematchFor: "",
    tournamentID: "",
    receiverIsPermanent: false,
    ratingKey: "",
    requireEstablishedRating: false,
    onlyFollowedPlayers: false,
    ...over,
  }) as SoughtGame;

const summary = (seeks: Array<SoughtGame>, username?: string) =>
  collapseIdenticalSeeks(seeks, username).map((g) => [g.sg.seekID, g.count]);

describe("collapseIdenticalSeeks", () => {
  it("shows someone's identical correspondence seeks once, with a count", () => {
    expect(
      summary([
        seek({ seekID: "a" }),
        seek({ seekID: "b" }),
        seek({ seekID: "c", lexicon: "NWL23" }),
        seek({ seekID: "d" }),
      ]),
    ).toEqual([
      ["a", 3],
      ["c", 1],
    ]);
  });

  it("keeps seeks that differ in anything an acceptor cares about", () => {
    expect(
      summary([
        seek({ seekID: "a" }),
        seek({ seekID: "b", timeBankMinutes: 60 }),
        seek({ seekID: "c", rated: false }),
        seek({ seekID: "d", seekerID: "u2", seeker: "other" }),
      ]),
    ).toEqual([
      ["a", 1],
      ["b", 1],
      ["c", 1],
      ["d", 1],
    ]);
  });

  it("leaves the viewer's own, real-time seeks and match requests alone", () => {
    expect(
      summary(
        [
          seek({ seekID: "a" }),
          seek({ seekID: "b" }),
          seek({ seekID: "c", seeker: "me", seekerID: "me" }),
          seek({ seekID: "d", seeker: "me", seekerID: "me" }),
          seek({ seekID: "e", gameMode: GameMode.REAL_TIME }),
          seek({ seekID: "f", gameMode: GameMode.REAL_TIME }),
          seek({ seekID: "g", receiverIsPermanent: true }),
          seek({ seekID: "h", receiverIsPermanent: true }),
        ],
        "me",
      ),
    ).toEqual([
      ["a", 2],
      ["c", 1],
      ["d", 1],
      ["e", 1],
      ["f", 1],
      ["g", 1],
      ["h", 1],
    ]);
  });
});

describe("pickSeek", () => {
  it("can take any seek of a group", () => {
    const group = collapseIdenticalSeeks([
      seek({ seekID: "a" }),
      seek({ seekID: "b" }),
      seek({ seekID: "c" }),
    ])[0];
    expect(pickSeek(group, () => 0).seekID).toBe("a");
    expect(pickSeek(group, () => 0.5).seekID).toBe("b");
    expect(pickSeek(group, () => 0.99).seekID).toBe("c");
  });
});
