import React, { useRef, useState, useCallback } from "react";
import { useDrag, useDrop } from "react-dnd";
import { List, Space, Typography, Input, message } from "antd";
import { HolderOutlined, EditOutlined } from "@ant-design/icons";
import { CollectionGame } from "../gen/api/proto/collections_service/collections_service_pb";
import { useClient } from "../utils/hooks/connect";
import { CollectionsService } from "../gen/api/proto/collections_service/collections_service_pb";

const { Text } = Typography;

const CHAPTER_ITEM_TYPE = "CHAPTER_ITEM";

interface DragItem {
  index: number;
  gameId: string;
  type: string;
}

interface DraggableChapterItemProps {
  game: CollectionGame;
  index: number;
  chapterNum: number;
  isActive: boolean;
  isOwner: boolean;
  collectionUuid: string;
  onChapterChange: (chapter: number) => void;
  onMoveChapter: (fromIndex: number, toIndex: number) => void;
  onChapterUpdate?: () => void;
}

export const DraggableChapterItem: React.FC<DraggableChapterItemProps> = ({
  game,
  index,
  chapterNum,
  isActive,
  isOwner,
  collectionUuid,
  onChapterChange,
  onMoveChapter,
  onChapterUpdate,
}) => {
  const ref = useRef<HTMLDivElement>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editingTitle, setEditingTitle] = useState(
    game.chapterTitle || `Chapter ${chapterNum}`,
  );
  const [originalTitle, setOriginalTitle] = useState(
    game.chapterTitle || `Chapter ${chapterNum}`,
  );
  const collectionsClient = useClient(CollectionsService);

  const [{ isDragging }, drag] = useDrag({
    type: CHAPTER_ITEM_TYPE,
    item: (): DragItem => ({
      index,
      gameId: game.gameId,
      type: CHAPTER_ITEM_TYPE,
    }),
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
    canDrag: isOwner,
  });

  const [{ isOver }, drop] = useDrop({
    accept: CHAPTER_ITEM_TYPE,
    collect: (monitor) => ({
      isOver: monitor.isOver(),
    }),
    drop: (item: DragItem) => {
      if (!ref.current) return;

      const fromIndex = item.index;
      const toIndex = index;

      if (fromIndex !== toIndex) {
        onMoveChapter(fromIndex, toIndex);
      }
    },
    canDrop: () => isOwner,
  });

  // Combine drag and drop refs
  drag(drop(ref));

  const handleClick = () => {
    if (!isDragging && !isEditing) {
      onChapterChange(chapterNum);
    }
  };

  const handleSaveTitle = useCallback(async () => {
    const trimmedTitle = editingTitle.trim();

    if (!trimmedTitle) {
      message.error("Chapter title cannot be empty");
      return;
    }

    // Check if the title has actually changed
    if (trimmedTitle === originalTitle) {
      // No change, just exit edit mode without API call
      setIsEditing(false);
      return;
    }

    try {
      await collectionsClient.updateChapterTitle({
        collectionUuid,
        gameId: game.gameId,
        chapterTitle: trimmedTitle,
      });

      setIsEditing(false);
      setOriginalTitle(trimmedTitle); // Update the original title for future comparisons
      onChapterUpdate?.();
      message.success("Chapter title updated");
    } catch (err) {
      console.error("Failed to update chapter title:", err);
      message.error("Failed to update chapter title");
    }
  }, [
    collectionsClient,
    collectionUuid,
    game.gameId,
    editingTitle,
    originalTitle,
    onChapterUpdate,
  ]);

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditingTitle(originalTitle);
  };

  const handleStartEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    const currentTitle = game.chapterTitle || `Chapter ${chapterNum}`;
    setOriginalTitle(currentTitle);
    setEditingTitle(currentTitle);
    setIsEditing(true);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSaveTitle();
    } else if (e.key === "Escape") {
      handleCancelEdit();
    }
  };

  const itemStyle: React.CSSProperties = {
    cursor: isOwner ? (isDragging ? "grabbing" : "grab") : "pointer",
    opacity: isDragging ? 0.5 : 1,
    position: "relative",
  };

  return (
    <div ref={ref}>
      <List.Item
        className={`chapter-item ${isActive ? "active" : ""} ${isDragging ? "dragging" : ""}`}
        onClick={handleClick}
        style={itemStyle}
      >
        <Space direction="vertical" style={{ width: "100%" }}>
          <div
            style={{ display: "flex", alignItems: "flex-start", gap: "8px" }}
          >
            {isOwner && (
              <HolderOutlined
                className="drag-handle"
                style={{
                  color: "#888",
                  cursor: "grab",
                  opacity: isDragging ? 0 : 0.6,
                  transition: "opacity 0.2s",
                  paddingTop: "4px",
                }}
              />
            )}
            <img
              src={`/gameimg/${game.gameId}-v2.png`}
              alt={`Preview of ${game.chapterTitle || `Chapter ${chapterNum}`}`}
              style={{
                width: "60px",
                height: "60px",
                objectFit: "cover",
                borderRadius: "4px",
                flexShrink: 0,
              }}
            />
            <div style={{ flex: 1, minWidth: 0 }}>
              {isEditing ? (
                <Input
                  size="small"
                  value={editingTitle}
                  onChange={(e) => setEditingTitle(e.target.value)}
                  onBlur={handleSaveTitle}
                  onKeyDown={handleKeyPress}
                  style={{ width: "100%" }}
                  autoFocus
                  placeholder={`Chapter ${chapterNum}`}
                />
              ) : (
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "4px",
                  }}
                >
                  <Text strong={isActive} style={{ flex: 1 }}>
                    {chapterNum}. {game.chapterTitle || `Chapter ${chapterNum}`}
                  </Text>
                  {isOwner && (
                    <EditOutlined
                      style={{
                        color: "#888",
                        cursor: "pointer",
                        opacity: 0.6,
                        fontSize: "12px",
                      }}
                      onClick={handleStartEdit}
                      title="Edit chapter title"
                    />
                  )}
                </div>
              )}
              {game.isAnnotated && (
                <Text type="secondary" style={{ fontSize: "12px" }}>
                  Annotated Game
                </Text>
              )}
            </div>
          </div>
        </Space>
      </List.Item>
    </div>
  );
};
