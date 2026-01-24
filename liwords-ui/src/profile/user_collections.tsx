import React, { useEffect, useState, useCallback } from "react";
import { Button, Card, List, Typography, Space, Tooltip } from "antd";
import { FolderOutlined, BookOutlined } from "@ant-design/icons";
import moment from "moment";
import { Link } from "react-router";
import { useClient } from "../utils/hooks/connect";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import {
  CollectionsService,
  Collection,
} from "../gen/api/proto/collections_service/collections_service_pb";

const { Text } = Typography;

interface UserCollectionsCardProps {
  userUuid: string;
  isOwnProfile: boolean;
}

const collectionsPageSize = 20;

export const UserCollectionsCard: React.FC<UserCollectionsCardProps> = ({
  userUuid,
  isOwnProfile,
}) => {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(false);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const collectionsClient = useClient(CollectionsService);

  const fetchUserCollections = useCallback(async () => {
    setLoading(true);
    try {
      const response = await collectionsClient.getUserCollections({
        userUuid,
        limit: collectionsPageSize,
        offset,
      });
      const fetchedCollections = response.collections || [];
      setCollections(fetchedCollections);
      // If we got fewer collections than requested, there are no more
      setHasMore(fetchedCollections.length === collectionsPageSize);
    } catch (err) {
      console.error("Failed to fetch user collections:", err);
    } finally {
      setLoading(false);
    }
  }, [collectionsClient, userUuid, offset]);

  const fetchPrev = useCallback(() => {
    setOffset((prev) => Math.max(0, prev - collectionsPageSize));
  }, []);

  const fetchNext = useCallback(() => {
    setOffset((prev) => prev + collectionsPageSize);
  }, []);

  useEffect(() => {
    fetchUserCollections();
  }, [fetchUserCollections]);

  if (collections.length === 0 && !loading) {
    return null;
  }

  return (
    <Card
      title={
        <Space>
          <FolderOutlined />
          {isOwnProfile ? "My Collections" : "Collections"}
        </Space>
      }
      className="game-history-card"
      loading={loading}
    >
      <List
        itemLayout="horizontal"
        dataSource={collections}
        renderItem={(collection) => (
          <List.Item
            actions={[
              <Link key="view" to={`/collections/${collection.uuid}`}>
                View
              </Link>,
            ]}
          >
            <List.Item.Meta
              avatar={
                <BookOutlined style={{ fontSize: 20, paddingLeft: 16 }} />
              }
              title={
                <Link to={`/collections/${collection.uuid}`}>
                  {collection.title}
                </Link>
              }
              description={
                <Space direction="vertical" size="small">
                  {collection.description && (
                    <Text type="secondary">{collection.description}</Text>
                  )}
                  <Space>
                    <Text type="secondary">
                      {collection.gameCount || 0} game(s)
                    </Text>
                    <Text type="secondary">•</Text>
                    <Tooltip
                      title={
                        collection.updatedAt
                          ? moment(timestampDate(collection.updatedAt)).format(
                              "LLL",
                            )
                          : ""
                      }
                    >
                      <Text type="secondary">
                        Updated{" "}
                        {collection.updatedAt
                          ? moment(
                              timestampDate(collection.updatedAt),
                            ).fromNow()
                          : "never"}
                      </Text>
                    </Tooltip>
                  </Space>
                </Space>
              }
            />
          </List.Item>
        )}
      />
      <div className="game-history-controls" style={{ marginTop: 16 }}>
        <Button disabled={offset === 0} onClick={fetchPrev}>
          Prev
        </Button>
        <Button disabled={!hasMore} onClick={fetchNext}>
          Next
        </Button>
      </div>
    </Card>
  );
};
