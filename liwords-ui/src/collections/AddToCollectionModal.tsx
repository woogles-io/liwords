import React, { useState, useEffect, useCallback } from "react";
import { Modal, Select, Input, Button, Form, Space, message, Spin } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { useClient } from "../utils/hooks/connect";
import { CollectionsService } from "../gen/api/proto/collections_service/collections_service_pb";
import { Collection } from "../gen/api/proto/collections_service/collections_service_pb";
import { useLoginStateStoreContext } from "../store/store";

const { Option } = Select;

interface AddToCollectionModalProps {
  visible: boolean;
  gameId: string;
  isAnnotated: boolean;
  onClose: () => void;
  onSuccess?: (collectionUuid: string) => void;
}

export const AddToCollectionModal: React.FC<AddToCollectionModalProps> = ({
  visible,
  gameId,
  isAnnotated,
  onClose,
  onSuccess,
}) => {
  const [form] = Form.useForm();
  const collectionsClient = useClient(CollectionsService);
  const { loginState } = useLoginStateStoreContext();

  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [creatingNew, setCreatingNew] = useState(false);

  const fetchUserCollections = useCallback(async () => {
    setLoading(true);
    try {
      const response = await collectionsClient.getUserCollections({
        userUuid: loginState.userID,
        limit: 50,
        offset: 0,
      });
      setCollections(response.collections || []);
    } catch (err) {
      console.error("Failed to fetch collections:", err);
      message.error("Failed to load your collections");
    } finally {
      setLoading(false);
    }
  }, [collectionsClient, loginState.userID]);

  // Fetch user's collections when modal opens
  useEffect(() => {
    if (visible && loginState.userID) {
      fetchUserCollections();
    }
  }, [visible, loginState.userID, fetchUserCollections]);

  const handleSubmit = async (values: {
    collectionId: string;
    newTitle?: string;
    newDescription?: string;
    chapterTitle?: string;
  }) => {
    setSubmitting(true);

    try {
      let collectionUuid = values.collectionId;

      // If creating a new collection
      if (collectionUuid === "new") {
        if (!values.newTitle) {
          message.error("Please enter a title for the new collection");
          setSubmitting(false);
          return;
        }

        const createResponse = await collectionsClient.createCollection({
          title: values.newTitle,
          description: values.newDescription || "",
          public: true,
        });

        collectionUuid = createResponse.collectionUuid;
        message.success("Collection created successfully");
      }

      // Add game to collection
      await collectionsClient.addGameToCollection({
        collectionUuid,
        gameId,
        chapterTitle: values.chapterTitle || "",
        isAnnotated,
      });

      message.success("Game added to collection");
      onSuccess?.(collectionUuid);
      form.resetFields();
      onClose();
    } catch (err) {
      console.error("Failed to add game to collection:", err);
      message.error("Failed to add game to collection");
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    form.resetFields();
    setCreatingNew(false);
    onClose();
  };

  return (
    <Modal
      title="Add Game to Collection"
      open={visible}
      onCancel={handleCancel}
      footer={null}
      width={500}
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{ collectionId: "" }}
      >
        <Form.Item
          name="collectionId"
          label="Select Collection"
          rules={[{ required: true, message: "Please select a collection" }]}
        >
          <Select
            placeholder="Choose a collection or create new"
            onChange={(value) => setCreatingNew(value === "new")}
            loading={loading}
            notFoundContent={
              loading ? <Spin size="small" /> : "No collections found"
            }
            allowClear
          >
            <Option value="new">
              <PlusOutlined /> Create New Collection
            </Option>
            {collections
              .filter(
                (collection) =>
                  collection.title &&
                  collection.title.trim() &&
                  collection.uuid &&
                  collection.uuid.trim(),
              )
              .map((collection) => (
                <Option key={collection.uuid} value={collection.uuid}>
                  {collection.title}
                </Option>
              ))}
          </Select>
        </Form.Item>

        {creatingNew && (
          <>
            <Form.Item
              name="newTitle"
              label="Collection Title"
              rules={[{ required: true, message: "Please enter a title" }]}
            >
              <Input placeholder="e.g., 2024 World Championship" />
            </Form.Item>

            <Form.Item name="newDescription" label="Description (Optional)">
              <Input.TextArea
                rows={3}
                placeholder="Add a description for your collection..."
              />
            </Form.Item>
          </>
        )}

        <Form.Item name="chapterTitle" label="Chapter Title (Optional)">
          <Input placeholder="e.g., Round 1 - Smith vs Jones" />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={submitting}>
              {creatingNew
                ? "Create Collection & Add Game"
                : "Add to Collection"}
            </Button>
            <Button onClick={handleCancel}>Cancel</Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
};
