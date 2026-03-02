import React from "react";
import moment from "moment";
import { Card, Tag, Tooltip } from "antd";
import { PlayerAvatar } from "../shared/player_avatar";
import { Link } from "react-router";
import { VariantIcon } from "../shared/variant_icons";
import { FundOutlined } from "@ant-design/icons";
import { timeToString } from "../store/constants";
import { GameInfoResponse, RatingMode } from "../gen/api/proto/ipc/omgwords_pb";
import { challengeRuleNamesShort } from "../constants/challenge_rules";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { lexiconCodeToProfileRatingName } from "../shared/lexica";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type GameCardProps = {
  game: GameInfoResponse;
  userID: string;
};
export const GameCard = React.memo((props: GameCardProps) => {
  const { game, userID } = props;
  const special = ["Unwoogler", "AnotherUnwoogler", userID];
  const {
    createdAt,
    gameId,
    players,
    winner,
    scores,
    gameRequest,
    gameEndReason,
    timeControlName,
  } = game;
  const whenMoment = moment(createdAt ? timestampDate(createdAt) : "");
  const when = (
    <Tooltip title={whenMoment.format("LLL")}>{whenMoment.fromNow()}</Tooltip>
  );
  if (!(players?.length > 1)) {
    return null;
  }
  const userplace =
    special.indexOf(players[0].userId) > special.indexOf(players[1].userId)
      ? 0
      : 1;
  const opponent = players[1 - userplace];
  const opponentLink = (
    <div className="opponent-link">
      <PlayerAvatar
        player={{
          userId: opponent.userId,
          nickname: opponent.nickname,
        }}
      />
      <Link to={`/profile/${encodeURIComponent(opponent.nickname)}`}>
        {players[1 - userplace].nickname}
      </Link>
    </div>
  );

  const challenge =
    challengeRuleNamesShort[gameRequest?.challengeRule ?? ChallengeRule.VOID];

  let endReason = "";
  switch (gameEndReason) {
    case GameEndReason.TIME:
      endReason = "Time out";
      break;
    case GameEndReason.CONSECUTIVE_ZEROES:
      endReason = "Six-zero rule";
      break;
    case GameEndReason.RESIGNED:
      endReason = "Resignation";
      break;
    case GameEndReason.FORCE_FORFEIT:
      endReason = "Forfeit";
      break;
    case GameEndReason.ABORTED:
      endReason = "Aborted";
      break;
    case GameEndReason.CANCELLED:
      endReason = "Cancelled";
      break;
    case GameEndReason.TRIPLE_CHALLENGE:
      endReason = "Triple challenge";
      break;
    case GameEndReason.STANDARD:
      endReason = "Completed";
  }

  const getDetails = (
    <div className="detail-icons">
      <VariantIcon vcode={gameRequest?.rules?.variantName} />
      {gameRequest?.ratingMode === RatingMode.RATED ? (
        <Tooltip title="Rated">
          <FundOutlined />
        </Tooltip>
      ) : null}
      <Tooltip title="Challenge Mode">
        <span className={`challenge-rule mode_${challenge}`}>{challenge}</span>
      </Tooltip>
      {players[userplace].first && <Tag className="ant-tag-first">1st</Tag>}
    </div>
  );

  let result = "Loss";
  if (winner === -1) {
    result = "Tie";
  } else if (winner === userplace) {
    result = "Win";
  }
  const actions = [
    <Link
      key="examine-action"
      to={`/game/${encodeURIComponent(String(gameId ?? ""))}`}
    >
      Analyze
    </Link>,
  ];
  const scoreDisplay = (
    <>
      <div>
        <h3>
          {(scores?.[userplace] || 0).toString() +
            " - " +
            (scores?.[1 - userplace] || 0).toString()}
        </h3>
        <p>{when}</p>
      </div>
      <Tag className={`ant-tag-${result.toLowerCase()}`}>{result}</Tag>
    </>
  );
  let time: string;
  if (gameRequest?.gameMode === 1) {
    // Correspondence game
    time = "Correspondence";
  } else {
    time = `${timeControlName} ${timeToString(
      gameRequest?.initialTimeSeconds ?? 0,
      gameRequest?.incrementSeconds ?? 0,
      gameRequest?.maxOvertimeMinutes ?? 0,
    )}`;
  }
  return (
    <Card
      className={`game-info ${result.toLowerCase()}`}
      title={scoreDisplay}
      actions={actions}
    >
      {opponentLink}
      <div className="variant-info">
        {lexiconCodeToProfileRatingName(gameRequest?.lexicon ?? "")} -{" "}
        <span className="time-control">{time}</span>
      </div>
      <p>{endReason}</p>
      {getDetails}
    </Card>
  );
});
