import React, { useCallback } from "react";
import { Drawer, Button, Typography, message } from "antd";
import { CloseOutlined, LinkOutlined } from "@ant-design/icons";
import { GameComment } from "../gen/api/proto/comments_service/comments_service_pb";
import { Comments } from "./comments";
import { Turn } from "../store/reducers/turns";
import {
  nicknameFromEvt,
  tilePlacementEventDisplay,
} from "../utils/cwgame/game_event";
import { Board } from "../utils/cwgame/board";
import { Alphabet } from "../constants/alphabets";
import { PlayerInfo } from "../gen/api/proto/ipc/omgwords_pb";
import { GameEvent_Type } from "../gen/api/proto/vendored/macondo/macondo_pb";

const { Title, Text } = Typography;

type Props = {
  visible: boolean;
  onClose: () => void;
  eventNumber: number;
  comments: GameComment[];
  turns: Turn[];
  board: Board;
  alphabet: Alphabet;
  players: PlayerInfo[];
  onAddComment: (comment: string) => void;
  onEditComment: (commentId: string, comment: string) => void;
  onDeleteComment: (commentId: string) => void;
  gameId: string;
  baseUrl?: string; // Optional override for copy link URL base
};

export const CommentsDrawer: React.FC<Props> = ({
  visible,
  onClose,
  eventNumber,
  comments,
  turns,
  board,
  alphabet,
  players,
  onAddComment,
  onEditComment,
  onDeleteComment,
  gameId,
  baseUrl,
}) => {
  // Find the turn and event that contains this eventNumber
  const findTurnAndEvent = () => {
    for (let i = 0; i < turns.length; i++) {
      const turn = turns[i];
      const eventIndexInTurn = eventNumber - turn.firstEvtIdx;
      if (eventIndexInTurn >= 0 && eventIndexInTurn < turn.events.length) {
        return {
          turn,
          event: turn.events[eventIndexInTurn],
          turnIndex: i,
        };
      }
    }
    return null;
  };

  const turnAndEvent = findTurnAndEvent();
  const currentTurn = turnAndEvent?.turn;
  const currentEvent = turnAndEvent?.event;
  const turnIndex = turnAndEvent?.turnIndex ?? 0;

  // Filter comments to only show those for this specific event
  const eventComments = comments.filter((c) => c.eventNumber === eventNumber);

  const getTurnDisplayInfo = () => {
    if (!currentEvent) {
      return {
        player: "Unknown",
        play: "No move data",
        score: 0,
        playScore: 0,
      };
    }

    const player = nicknameFromEvt(currentEvent, players);
    const cumulativeScore = currentEvent.cumulative;

    // Calculate the score for just this play
    let playScore = 0;
    if (turnIndex > 0 && currentTurn) {
      // Find the previous event for the same player to get their previous cumulative score
      const playerIndex = currentEvent.playerIndex;
      let previousCumulative = 0;

      // Look through all previous events to find the last one by this player
      for (let i = turnIndex - 1; i >= 0; i--) {
        const prevTurn = turns[i];
        if (prevTurn && prevTurn.events.length > 0) {
          const lastEventInTurn = prevTurn.events[prevTurn.events.length - 1];
          if (lastEventInTurn.playerIndex === playerIndex) {
            previousCumulative = lastEventInTurn.cumulative;
            break;
          }
        }
      }

      playScore = cumulativeScore - previousCumulative;
    } else {
      // First turn for this player
      playScore = cumulativeScore;
    }

    let play: string;

    switch (currentEvent.type) {
      case GameEvent_Type.TILE_PLACEMENT_MOVE:
        play = tilePlacementEventDisplay(currentEvent, board, alphabet);
        break;
      case GameEvent_Type.PASS:
        play = "Pass";
        break;
      case GameEvent_Type.EXCHANGE:
        const exchangedTiles = currentEvent.exchanged || "";
        play = exchangedTiles ? `Exchange ${exchangedTiles}` : "Exchange";
        break;
      case GameEvent_Type.CHALLENGE_BONUS:
        play = "Challenge bonus";
        break;
      case GameEvent_Type.PHONY_TILES_RETURNED:
        play = "Phony tiles returned";
        break;
      case GameEvent_Type.END_RACK_PTS:
        play = "End rack points";
        break;
      case GameEvent_Type.TIME_PENALTY:
        play = "Time penalty";
        break;
      case GameEvent_Type.END_RACK_PENALTY:
        play = "End rack penalty";
        break;
      case GameEvent_Type.UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
        play = "Unsuccessful challenge turn loss";
        break;
      default:
        play = `Unknown move type (${currentEvent.type})`;
    }

    return { player, play, score: cumulativeScore, playScore };
  };

  const { player, play, score, playScore } = getTurnDisplayInfo();

  const handleCopyLink = useCallback(() => {
    const url = baseUrl ? new URL(baseUrl) : new URL(window.location.href);
    // Use both turn and comments parameters for full context
    // turn shows the board position AFTER the move (eventNumber + 2)
    // comments opens the drawer for the specific event (eventNumber + 1)
    const commentEventNumber = eventNumber + 1;
    const turnPositionNumber = eventNumber + 2;
    url.searchParams.set("turn", turnPositionNumber.toString());
    url.searchParams.set("comments", commentEventNumber.toString());

    navigator.clipboard
      .writeText(url.toString())
      .then(() => {
        message.success("Link copied to clipboard!");
      })
      .catch(() => {
        message.error("Failed to copy link");
      });
  }, [eventNumber, baseUrl]);

  const handleAddComment = useCallback(
    (comment: string) => {
      onAddComment(comment);
    },
    [onAddComment],
  );

  const drawerTitle = (
    <div className="comments-drawer-header">
      <Title level={4} style={{ margin: 0 }}>
        Comments for Turn {turnIndex + 1}
      </Title>
    </div>
  );

  return (
    <Drawer
      title={drawerTitle}
      placement="right"
      onClose={onClose}
      open={visible}
      width={600}
      className="comments-drawer"
      extra={
        <Button
          type="text"
          icon={<CloseOutlined />}
          onClick={onClose}
          size="small"
        />
      }
    >
      {/* Turn Context */}
      <div className="turn-context">
        <div className="turn-info">
          <Text strong>{player}</Text>
          <Text className="play-info">{play}</Text>
          <Text className="score-info">
            +{playScore} (Total: {score})
          </Text>
        </div>
      </div>

      {/* Comments Section */}
      <div className="comments-section">
        <div className="comments-wrapper">
          <Comments
            comments={eventComments}
            addComment={handleAddComment}
            editComment={onEditComment}
            deleteComment={onDeleteComment}
          />
        </div>
      </div>

      {/* Footer with Share */}
      <div className="drawer-footer">
        <Button
          icon={<LinkOutlined />}
          onClick={handleCopyLink}
          size="small"
          type="dashed"
        >
          Copy Link to Discussion
        </Button>
      </div>
    </Drawer>
  );
};
