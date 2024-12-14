import { DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Card, Input, Popconfirm } from "antd";
import moment from "moment";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import { canMod } from "../mod/perms";
import { useLoginStateStoreContext } from "../store/store";
import { timestampDate } from "@bufbuild/protobuf/wkt";

type Props = {
  comments: Array<GameComment>;
  deleteComment: (commentID: string) => void;
  editComment: (commentID: string, comment: string) => void;
  addComment: (comment: string) => void;
};

type SingleCommentProps = {
  comment: GameComment;
  mine: boolean;
  canMod: boolean;
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
  const myRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    console.log(myRef.current);
    myRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, []);
  const [inputValue, setInputValue] = useState(props.initialValue);
  return (
    <div>
      <Input.TextArea
        value={inputValue}
        onChange={(evt) => setInputValue(evt.target.value)}
        rows={4}
        autoFocus
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
      {/* a lil hack for scrolling into view */}
      <div style={{ position: "relative", top: 80 }} ref={myRef} />
    </div>
  );
};

export const Comment = (props: SingleCommentProps) => {
  const [popupOpen, setPopupOpen] = useState(false);
  const initialCommentDisplay = useMemo(
    () => <p>{props.comment.comment}</p>,
    [props.comment.comment],
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
            {props.comment.lastEdited
              ? moment(timestampDate(props.comment.lastEdited)).fromNow()
              : ""}
          </span>
        </>
      }
      extra={
        props.mine || props.canMod ? (
          <>
            {props.mine && (
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
                    />,
                  );
                }}
              />
            )}
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
  const [newEditorVisible, setNewEditorVisible] = useState(false);
  const myRef = useRef<HTMLDivElement | null>(null);
  let footer = <></>;
  const { loginState } = useLoginStateStoreContext();

  useEffect(() => {
    if (props.comments.length === 0) {
      // This shouldn't be showing at all if there are no comments,
      // unless the user clicked and interacted with the scorecard.
      // In this case, we want to make sure the "add new comment" button
      // is visible.
      myRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [props.comments]);

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
      <>
        <span
          ref={myRef}
          className="add-cmt-pseudo-btn"
          onClick={() => {
            setNewEditorVisible(true);
          }}
        >
          Add a comment
        </span>
        <div style={{ position: "relative", top: 50 }} ref={myRef} />
      </>
    );
  }

  return (
    <div className="turn-comments">
      {props.comments.map((c) => {
        return (
          <Comment
            key={c.commentId}
            comment={c}
            mine={c.userId === loginState.userID}
            deleteComment={props.deleteComment}
            editComment={props.editComment}
            canMod={canMod(loginState.perms)}
          />
        );
      })}
      {footer}
    </div>
  );
};
