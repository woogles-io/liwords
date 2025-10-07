import { create } from "@bufbuild/protobuf";
import { MatchUserSchema } from "../gen/api/proto/ipc/omgseeks_pb";
import { BotTypesEnum } from "./bots";
import { sendSeek } from "./sought_game_interactions";

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
