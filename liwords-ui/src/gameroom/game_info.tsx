import React from "react";
import { Card } from "antd";
import ReactMarkdown from "react-markdown";
import { timeCtrlToDisplayName, timeToString } from "../store/constants";
import { VariantIcon } from "../shared/variant_icons";
import { MatchLexiconDisplay } from "../shared/lexicon_display";
import {
  GameEndReason,
  GameInfoResponseSchema,
  GameRequestSchema,
  GameRulesSchema,
  PlayerInfo,
} from "../gen/api/proto/ipc/omgwords_pb";
import {
  BotRequest_BotCode,
  ChallengeRule,
} from "../gen/api/vendor/macondo/macondo_pb";
import { RatingMode } from "../gen/api/proto/ipc/omgwords_pb";
import { challengeRuleNames } from "../constants/challenge_rules";
import { GameInfoResponse } from "../gen/api/proto/ipc/omgwords_pb";
import { create } from "@bufbuild/protobuf";

export const defaultGameInfo = create(GameInfoResponseSchema, {
  players: new Array<PlayerInfo>(),
  gameRequest: create(GameRequestSchema, {
    lexicon: "",
    rules: create(GameRulesSchema, {
      variantName: "",
      boardLayoutName: "CrosswordGame",
      letterDistributionName: "english",
    }),
    initialTimeSeconds: 0,
    incrementSeconds: 0,
    challengeRule: ChallengeRule.VOID,
    ratingMode: RatingMode.RATED,
    maxOvertimeMinutes: 0,
    originalRequestId: "",
    playerVsBot: false,
    botType: BotRequest_BotCode.HASTY_BOT,
  }),
  tournamentId: "",
  gameEndReason: GameEndReason.NONE,
  timeControlName: "",
});

type Props = {
  meta: GameInfoResponse;
  tournamentName: string;
  colorOverride?: string;
  logoUrl?: string;
  description?: string;
};

export const GameInfo = React.memo((props: Props) => {
  const variant = (
    <VariantIcon
      vcode={props.meta.gameRequest?.rules?.variantName || "classic"}
      withName
    />
  );
  const rated =
    props.meta.gameRequest?.ratingMode === RatingMode.RATED
      ? "Rated"
      : "Unrated";

  const challenge =
    challengeRuleNames[
      props.meta.gameRequest?.challengeRule ?? ChallengeRule.VOID
    ];

  const card = (
    <Card className="game-info">
      <div className="metadata">
        {props.meta.tournamentId && (
          <p
            className="tournament-name"
            style={{ color: props.colorOverride || "ignore" }}
          >
            {props.tournamentName}
          </p>
        )}
        <p className="variant">
          {`${
            timeCtrlToDisplayName(
              props.meta.gameRequest?.initialTimeSeconds ?? 0,
              props.meta.gameRequest?.incrementSeconds ?? 0,
              props.meta.gameRequest?.maxOvertimeMinutes ?? 0,
              props.meta.timeControlName === "Annotated"
                ? props.meta.timeControlName
                : undefined,
            )[0]
          } ${timeToString(
            props.meta.gameRequest?.initialTimeSeconds ?? 0,
            props.meta.gameRequest?.incrementSeconds ?? 0,
            props.meta.gameRequest?.maxOvertimeMinutes ?? 0,
            props.meta.timeControlName === "Annotated",
          )}`}{" "}
          • {variant} •{" "}
          <MatchLexiconDisplay
            lexiconCode={props.meta.gameRequest?.lexicon ?? ""}
          />
        </p>
        <p>
          {challenge} challenge • {rated}
        </p>
        {props.description && (
          <div className="game-description">
            <ReactMarkdown>{props.description}</ReactMarkdown>
          </div>
        )}
      </div>
      {props.logoUrl && (
        <div className="logo-container">
          <img
            className="club-logo"
            src={props.logoUrl}
            alt={props.tournamentName}
          />
        </div>
      )}
    </Card>
  );
  return card;
});
