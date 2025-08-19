import React, { useEffect, useState, useCallback } from "react";
import { Card, List, Tag, Typography, Space } from "antd";
import { FolderOutlined, BookOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { useClient } from "../utils/hooks/connect";
import {
  CollectionsService,
  Collection,
} from "../gen/api/proto/collections_service/collections_service_pb";

const { Text } = Typography;

interface UserCollectionsCardProps {
  userUuid: string;
  isOwnProfile: boolean;
}

export const UserCollectionsCard: React.FC<UserCollectionsCardProps> = ({
  userUuid,
  isOwnProfile,
}) => {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(false);
  const collectionsClient = useClient(CollectionsService);

  const fetchUserCollections = useCallback(async () => {
    setLoading(true);
    try {
      const response = await collectionsClient.getUserCollections({
        userUuid,
        limit: 20,
        offset: 0,
      });
      setCollections(response.collections || []);
    } catch (err) {
      console.error("Failed to fetch user collections:", err);
    } finally {
      setLoading(false);
    }
  }, [collectionsClient, userUuid]);

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
                    {/* <Tag color={collection.public ? "green" : "orange"}>
                      {collection.public ? "Public" : "Private"}
                    </Tag> */}
                    <Text type="secondary">
                      {collection.gameCount || 0} game(s)
                    </Text>
                  </Space>
                </Space>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
};
