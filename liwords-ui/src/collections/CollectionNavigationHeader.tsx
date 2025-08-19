import React from "react";
import { Button, Space, Typography, Skeleton } from "antd";
import {
  LeftOutlined,
  RightOutlined,
  ArrowLeftOutlined,
  BookOutlined,
} from "@ant-design/icons";
import { useCollectionContext } from "./useCollectionContext";

const { Text } = Typography;

export const CollectionNavigationHeader: React.FC = () => {
  const collectionContext = useCollectionContext();

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
    hasPrevious,
    hasNext,
  } = collectionContext;

  if (error) {
    return null; // Silently fail if collection can't be loaded
  }

  return (
    <div className="collection-navigation-header">
      <div className="collection-nav-content">
        {/* Back to Collection */}
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={goBackToCollection}
          className="back-button"
        >
          Back to Collection
        </Button>

        {/* Collection Info */}
        <div className="collection-info">
          <BookOutlined className="collection-icon" />
          <Space size="small">
            {loading ? (
              <Skeleton.Input
                style={{ width: 120, height: 16 }}
                active
                size="small"
              />
            ) : (
              <Text className="collection-title" ellipsis>
                {collection?.title || "Collection"}
              </Text>
            )}
            <Text type="secondary" className="chapter-indicator">
              Ch. {currentChapter}/{totalChapters}
            </Text>
          </Space>
        </div>

        {/* Chapter Navigation */}
        <Space size="small" className="chapter-navigation">
          <Button
            type="text"
            icon={<LeftOutlined />}
            onClick={goToPreviousChapter}
            disabled={!hasPrevious}
            size="small"
          >
            Previous
          </Button>
          <Button
            type="primary"
            icon={<RightOutlined />}
            onClick={goToNextChapter}
            disabled={!hasNext}
            size="small"
          >
            Next
          </Button>
        </Space>
      </div>
    </div>
  );
};
