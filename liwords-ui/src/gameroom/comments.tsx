import { DeleteOutlined, EditOutlined } from '@ant-design/icons';
import { Card, Input, Popconfirm } from 'antd';
import moment from 'moment';
import React, { useEffect, useMemo } from 'react';
import { GameComment } from '../gen/api/proto/comments_service/comments_service_pb';
import { useMountedState } from '../utils/mounted';

type Props = {
  comments: Array<GameComment>;
  loggedInUserID: string;
  deleteComment: (commentID: string) => void;
  editComment: (commentID: string, comment: string) => void;
  addComment: (comment: string) => void;
};

type SingleCommentProps = {
  comment: GameComment;
  mine: boolean;
  deleteComment: (commentID: string) => void;
  editComment: (commentID: string, comment: string) => void;
};

type EditProps = {
  buttonCaption: string;
  initialValue: string;
  setNotEditing: () => void;
  updateComment: (c: string) => void;
};

export const CommentEditor = (props: EditProps) => {
  const { useState } = useMountedState();

  const [inputValue, setInputValue] = useState(props.initialValue);
  return (
    <>
      <Input.TextArea
        value={inputValue}
        onChange={(evt) => setInputValue(evt.target.value)}
        rows={4}
      ></Input.TextArea>
      <span
        className="add-cmt-pseudo-btn"
        onClick={() => {
          if (inputValue.trim() !== props.initialValue.trim()) {
            props.updateComment(inputValue.trim());
          }
          props.setNotEditing();
        }}
      >
        {props.buttonCaption}
      </span>
    </>
  );
};

export const Comment = (props: SingleCommentProps) => {
  const { useState } = useMountedState();
  const [popupOpen, setPopupOpen] = useState(false);
  const initialCommentDisplay = useMemo(
    () => <p>{props.comment.comment}</p>,
    [props.comment.comment]
  );
  const [commentDisplay, setCommentDisplay] = useState(initialCommentDisplay);
  useEffect(() => {
    setCommentDisplay(initialCommentDisplay);
  }, [initialCommentDisplay]);
  return (
    <Card
      size="small"
      title={
        <>
          <a href={`/profile/${props.comment.username}`}>
            {props.comment.username}
          </a>
          <span className="timeago">
            {moment(props.comment.lastEdited?.toDate()).fromNow()}
          </span>
        </>
      }
      extra={
        props.mine ? (
          <>
            <EditOutlined
              onClick={() => {
                setCommentDisplay(
                  <CommentEditor
                    initialValue={props.comment.comment}
                    buttonCaption="Submit"
                    setNotEditing={() =>
                      setCommentDisplay(initialCommentDisplay)
                    }
                    updateComment={(cmt: string) =>
                      props.editComment(props.comment.commentId, cmt)
                    }
                  />
                );
              }}
            />
            <Popconfirm
              title="Are you sure you wish to delete this comment?"
              onCancel={() => setPopupOpen(false)}
              onConfirm={() => {
                props.deleteComment(props.comment.commentId);
                setPopupOpen(false);
              }}
              open={popupOpen}
              okText="Yes"
              cancelText="No"
            >
              <DeleteOutlined onClick={() => setPopupOpen(true)} />
            </Popconfirm>
          </>
        ) : null
      }
    >
      {commentDisplay}
    </Card>
  );
};

export const Comments = (props: Props) => {
  const { useState } = useMountedState();
  const [newEditorVisible, setNewEditorVisible] = useState(false);

  let footer = <></>;

  if (newEditorVisible) {
    footer = (
      <CommentEditor
        buttonCaption="Submit new comment"
        initialValue=""
        setNotEditing={() => setNewEditorVisible(false)}
        updateComment={props.addComment}
      />
    );
  } else {
    footer = (
      <span
        className="add-cmt-pseudo-btn"
        onClick={() => setNewEditorVisible(true)}
      >
        Add a comment
      </span>
    );
  }

  return (
    <div className="turn-comments">
      {props.comments.map((c) => {
        return (
          <Comment
            key={c.commentId}
            comment={c}
            mine={c.userId === props.loggedInUserID}
            deleteComment={props.deleteComment}
            editComment={props.editComment}
          />
        );
      })}
      {footer}
    </div>
  );
};
