import React, { useState } from "react";
import { Link } from "react-router";
import { Modal, Button, Alert, theme } from "antd";
import {
  ClockCircleOutlined,
  FileTextOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { SoughtGame } from "../store/reducers/lobby_reducer";
import { ProfileUpdate_Rating } from "../gen/api/proto/ipc/users_pb";
import { PlayerAvatar } from "../shared/player_avatar";
import { DisplayUserFlag } from "../shared/display_flag";
import { MatchLexiconDisplay } from "../shared/lexicon_display";
import { timeFormat, challengeFormat } from "./sought_games";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { RatingBadge } from "./rating_badge";
import { timeCtrlToDisplayName } from "../store/constants";
import { VariantIcon } from "../shared/variant_icons";
import { variantDescriptions } from "./variant_utils";

type Props = {
  open: boolean;
  seek: SoughtGame | null;
  onAccept: () => void;
  onCancel: () => void;
  onDecline?: () => void;
  userRatings: { [key: string]: ProfileUpdate_Rating };
};

export const SeekConfirmModal = (props: Props) => {
  const { open, seek, onAccept, onCancel, onDecline, userRatings } = props;
  const [showVariantInfo, setShowVariantInfo] = useState(false);
  const { token } = theme.useToken();

  if (!seek) {
    return null;
  }

  // Check if this is a direct match request vs an open seek
  const isMatchRequest = seek.receiverIsPermanent || false;
  const modalTitle = isMatchRequest ? "Accept Match" : "Accept Seek";
  const buttonText = isMatchRequest ? "Accept Match" : "Accept Seek";

  // Check if user has ANY ratings (to determine if they're completely new)
  const hasAnyRatings = Object.keys(userRatings).length > 0;

  // Check if this is a non-VOID challenge game
  const isNonVoidChallenge = seek.challengeRule !== ChallengeRule.VOID;

  // Show challenge warning if they have no ratings and this is non-VOID
  const showChallengeWarning = !hasAnyRatings && isNonVoidChallenge;

  // Check time control warning - only warn if FASTER than what they've played
  const seekTimeCtrl = timeCtrlToDisplayName(
    seek.initialTimeSecs,
    seek.incrementSecs,
    seek.maxOvertimeMinutes,
  )[1]; // Get the format string (ultrablitz, blitz, rapid, regular)

  // Time control speed order (fastest to slowest)
  const timeCtrlSpeed: { [key: string]: number } = {
    ultrablitz: 0,
    blitz: 1,
    rapid: 2,
    regular: 3,
  };

  const seekSpeed = timeCtrlSpeed[seekTimeCtrl] ?? 3;

  // Check if user has ANY rating in this time control
  const hasTimeCtrlRating = Object.keys(userRatings).some((key) =>
    key.endsWith(`.${seekTimeCtrl}`),
  );

  // Check if user has ANY rating in a SLOWER time control
  const hasSlowerTimeCtrlRating = Object.keys(userRatings).some((key) => {
    const match = key.match(/\.(ultrablitz|blitz|rapid|regular)$/);
    if (match) {
      const userSpeed = timeCtrlSpeed[match[1]] ?? 3;
      return userSpeed > seekSpeed; // This game is slower than the seek
    }
    return false;
  });

  // Show time control warning if they have slower games but not this speed or faster
  const showTimeCtrlWarning =
    hasAnyRatings && !hasTimeCtrlRating && hasSlowerTimeCtrlRating;

  // Check if user has rating in this lexicon (at any variant/time control)
  // Rating keys are formatted as: lexicon.variant.timeCtrl
  // Extract just the lexicon part (first component)
  const lexicon = seek.ratingKey.split(".")[0]; // e.g., "CSW24"
  const hasLexiconRating = Object.keys(userRatings).some((key) =>
    key.startsWith(lexicon + "."),
  );

  // Check if user has rating in this variant (at any lexicon/time control)
  // Rating keys are formatted as: lexicon.variant.timeCtrl
  const seekVariant = seek.variant || "classic";
  const hasVariantRating = Object.keys(userRatings).some((key) => {
    const parts = key.split(".");
    return parts.length >= 2 && parts[1] === seekVariant;
  });
  const isNonClassicVariant =
    seekVariant !== "classic" && seekVariant !== "" && seekVariant;
  const showVariantWarning =
    hasAnyRatings && isNonClassicVariant && !hasVariantRating;

  // Determine which warning to show (priority order)
  let warningMessage = "";
  if (!hasLexiconRating) {
    warningMessage = `You have no rated games in this lexicon (${seek.lexicon}) - Are you sure you want to play?`;
  } else if (showVariantWarning) {
    warningMessage = `You have no rated games in this variant (${seekVariant}) - Are you sure you want to play?`;
  } else if (showTimeCtrlWarning) {
    warningMessage = `You have no rated games in this time control (${seekTimeCtrl}) - Are you sure you want to play?`;
  } else if (showChallengeWarning) {
    warningMessage =
      "This game uses non-VOID challenges - make sure you understand the rules";
  }

  return (
    <Modal
      title={modalTitle}
      open={open}
      onCancel={onCancel}
      footer={[
        ...(isMatchRequest && onDecline
          ? [
              <Button key="decline" danger onClick={onDecline}>
                Decline
              </Button>,
            ]
          : []),
        <Button key="accept" type="primary" onClick={onAccept}>
          {buttonText}
        </Button>,
      ]}
      width={500}
    >
      <div style={{ marginBottom: 24 }}>
        <h4
          style={{
            color: "#999",
            fontSize: 12,
            fontWeight: 600,
            letterSpacing: 1,
            marginBottom: 16,
          }}
        >
          GAME SETTINGS
        </h4>

        <div
          style={{ display: "flex", alignItems: "center", marginBottom: 12 }}
        >
          <FileTextOutlined
            style={{ fontSize: 20, marginRight: 12, color: "#1890ff" }}
          />
          <MatchLexiconDisplay lexiconCode={seek.lexicon} />
        </div>

        <div
          style={{ display: "flex", alignItems: "center", marginBottom: 12 }}
        >
          <ClockCircleOutlined
            style={{ fontSize: 20, marginRight: 12, color: "#1890ff" }}
          />
          {timeFormat(
            seek.initialTimeSecs,
            seek.incrementSecs,
            seek.maxOvertimeMinutes,
            seek.gameMode,
          )}
        </div>

        <div
          style={{ display: "flex", alignItems: "center", marginBottom: 12 }}
        >
          <SettingOutlined
            style={{ fontSize: 20, marginRight: 12, color: "#1890ff" }}
          />
          {challengeFormat(seek.challengeRule)},{" "}
          {seek.rated ? "Rated" : "Unrated"}
        </div>
      </div>

      {seek.variant && seek.variant !== "classic" && (
        <div style={{ marginBottom: 24 }}>
          <h4
            style={{
              color: "#999",
              fontSize: 12,
              fontWeight: 600,
              letterSpacing: 1,
              marginBottom: 16,
            }}
          >
            VARIANT
          </h4>
          <div
            style={{ display: "flex", alignItems: "center", marginBottom: 8 }}
          >
            <VariantIcon vcode={seek.variant} />
            <span style={{ marginLeft: 8, fontWeight: 500 }}>
              {seek.variant === "wordsmog"
                ? "WordSmog"
                : seek.variant === "classic_super"
                  ? "ZOMGWords"
                  : seek.variant}
            </span>
            <a
              onClick={() => setShowVariantInfo(!showVariantInfo)}
              style={{ marginLeft: 12, fontSize: 13, cursor: "pointer" }}
            >
              {showVariantInfo ? "Hide info" : "Read more"}
            </a>
          </div>
          {showVariantInfo && (
            <div
              style={{
                padding: 12,
                backgroundColor: token.colorFillQuaternary,
                borderRadius: 4,
                fontSize: 13,
                lineHeight: 1.6,
              }}
            >
              {variantDescriptions[seek.variant]?.description || (
                <p style={{ margin: 0 }}>
                  This is a non-standard variant with special rules. Good luck!
                </p>
              )}
            </div>
          )}
        </div>
      )}

      <div style={{ marginBottom: 24 }}>
        <h4
          style={{
            color: "#999",
            fontSize: 12,
            fontWeight: 600,
            letterSpacing: 1,
            marginBottom: 16,
          }}
        >
          YOUR OPPONENT
        </h4>

        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <PlayerAvatar
            player={{
              userId: seek.seekerID,
              nickname: seek.seeker,
            }}
          />
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <DisplayUserFlag uuid={seek.seekerID} />
              <span style={{ fontWeight: 500 }}>{seek.seeker}</span>
              <RatingBadge rating={seek.userRating} />
            </div>
            <div style={{ marginTop: 8 }}>
              <Link
                to={`/profile/${encodeURIComponent(seek.seeker)}`}
                style={{ fontSize: 13 }}
              >
                View Profile
              </Link>
            </div>
          </div>
        </div>
      </div>

      {warningMessage && (
        <Alert message={warningMessage} type="warning" showIcon />
      )}
    </Modal>
  );
};
