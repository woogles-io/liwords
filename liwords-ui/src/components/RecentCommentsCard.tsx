import { Button, Card, Table, Tooltip } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { useCollectionContext } from "../collections/useCollectionContext";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";

type Props = {
  comments: Array<GameComment>;
  fetchPrev?: () => void;
  fetchNext?: () => void;
  titleOverride?: string;
  collection?: Collection;
};

export const RecentCommentsCard = React.memo((props: Props) => {
  // Get collection context if available
  const collectionContext = useCollectionContext();

  const formattedComments = props.comments.map((comment) => {
    // Get game info from gameMeta if available
    const players = comment.gameMeta?.players || "Unknown players";

    // Create URL with both turn and comments parameters for full context
    // turn shows the board position AFTER the move (eventNumber + 2)
    // comments opens the drawer for the specific event (eventNumber + 1)
    const commentEventNumber = comment.eventNumber + 1;
    const turnPositionNumber = comment.eventNumber + 2;
    const params = new URLSearchParams();
    params.set("turn", turnPositionNumber.toString());
    params.set("comments", commentEventNumber.toString());

    // If collection prop is provided, use it to find the chapter
    if (props.collection) {
      const chapterIndex = props.collection.games.findIndex(
        (game) => game.gameId === comment.gameId,
      );
      if (chapterIndex !== -1) {
        const chapterNumber = chapterIndex + 1;
        params.set("collection", props.collection.uuid);
        params.set("chapter", chapterNumber.toString());
        params.set("total", props.collection.games.length.toString());
      }
    } else if (collectionContext) {
      // Fall back to using collection context if available (when not in collection viewer)
      params.set("collection", collectionContext.collectionUuid);
      params.set("chapter", collectionContext.currentChapter.toString());
      params.set("total", collectionContext.totalChapters.toString());
    }

    const url = `/anno/${encodeURIComponent(comment.gameId)}?${params.toString()}`;

    const whenMoment = moment(
      comment.lastEdited ? timestampDate(comment.lastEdited) : "",
    );
    const when = (
      <Tooltip title={whenMoment.format("LLL")}>{whenMoment.fromNow()}</Tooltip>
    );

    // Truncate comment for preview
    const commentPreview =
      comment.comment.length > 100
        ? comment.comment.substring(0, 100) + "..."
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
      title={props.titleOverride || "Recent Comments"}
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
