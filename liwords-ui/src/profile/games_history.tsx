import React from "react";
import { Link } from "react-router";
import moment from "moment";
import { Button, Card, InputNumber, Table, Tag, Tooltip } from "antd";
import { CheckCircleTwoTone } from "@ant-design/icons";
import { FundOutlined } from "@ant-design/icons";
import { timeToString } from "../store/constants";
import { VariantIcon } from "../shared/variant_icons";
import { GameInfoResponse, RatingMode } from "../gen/api/proto/ipc/omgwords_pb";
import { challengeRuleNamesShort } from "../constants/challenge_rules";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";
import { lexiconCodeToProfileRatingName } from "../shared/lexica";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type Props = {
  games: Array<GameInfoResponse>;
  username: string;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  userID: string;
  currentOffset: number;
  currentPageSize: number;
  desiredOffset: number;
  desiredPageSize: number;
  onChangePageNumber: (value: number | string | null) => void;
};

export const GamesHistoryCard = React.memo((props: Props) => {
  const {
    userID,
    currentOffset,
    currentPageSize,
    desiredOffset,
    desiredPageSize,
  } = props;

  // The view currently assumes:
  // currentPageSize === desiredPageSize
  // currentOffset === (currentPageNumber - 1) * currentPageSize
  // desiredOffset === (desiredPageNumber - 1) * desiredPageSize
  const currentPageNumber = Math.floor(currentOffset / currentPageSize + 1);
  const desiredPageNumber = Math.floor(desiredOffset / desiredPageSize + 1);
  const isSamePageSize = currentPageSize === desiredPageSize;
  const isSamePage = isSamePageSize && currentOffset === desiredOffset;
  void isSamePage;
  void currentPageNumber;
  // The above is not currently used, but here's a possible usage:
  //    {String(currentPageNumber)}
  //    {!isSamePage && (
  //      <React.Fragment> &rarr; {String(desiredPageNumber)}</React.Fragment>
  //    )}

  const special = ["Unwoogler", "AnotherUnwoogler", userID];
  const formattedGames = props.games
    .filter(
      (item) =>
        item.players?.length && item.gameEndReason !== GameEndReason.CANCELLED,
    )
    .map((item) => {
      const userplace =
        special.indexOf(item.players[0].userId) >
        special.indexOf(item.players[1].userId)
          ? 0
          : 1;
      const opponent = (
        <Link
          to={`/profile/${encodeURIComponent(
            item.players[1 - userplace].nickname,
          )}`}
        >
          {item.players[1 - userplace].nickname}
        </Link>
      );
      const scores = item.scores ? (
        <Link to={`/game/${encodeURIComponent(String(item.gameId ?? ""))}`}>
          {item.scores[userplace]} - {item.scores[1 - userplace]}
        </Link>
      ) : (
        ""
      );
      let result = <Tag color="red">Loss</Tag>;
      const challenge =
        challengeRuleNamesShort[
          item.gameRequest?.challengeRule ?? ChallengeRule.VOID
        ];

      const getDetails = () => {
        return (
          <>
            <VariantIcon vcode={item.gameRequest?.rules?.variantName} />{" "}
            <span className={`challenge-rule mode_${challenge}`}>
              {challenge}
            </span>
            {item.gameRequest?.ratingMode === RatingMode.RATED ? (
              <Tooltip title="Rated">
                <FundOutlined />
              </Tooltip>
            ) : null}
          </>
        );
      };
      if (item.winner === -1) {
        result = <Tag color="gray">Tie</Tag>;
      } else if (item.winner === userplace) {
        result = <Tag color="green">Win</Tag>;
      }
      let turnOrder = null;
      if (item.players[userplace].first) {
        turnOrder = <CheckCircleTwoTone twoToneColor="#52c41a" />;
      }
      const whenMoment = moment(
        item.createdAt ? timestampDate(item.createdAt) : "",
      );
      const when = (
        <Tooltip title={whenMoment.format("LLL")}>
          {whenMoment.fromNow()}
        </Tooltip>
      );
      let endReason = "";
      switch (item.gameEndReason) {
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
      let time: string;
      if (item.gameRequest?.gameMode === 1) {
        // Correspondence game
        time = "Correspondence";
      } else {
        time = `${item.timeControlName} ${timeToString(
          item.gameRequest?.initialTimeSeconds ?? 0,
          item.gameRequest?.incrementSeconds ?? 0,
          item.gameRequest?.maxOvertimeMinutes ?? 0,
        )}`;
      }
      return {
        gameId: item.gameId, // used by rowKey
        details: getDetails(),
        result,
        opponent,
        scores,
        turnOrder,
        endReason,
        lexicon: lexiconCodeToProfileRatingName(
          item.gameRequest?.lexicon ?? "",
        ),
        time,
        when,
      };
    })
    .filter((item) => item !== null);
  const columns = [
    {
      className: "result",
      dataIndex: "result",
      key: "result",
      title: " ",
    },
    {
      className: "when",
      dataIndex: "when",
      key: "when",
      title: " ",
    },
    {
      className: "opponent",
      dataIndex: "opponent",
      key: "opponent",
      title: "Opponent",
    },
    {
      className: "score",
      dataIndex: "scores",
      key: "scores",
      title: "Final Score",
    },
    {
      className: "turn-order",
      dataIndex: "turnOrder",
      key: "turnOrder",
      title: "First",
    },
    {
      className: "end-reason",
      dataIndex: "endReason",
      key: "endReason",
      title: "End",
    },
    {
      className: "lexicon",
      dataIndex: "lexicon",
      key: "lexicon",
      title: "Words",
    },
    {
      className: "time",
      dataIndex: "time",
      key: "time",
      title: "Time Settings",
    },
    {
      title: "Details",
      className: "details",
      dataIndex: "details",
      key: "details",
    },
  ];
  // TODO: use the normal Ant table pagination when the backend can give us a total
  return (
    <Card title="Game history" className="game-history-card">
      <Table
        className="game-history"
        columns={columns}
        dataSource={formattedGames}
        pagination={{
          hideOnSinglePage: true,
          defaultPageSize: Infinity,
        }}
        rowKey="gameId"
      />
      <div className="game-history-controls">
        <InputNumber
          inputMode="numeric"
          min={1}
          value={desiredPageNumber}
          onChange={props.onChangePageNumber}
        />
        <Button disabled={!props.fetchPrev} onClick={props.fetchPrev}>
          Prev
        </Button>
        <Button disabled={!props.fetchNext} onClick={props.fetchNext}>
          Next
        </Button>
      </div>
    </Card>
  );
});
