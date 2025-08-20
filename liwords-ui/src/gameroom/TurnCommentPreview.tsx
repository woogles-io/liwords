import React from "react";
import { MessageOutlined } from "@ant-design/icons";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";

type Props = {
  comments: GameComment[];
  onExpandComments: () => void;
  className?: string;
};

export const TurnCommentPreview: React.FC<Props> = ({
  comments,
  onExpandComments,
  className = "",
}) => {
  if (comments.length === 0) {
    return (
      <div
        className={`turn-comment-preview no-comments ${className}`}
        onClick={onExpandComments}
      >
        <MessageOutlined className="comment-icon-subtle" />
      </div>
    );
  }

  return (
    <div
      className={`turn-comment-preview has-comments ${className}`}
      onClick={onExpandComments}
    >
      <MessageOutlined className="comment-icon" />
      <span className="comment-count">{comments.length}</span>
    </div>
  );
};
