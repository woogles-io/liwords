import { Button, Card, Table, Tooltip } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { useCollectionContext } from "../collections/useCollectionContext";

type Props = {
  comments: Array<GameComment>;
  fetchPrev?: () => void;
  fetchNext?: () => void;
};

export const RecentCommentsCard = React.memo((props: Props) => {
  // Get collection context if available
  const collectionContext = useCollectionContext();

  const formattedComments = props.comments.map((comment) => {
    // Get game info from gameMeta if available
    const players = comment.gameMeta?.players || "Unknown players";

    // Create URL with proper turn numbering and optional collection context
    const turnNumber = comment.eventNumber + 1;
    const baseUrl = `/anno/${encodeURIComponent(comment.gameId)}?turn=${turnNumber}`;

    // Add collection context if available
    const url = collectionContext
      ? `${baseUrl}&collection=${collectionContext.collectionUuid}`
      : baseUrl;

    const whenMoment = moment(
      comment.lastEdited ? timestampDate(comment.lastEdited) : "",
    );
    const when = (
      <Tooltip title={whenMoment.format("LLL")}>{whenMoment.fromNow()}</Tooltip>
    );

    // Truncate comment for preview
    const commentPreview =
      comment.comment.length > 50
        ? comment.comment.substring(0, 50) + "..."
        : comment.comment;

    return {
      commentId: comment.commentId,
      commenter: (
        <Link to={`/profile/${comment.username}`}>{comment.username}</Link>
      ),
      when,
      gameLink: <Link to={url}>{players}</Link>,
      commentPreview: (
        <div style={{ maxWidth: "300px", overflow: "hidden" }}>
          <Link to={url} style={{ color: "inherit" }}>
            {commentPreview}
          </Link>
        </div>
      ),
    };
  });

  const columns = [
    {
      title: "Commenter",
      dataIndex: "commenter",
      key: "commenter",
      width: "15%",
    },
    {
      title: "When",
      dataIndex: "when",
      key: "when",
      width: "15%",
    },
    {
      title: "Game",
      dataIndex: "gameLink",
      key: "gameLink",
      width: "20%",
    },
    {
      title: "Comment",
      dataIndex: "commentPreview",
      key: "commentPreview",
      width: "50%",
    },
  ];

  return (
    <Card
      title="Recent Comments"
      className="game-history-card"
      style={{ marginBottom: "24px" }}
    >
      <Table
        className="game-history"
        columns={columns}
        dataSource={formattedComments}
        pagination={{
          hideOnSinglePage: true,
          defaultPageSize: Infinity,
        }}
        rowKey="commentId"
        size="small"
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
