import React, { useState, useCallback } from "react";
import {
  Card,
  Button,
  Typography,
  Divider,
  message,
  Input,
  Space,
  App,
} from "antd";
import {
  LeftOutlined,
  RightOutlined,
  BookOutlined,
  EditOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import { DndProvider } from "react-dnd";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";
import { useLoginStateStoreContext } from "../store/store";
import { useClient } from "../utils/hooks/connect";
import { CollectionsService } from "../gen/api/proto/collections_service/collections_service_pb";
import { DraggableChapterItem } from "./DraggableChapterItem";
import { MultiBackend } from "react-dnd-multi-backend";
import { HTML5Backend } from "react-dnd-html5-backend";
import { TouchBackend } from "react-dnd-touch-backend";
import { TouchTransition, MouseTransition } from "react-dnd-multi-backend";
import { useNavigate } from "react-router";

const { Title, Text, Paragraph } = Typography;

// Multi-backend configuration for drag and drop
const multiBackendOptions = {
  backends: [
    {
      id: "html5",
      backend: HTML5Backend,
      transition: MouseTransition,
    },
    {
      id: "touch",
      backend: TouchBackend,
      options: {
        enableMouseEvents: true,
        delayTouchStart: 200,
      },
      transition: TouchTransition,
    },
  ],
};

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
  const navigate = useNavigate();
  const { modal } = App.useApp();
  const [reordering, setReordering] = useState(false);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [editedTitle, setEditedTitle] = useState(collection.title);
  const [editedDescription, setEditedDescription] = useState(
    collection.description,
  );

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

  const handleUpdateCollection = useCallback(async () => {
    const trimmedTitle = editedTitle.trim();

    if (!trimmedTitle) {
      message.error("Collection title cannot be empty");
      return;
    }

    try {
      await collectionsClient.updateCollection({
        collectionUuid: collection.uuid,
        title: trimmedTitle,
        description: editedDescription,
        public: collection.public,
      });

      message.success("Collection updated");
      setIsEditingTitle(false);
      await handleRefreshCollection();
    } catch (err) {
      console.error("Failed to update collection:", err);
      message.error("Failed to update collection");
    }
  }, [
    collectionsClient,
    collection.uuid,
    collection.public,
    editedTitle,
    editedDescription,
    handleRefreshCollection,
  ]);

  const handleDeleteCollection = useCallback(() => {
    modal.confirm({
      title: <span className="readable-text-color">Delete Collection</span>,
      content: (
        <span className="readable-text-color">
          Are you sure you want to delete "{collection.title}"? This action
          cannot be undone.
        </span>
      ),
      okText: "Delete",
      okType: "danger",
      cancelText: "Cancel",
      onOk: async () => {
        try {
          await collectionsClient.deleteCollection({
            collectionUuid: collection.uuid,
          });
          message.success("Collection deleted");
          navigate("/editor");
        } catch (err) {
          console.error("Failed to delete collection:", err);
          message.error("Failed to delete collection");
        }
      },
    });
  }, [collectionsClient, collection.uuid, collection.title, navigate, modal]);

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
        {isEditingTitle ? (
          <Space
            direction="vertical"
            style={{ width: "100%", marginBottom: "16px" }}
          >
            <Input
              value={editedTitle}
              onChange={(e) => setEditedTitle(e.target.value)}
              placeholder="Collection title"
              autoFocus
            />
            <Input.TextArea
              value={editedDescription}
              onChange={(e) => setEditedDescription(e.target.value)}
              placeholder="Collection description (optional)"
              rows={3}
            />
            <Space>
              <Button
                type="primary"
                size="small"
                onClick={handleUpdateCollection}
              >
                Save
              </Button>
              <Button
                size="small"
                onClick={() => {
                  setIsEditingTitle(false);
                  setEditedTitle(collection.title);
                  setEditedDescription(collection.description);
                }}
              >
                Cancel
              </Button>
            </Space>
          </Space>
        ) : (
          <>
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "flex-start",
                width: "100%",
              }}
            >
              <Title level={4} style={{ margin: 0 }}>
                {collection.title}
              </Title>
              {isOwner && (
                <Space size="small">
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={() => setIsEditingTitle(true)}
                    title="Edit collection"
                  />
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={handleDeleteCollection}
                    title="Delete collection"
                  />
                </Space>
              )}
            </div>
            {collection.description && (
              <Paragraph
                type="secondary"
                ellipsis={{ rows: 2, expandable: true }}
              >
                {collection.description}
              </Paragraph>
            )}
            <Text type="secondary">
              Collection by {collection.creatorUsername} • {totalChapters}{" "}
              chapters
            </Text>
          </>
        )}
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
        <DndProvider backend={MultiBackend} options={multiBackendOptions}>
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
