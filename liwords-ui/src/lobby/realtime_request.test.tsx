import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GameMode } from "../gen/api/proto/ipc/omgwords_pb";
import { MatchUserSchema } from "../gen/api/proto/ipc/omgseeks_pb";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { BotTypesEnum } from "./bots";
import { findMyRealtimeRequests } from "./realtime_request";
import { RealtimeRequestNote } from "./realtime_request_note";

const seek = (over: Partial<SoughtGame>): SoughtGame => ({
  seeker: "someone",
  seekerID: "someone-id",
  lexicon: "CSW24",
  initialTimeSecs: 1200,
  incrementSecs: 0,
  maxOvertimeMinutes: 1,
  challengeRule: 0,
  userRating: "1500",
  rated: true,
  seekID: "seek",
  playerVsBot: false,
  botType: BotTypesEnum.MASTER,
  variant: "classic",
  minRatingRange: -500,
  maxRatingRange: 500,
  gameMode: GameMode.REAL_TIME,
  receiver: create(MatchUserSchema, {}),
  rematchFor: "",
  tournamentID: "",
  receiverIsPermanent: false,
  ratingKey: "",
  requireEstablishedRating: false,
  onlyFollowedPlayers: false,
  ...over,
});

const ids = (sgs: Array<SoughtGame>) => sgs.map((sg) => sg.seekID);

describe("findMyRealtimeRequests", () => {
  const myCorres = seek({
    seekID: "my-corres",
    seeker: "andy",
    seekerID: "andy-id",
    gameMode: GameMode.CORRESPONDENCE,
  });
  const theirRealtime = seek({ seekID: "their-realtime" });
  const myRealtime = seek({
    seekID: "my-realtime",
    seeker: "andy",
    seekerID: "andy-id",
  });

  it("finds the user's open real-time seeks, not correspondence ones or someone else's", () => {
    const myOther = seek({
      seekID: "my-other",
      seeker: "andy",
      seekerID: "andy-id",
    });
    expect(
      ids(
        findMyRealtimeRequests(
          [myCorres, theirRealtime, myRealtime, myOther],
          [],
          "andy-id",
          "andy",
        ),
      ),
    ).toEqual(["my-realtime", "my-other"]);
  });

  it("finds a real-time match request the user sent", () => {
    const sent = seek({
      seekID: "my-match",
      seeker: "andy",
      seekerID: undefined,
      receiverIsPermanent: true,
    });
    expect(
      ids(findMyRealtimeRequests([myCorres], [sent], "andy-id", "andy")),
    ).toEqual(["my-match"]);
  });

  it("finds nothing when the user's only seeks are correspondence ones", () => {
    expect(
      findMyRealtimeRequests([myCorres, theirRealtime], [], "andy-id", "andy"),
    ).toEqual([]);
  });

  it("finds nothing without a user, even for a seek without a seeker ID", () => {
    expect(
      findMyRealtimeRequests(
        [seek({ seekerID: undefined })],
        [],
        undefined,
        "",
      ),
    ).toEqual([]);
  });
});

describe("RealtimeRequestNote", () => {
  it("names an open seek and shows it on click", () => {
    const onShow = vi.fn();
    render(<RealtimeRequestNote requests={[seek({})]} onShow={onShow} />);
    expect(
      screen.getByText(/You also have a real-time seek open\./),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show it" }));
    expect(onShow).toHaveBeenCalledTimes(1);
  });

  it("names a match request", () => {
    render(
      <RealtimeRequestNote requests={[seek({ receiverIsPermanent: true })]} />,
    );
    expect(
      screen.getByText(/You also have a real-time match request open\./),
    ).toBeInTheDocument();
  });

  it("counts several", () => {
    render(
      <RealtimeRequestNote
        requests={[seek({ seekID: "a" }), seek({ seekID: "b" })]}
      />,
    );
    expect(
      screen.getByText(/You also have 2 real-time seeks open\./),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Show them" }),
    ).toBeInTheDocument();
  });
});
