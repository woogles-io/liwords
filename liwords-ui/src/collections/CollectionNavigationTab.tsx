import React, { useState, useCallback, useEffect } from "react";
import { Button, Typography, Skeleton, List } from "antd";
import {
  LeftOutlined,
  RightOutlined,
  ArrowLeftOutlined,
  BookOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import { useCollectionContext } from "./useCollectionContext";

const { Text } = Typography;

export const CollectionNavigationTab: React.FC = () => {
  const collectionContext = useCollectionContext();
  const [maxHeight, setMaxHeight] = useState<number | undefined>(0);

  const setHeight = useCallback(() => {
    const tabPaneHeight = document.getElementById("chat")?.clientHeight;
    setMaxHeight(tabPaneHeight ? tabPaneHeight - 117 : undefined);
  }, []);

  useEffect(() => {
    setHeight();
  }, [setHeight]);

  useEffect(() => {
    window.addEventListener("resize", setHeight);
    return () => {
      window.removeEventListener("resize", setHeight);
    };
  }, [setHeight]);

  // Don't render if no collection context
  if (!collectionContext) {
    return null;
  }

  const {
    collection,
    currentChapter,
    totalChapters,
    loading,
    error,
    goToPreviousChapter,
    goToNextChapter,
    goBackToCollection,
    goToChapter,
    hasPrevious,
    hasNext,
  } = collectionContext;

  if (error) {
    return (
      <div className="collection-navigation-tab">
        <p>Error loading collection</p>
      </div>
    );
  }

  return (
    <div
      className="collection-navigation-tab"
      style={
        maxHeight
          ? {
              maxHeight: maxHeight,
              overflowY: "auto",
            }
          : undefined
      }
    >
      {/* Collection Header */}
      <div className="collection-info-section">
        <div className="collection-header">
          <BookOutlined className="collection-icon" />
          <div className="collection-details">
            <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
              {loading ? (
                <Skeleton.Input
                  style={{ width: "100%", height: 16 }}
                  active
                  size="small"
                />
              ) : (
                <>
                  <Text className="collection-title" ellipsis>
                    {collection?.title || "Collection"}
                  </Text>
                  <Button
                    type="text"
                    size="small"
                    icon={<ArrowLeftOutlined />}
                    onClick={goBackToCollection}
                    title="Back to Collection"
                  />
                </>
              )}
            </div>
            <Text type="secondary" className="chapter-indicator">
              Chapter {currentChapter} of {totalChapters}
            </Text>
          </div>
        </div>
      </div>

      {/* Chapter List */}
      <div className="chapter-list-section">
        {loading ? (
          <Skeleton active paragraph={{ rows: 3 }} />
        ) : (
          <List
            size="small"
            dataSource={collection?.games || []}
            renderItem={(game, index) => {
              const chapterNum = index + 1;
              const isActive = chapterNum === currentChapter;

              return (
                <List.Item
                  className={`chapter-list-item ${isActive ? "active" : ""}`}
                  onClick={() => goToChapter && goToChapter(chapterNum)}
                  style={{ cursor: "pointer" }}
                >
                  <div className="chapter-item-content">
                    <div className="chapter-item-header">
                      <PlayCircleOutlined className="chapter-icon" />
                      <Text strong={isActive} className="chapter-title">
                        Ch. {chapterNum}:{" "}
                        {game.chapterTitle || `Chapter ${chapterNum}`}
                      </Text>
                    </div>
                    <Text type="secondary" className="chapter-type">
                      {game.isAnnotated ? "Annotated" : "Game Record"}
                    </Text>
                  </div>
                </List.Item>
              );
            }}
          />
        )}
      </div>

      {/* Navigation Controls */}
      <div className="collection-controls">
        <div className="chapter-navigation">
          <div className="button-group">
            <Button
              icon={<LeftOutlined />}
              onClick={goToPreviousChapter}
              disabled={!hasPrevious}
              className="nav-button"
            >
              Previous
            </Button>
            <Button
              type="primary"
              icon={<RightOutlined />}
              onClick={goToNextChapter}
              disabled={!hasNext}
              className="nav-button"
            >
              Next
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
