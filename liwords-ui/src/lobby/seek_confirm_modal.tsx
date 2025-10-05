import React from "react";
import { Link } from "react-router";
import { Modal, Button, Alert } from "antd";
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
import { ChallengeRule } from "../gen/api/vendor/macondo/macondo_pb";
import { RatingBadge } from "./rating_badge";
import { timeCtrlToDisplayName } from "../store/constants";

type Props = {
  open: boolean;
  seek: SoughtGame | null;
  onAccept: () => void;
  onCancel: () => void;
  userRatings: { [key: string]: ProfileUpdate_Rating };
};

export const SeekConfirmModal = (props: Props) => {
  const { open, seek, onAccept, onCancel, userRatings } = props;

  if (!seek) {
    return null;
  }

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

  // Check if user has rating in this lexicon (at any time control)
  // Rating keys are formatted as: lexicon.variant.timeCtrl
  const lexiconPrefix = seek.ratingKey.substring(
    0,
    seek.ratingKey.lastIndexOf("."),
  ); // e.g., "CSW24.classic"
  const hasLexiconRating = Object.keys(userRatings).some((key) =>
    key.startsWith(lexiconPrefix),
  );

  // Determine which warning to show (priority order)
  let warningMessage = "";
  if (!hasLexiconRating) {
    warningMessage = `You have no rated games in this lexicon (${seek.lexicon}) - Are you sure you want to play?`;
  } else if (showTimeCtrlWarning) {
    warningMessage = `You have no rated games in this time control (${seekTimeCtrl}) - Are you sure you want to play?`;
  } else if (showChallengeWarning) {
    warningMessage =
      "This game uses non-VOID challenges - make sure you understand the rules";
  }

  return (
    <Modal
      title="Accept Seek"
      open={open}
      onCancel={onCancel}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          Cancel
        </Button>,
        <Button key="accept" type="primary" onClick={onAccept}>
          Accept
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
          {seek.variant && seek.variant !== "classic" && `, ${seek.variant}`}
        </div>
      </div>

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
            player={{ displayName: seek.seeker, uuid: seek.seekerID }}
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
