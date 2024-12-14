import { Button, Card, Table, Tooltip } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router-dom";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import { BroadcastGamesResponse_BroadcastGame } from "../gen/api/proto/omgwords_service/omgwords_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type Props = {
  games: Array<BroadcastGamesResponse_BroadcastGame>;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  loggedInUserID: string;
  showAnnotator: boolean;
};

export const AnnotatedGamesHistoryCard = React.memo((props: Props) => {
  const formattedGames = props.games.map((item) => {
    const players = item.playersInfo
      .map((v: PlayerInfo) => {
        return v.fullName;
      })
      .join(" vs ");
    const url = `/anno/${encodeURIComponent(item.gameId)}?turn=1`;
    const whenMoment = moment(
      item.createdAt ? timestampDate(item.createdAt) : "",
    );
    const when = (
      <Tooltip title={whenMoment.format("LLL")}>{whenMoment.fromNow()}</Tooltip>
    );
    const edit =
      props.loggedInUserID === item.creatorId ? (
        <Link to={`/editor/${encodeURIComponent(item.gameId)}`}>Edit</Link>
      ) : (
        ""
      );
    return {
      gameId: item.gameId,
      lexicon: item.lexicon,
      when,
      link: <Link to={url}>{players}</Link>,
      edit,
      creator: (
        <Link to={`/profile/${item.creatorUsername}`}>
          {item.creatorUsername}
        </Link>
      ),
    };
  });
  const columns = [
    {
      dataIndex: "when",
      key: "when",
      title: "When",
    },
    {
      title: "Players",
      dataIndex: "link",
      key: "link",
    },
    {
      title: "Words",
      key: "lexicon",
      dataIndex: "lexicon",
    },
    {
      key: "edit",
      dataIndex: "edit",
      title: " ",
    },
  ];
  if (props.showAnnotator) {
    columns.splice(2, 0, {
      title: "Annotator",
      key: "annotator",
      dataIndex: "creator",
    });
  }

  return (
    <Card title="Annotated game history" className="game-history-card">
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
