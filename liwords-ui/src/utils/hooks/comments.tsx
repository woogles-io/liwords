import { PromiseClient } from '@domino14/connect-web';
import { useCallback, useEffect, useState } from 'react';
import { GameCommentService } from '../../gen/api/proto/comments_service/comments_service_connectweb';
import { GameComment } from '../../gen/api/proto/comments_service/comments_service_pb';
import { useGameContextStoreContext } from '../../store/store';
import { flashError } from './connect';

export const useComments = (
  commentsClient: PromiseClient<typeof GameCommentService>,
  enableComments: boolean
) => {
  const { gameContext } = useGameContextStoreContext();

  const [comments, setComments] = useState<Array<GameComment>>([]);

  const fetchComments = useCallback(async () => {
    try {
      const comments = await commentsClient.getGameComments({
        gameId: gameContext.gameID,
      });
      setComments(comments.comments);
    } catch (e) {
      console.error(e);
    }
  }, [commentsClient, gameContext.gameID]);

  useEffect(() => {
    if (!gameContext.gameID) {
      return;
    }
    if (!enableComments) {
      return;
    }
    fetchComments();
  }, [enableComments, gameContext.gameID, fetchComments]);

  const editComment = async (cid: string, comment: string) => {
    try {
      await commentsClient.editGameComment({
        commentId: cid,
        comment: comment,
      });
      // If it succeeded, the backend accepted the comment. Fetch and dispatch
      // all messages.
      // In the future we can just wait for socket messages.
      fetchComments();
    } catch (e) {
      flashError(e);
    }
  };

  const deleteComment = async (cid: string) => {
    try {
      await commentsClient.deleteGameComment({
        commentId: cid,
      });
      fetchComments();
    } catch (e) {
      flashError(e);
    }
  };

  const addNewComment = async (
    gameID: string,
    eventNumber: number,
    comment: string
  ) => {
    try {
      await commentsClient.addGameComment({
        gameId: gameID,
        eventNumber,
        comment,
      });
      fetchComments();
    } catch (e) {
      flashError(e);
    }
  };

  return {
    comments,
    editComment,
    deleteComment,
    addNewComment,
  };
};
