import React, { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router";
import { Spin, message, Card, Button } from "antd";
import { PlayCircleOutlined } from "@ant-design/icons";
import { useClient } from "../utils/hooks/connect";
import { CollectionsService } from "../gen/api/proto/collections_service/collections_service_pb";
import {
  Collection,
  CollectionGame,
} from "../gen/api/proto/collections_service/collections_service_pb";
import { CollectionNavigation } from "./CollectionNavigation";
import "./collections.scss";

export const CollectionViewer: React.FC = () => {
  const { uuid, chapterNumber } = useParams<{
    uuid: string;
    chapterNumber?: string;
  }>();
  const navigate = useNavigate();
  const collectionsClient = useClient(CollectionsService);

  const [collection, setCollection] = useState<Collection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Parse chapter number, default to 1
  const currentChapter = parseInt(chapterNumber || "1", 10);

  useEffect(() => {
    const fetchCollection = async () => {
      if (!uuid) return;

      setLoading(true);
      setError(null);

      try {
        const response = await collectionsClient.getCollection({
          collectionUuid: uuid,
        });
        setCollection(response.collection || null);

        // If no chapter number specified, redirect to first chapter
        if (!chapterNumber && response.collection?.games?.length) {
          navigate(`/collections/${uuid}/chapter/1`, { replace: true });
        }
      } catch (err) {
        console.error("Failed to fetch collection:", err);
        setError("Failed to load collection");
        message.error("Failed to load collection");
      } finally {
        setLoading(false);
      }
    };

    fetchCollection();
  }, [uuid, chapterNumber, collectionsClient, navigate]);

  const handleChapterChange = (newChapter: number) => {
    if (!uuid) return;
    navigate(`/collections/${uuid}/chapter/${newChapter}`);
  };

  const handleCollectionUpdate = (updatedCollection: Collection) => {
    setCollection(updatedCollection);
  };

  if (loading) {
    return (
      <div className="collection-viewer-loading">
        <Spin size="large" />
      </div>
    );
  }

  if (error || !collection) {
    return (
      <div className="collection-viewer-error">
        <h2>Error Loading Collection</h2>
        <p>{error || "Collection not found"}</p>
      </div>
    );
  }

  const currentGame = collection.games[currentChapter - 1];

  if (!currentGame) {
    return (
      <div className="collection-viewer-error">
        <h2>Chapter Not Found</h2>
        <p>This collection doesn't have a chapter {currentChapter}</p>
      </div>
    );
  }

  return (
    <div className="collection-viewer">
      <div className="collection-sidebar">
        <CollectionNavigation
          collection={collection}
          currentChapter={currentChapter}
          onChapterChange={handleChapterChange}
          onCollectionUpdate={handleCollectionUpdate}
        />
      </div>
      <div className="collection-content">
        <Card className="chapter-card">
          <div className="chapter-header">
            <h2>{currentGame.chapterTitle || `Chapter ${currentChapter}`}</h2>
            <p className="chapter-type">
              {currentGame.isAnnotated ? "Annotated Game" : "Game Record"}
            </p>
          </div>

          <div className="chapter-actions">
            <Button
              type="primary"
              size="large"
              icon={<PlayCircleOutlined />}
              onClick={() => {
                const baseUrl = currentGame.isAnnotated
                  ? `/anno/${currentGame.gameId}`
                  : `/game/${currentGame.gameId}`;
                const params = new URLSearchParams({
                  collection: uuid!,
                  chapter: currentChapter.toString(),
                  total: collection.games.length.toString(),
                });
                const gameUrl = `${baseUrl}?${params.toString()}`;
                navigate(gameUrl);
              }}
            >
              View Game
            </Button>
          </div>

          <div className="chapter-info">
            <p>
              Click "View Game" to open this game. You can use the collection
              navigation to move between chapters or return here to explore
              other games in this collection.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
};
