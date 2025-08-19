import React, { useState, useCallback } from "react";
import { Card, Button, Typography, Divider, message } from "antd";
import { LeftOutlined, RightOutlined, BookOutlined } from "@ant-design/icons";
import { DndProvider } from "react-dnd";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import { useClient } from "../utils/hooks/connect";
import { CollectionsService } from "../gen/api/proto/collections_service/collections_service_pb";
import { DraggableChapterItem } from "./DraggableChapterItem";
import { HTML5Backend } from "react-dnd-html5-backend";

const { Title, Text, Paragraph } = Typography;

interface CollectionNavigationProps {
  collection: Collection;
  currentChapter: number;
  onChapterChange: (chapter: number) => void;
  onCollectionUpdate?: (updatedCollection: Collection) => void;
}

export const CollectionNavigation: React.FC<CollectionNavigationProps> = ({
  collection,
  currentChapter,
  onChapterChange,
  onCollectionUpdate,
}) => {
  const { loginState } = useLoginStateStoreContext();
  const collectionsClient = useClient(CollectionsService);
  const [reordering, setReordering] = useState(false);

  const handleRefreshCollection = useCallback(async () => {
    try {
      const response = await collectionsClient.getCollection({
        collectionUuid: collection.uuid,
      });
      if (response.collection) {
        onCollectionUpdate?.(response.collection);
      }
    } catch (err) {
      console.error("Failed to refresh collection:", err);
    }
  }, [collectionsClient, collection.uuid, onCollectionUpdate]);

  const totalChapters = collection.games.length;
  const hasPrevious = currentChapter > 1;
  const hasNext = currentChapter < totalChapters;

  // Check if current user is the collection owner
  const isOwner = loginState.userID === collection.creatorUuid;

  const handlePrevious = () => {
    if (hasPrevious) {
      onChapterChange(currentChapter - 1);
    }
  };

  const handleNext = () => {
    if (hasNext) {
      onChapterChange(currentChapter + 1);
    }
  };

  // Handle chapter reordering
  const handleMoveChapter = useCallback(
    async (fromIndex: number, toIndex: number) => {
      if (!isOwner || reordering) return;

      setReordering(true);

      // Optimistic update: reorder locally first
      const newGames = [...collection.games];
      const [movedGame] = newGames.splice(fromIndex, 1);
      newGames.splice(toIndex, 0, movedGame);

      // Create updated collection for immediate UI feedback
      const optimisticCollection = {
        ...collection,
        games: newGames,
      };

      onCollectionUpdate?.(optimisticCollection);

      try {
        // Prepare gameIds in new order for API call
        const gameIds = newGames.map((game) => game.gameId);

        await collectionsClient.reorderGames({
          collectionUuid: collection.uuid,
          gameIds,
        });

        message.success("Chapter order updated");
      } catch (error) {
        console.error("Failed to reorder chapters:", error);
        message.error("Failed to reorder chapters");

        // Rollback on error
        onCollectionUpdate?.(collection);
      } finally {
        setReordering(false);
      }
    },
    [collection, collectionsClient, isOwner, onCollectionUpdate, reordering],
  );

  return (
    <Card className="collection-navigation">
      <div className="collection-header">
        <BookOutlined style={{ fontSize: "24px", marginBottom: "8px" }} />
        <Title level={4}>{collection.title}</Title>
        {collection.description && (
          <Paragraph type="secondary" ellipsis={{ rows: 2, expandable: true }}>
            {collection.description}
          </Paragraph>
        )}
        <Text type="secondary">
          by {collection.creatorUsername} • {totalChapters} chapters
        </Text>
      </div>

      <Divider />

      <div className="chapter-controls">
        <div className="chapter-nav-row">
          <Button
            icon={<LeftOutlined />}
            onClick={handlePrevious}
            disabled={!hasPrevious}
            size="small"
          >
            Prev
          </Button>
          <Text strong className="chapter-text">
            Ch. {currentChapter}/{totalChapters}
          </Text>
          <Button
            icon={<RightOutlined />}
            onClick={handleNext}
            disabled={!hasNext}
            type="primary"
            size="small"
          >
            Next
          </Button>
        </div>
      </div>

      <Divider />

      <div className="chapter-list">
        <Title level={5}>
          Chapters{" "}
          {isOwner && (
            <Text type="secondary" style={{ fontSize: "12px" }}>
              (drag to reorder, click edit to rename)
            </Text>
          )}
        </Title>
        <DndProvider backend={HTML5Backend}>
          <div className="chapter-items">
            {collection.games.map((game, index) => {
              const chapterNum = index + 1;
              const isActive = chapterNum === currentChapter;

              return (
                <DraggableChapterItem
                  key={game.gameId}
                  game={game}
                  index={index}
                  chapterNum={chapterNum}
                  isActive={isActive}
                  isOwner={isOwner && !reordering}
                  collectionUuid={collection.uuid}
                  onChapterChange={onChapterChange}
                  onMoveChapter={handleMoveChapter}
                  onChapterUpdate={handleRefreshCollection}
                />
              );
            })}
          </div>
        </DndProvider>
      </div>
    </Card>
  );
};
