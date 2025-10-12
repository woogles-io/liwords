import React from "react";
import { Card, Button } from "antd";
import { EditOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router";
import ReactMarkdown from "react-markdown";
import { useQuery } from "@connectrpc/connect-query";
import { timeCtrlToDisplayName, timeToString } from "../store/constants";
import { VariantIcon } from "../shared/variant_icons";
import { MatchLexiconDisplay } from "../shared/lexicon_display";
import {
  GameDocument,
  GameEndReason,
  GameInfoResponseSchema,
  GameRequestSchema,
  GameRulesSchema,
  GameType,
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
import { getGameOwner } from "../gen/api/proto/omgwords_service/omgwords-GameEventService_connectquery";

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
  gameDocument?: GameDocument;
  currentUserId?: string;
};

export const GameInfo = React.memo((props: Props) => {
  const navigate = useNavigate();

  // Get game owner information for annotated games
  const { data: gameOwner } = useQuery(
    getGameOwner,
    { gameId: props.gameDocument?.uid || "" },
    {
      enabled: !!(
        props.gameDocument?.uid && props.meta.type === GameType.ANNOTATED
      ),
    },
  );

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

  // Check if this is an annotated game and if the current user can edit it
  const canEdit =
    props.gameDocument &&
    props.currentUserId &&
    props.meta.type === GameType.ANNOTATED &&
    gameOwner?.found &&
    gameOwner.creatorId === props.currentUserId;

  const handleEditClick = () => {
    if (props.gameDocument?.uid) {
      navigate(`/editor/${props.gameDocument.uid}`);
    }
  };

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
          {props.meta.gameRequest?.gameMode === 1
            ? "Correspondence"
            : `${
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
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <span>
            {challenge} challenge • {rated}
          </span>
          {canEdit && (
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={handleEditClick}
              title="Edit this annotated game"
              style={{
                padding: "2px 4px",
                height: "20px",
                fontSize: "12px",
                marginLeft: 8,
              }}
            >
              Edit
            </Button>
          )}
        </div>
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
