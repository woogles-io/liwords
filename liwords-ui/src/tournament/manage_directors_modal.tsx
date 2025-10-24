import React, { useState, useMemo, useCallback } from "react";
import {
  Modal,
  Table,
  Tag,
  Button,
  Space,
  Typography,
  Input,
  Form,
  Checkbox,
  message,
  Popconfirm,
} from "antd";
import { UserOutlined, DeleteOutlined, PlusOutlined } from "@ant-design/icons";
import { useTournamentStoreContext } from "../store/store";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import {
  TournamentPersonsSchema,
  TournamentPersonSchema,
} from "../gen/api/proto/ipc/tournament_pb";
import { create } from "@bufbuild/protobuf";
import { flashError, useClient } from "../utils/hooks/connect";
import { useLoginStateStoreContext } from "../store/store";
import { ActionType } from "../actions/actions";

const { Title, Text } = Typography;

type ManageDirectorsModalProps = {
  visible: boolean;
  onClose: () => void;
};

export const ManageDirectorsModal = ({
  visible,
  onClose,
}: ManageDirectorsModalProps) => {
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();
  const [addingDirector, setAddingDirector] = useState(false);

  const directors = useMemo(() => {
    // Transform director usernames to the expected format
    // HACK: Parse :readonly suffix to determine permission level
    // TODO: Replace with proper permissions field when backend schema is updated
    return tournamentContext.directors.map((username) => {
      const isReadOnly = username.endsWith(":readonly");
      const displayName = isReadOnly ? username.slice(0, -9) : username;
      return {
        username: displayName,
        viewOnly: isReadOnly,
      };
    });
  }, [tournamentContext.directors]);

  // Refetch tournament metadata to get updated directors list
  const refetchMetadata = useCallback(async () => {
    try {
      const response = await tClient.getTournamentMetadata({
        id: tournamentContext.metadata.id,
      });

      if (response.metadata) {
        dispatchTournamentContext({
          actionType: ActionType.SetTourneyMetadata,
          payload: {
            directors: response.directors,
            metadata: response.metadata,
          },
        });
      }
    } catch (e) {
      // Silent failure - the WebSocket message should eventually update it
      console.error("Failed to refetch metadata:", e);
    }
  }, [tClient, tournamentContext.metadata.id, dispatchTournamentContext]);

  const handleAddDirector = async (values: {
    username: string;
    viewOnly: boolean;
  }) => {
    try {
      setAddingDirector(true);

      // HACK: Rating field: -1=Read-only Director, otherwise Full Director
      // TODO: Replace with proper permissions field when backend schema is updated
      const request = create(TournamentPersonsSchema, {
        id: tournamentContext.metadata.id,
        division: "",
        persons: [
          create(TournamentPersonSchema, {
            id: values.username,
            rating: values.viewOnly ? -1 : 0,
          }),
        ],
      });

      await tClient.addDirectors(request);
      message.success(
        `Added ${values.username} as ${values.viewOnly ? "read-only" : "full"} director`,
      );
      form.resetFields();

      // Refetch metadata to update directors list
      await refetchMetadata();
    } catch (e) {
      flashError(e);
    } finally {
      setAddingDirector(false);
    }
  };

  const handleRemoveDirector = async (username: string) => {
    try {
      const request = create(TournamentPersonsSchema, {
        id: tournamentContext.metadata.id,
        division: "",
        persons: [
          create(TournamentPersonSchema, {
            id: username,
            rating: 0, // Doesn't matter for removal
          }),
        ],
      });

      await tClient.removeDirectors(request);
      message.success(`Removed ${username} as director`);

      // Refetch metadata to update directors list
      await refetchMetadata();
    } catch (e) {
      flashError(e);
    }
  };

  const columns = [
    {
      title: "Username",
      dataIndex: "username",
      key: "username",
      render: (username: string) => <strong>{username}</strong>,
    },
    {
      title: "Permissions",
      dataIndex: "viewOnly",
      key: "viewOnly",
      render: (viewOnly: boolean) => (
        <Tag color={viewOnly ? "orange" : "green"}>
          {viewOnly ? "Read-only" : "Full Director"}
        </Tag>
      ),
    },
    {
      title: "Actions",
      key: "actions",
      render: (_: unknown, record: { username: string; viewOnly: boolean }) => (
        <Popconfirm
          title="Remove director?"
          description={`Are you sure you want to remove ${record.username} as a director?`}
          onConfirm={() => handleRemoveDirector(record.username)}
          okText="Remove"
          cancelText="Cancel"
        >
          <Button size="small" danger icon={<DeleteOutlined />}>
            Remove
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Modal
      open={visible}
      onCancel={onClose}
      footer={null}
      width={800}
      title={
        <Space>
          <UserOutlined />
          <Title level={4} style={{ margin: 0 }}>
            Manage Directors
          </Title>
        </Space>
      }
    >
      <Text type="secondary" style={{ display: "block", marginBottom: "16px" }}>
        Add or remove tournament directors. Read-only directors can access the
        monitoring dashboard and reset participant streams, but cannot modify
        tournament settings or start rounds.
      </Text>

      <Form
        form={form}
        onFinish={handleAddDirector}
        layout="inline"
        style={{ marginBottom: "16px" }}
      >
        <Form.Item
          name="username"
          rules={[{ required: true, message: "Username is required" }]}
        >
          <Input placeholder="Username" prefix={<UserOutlined />} />
        </Form.Item>
        <Form.Item name="viewOnly" valuePropName="checked" initialValue={false}>
          <Checkbox>Read-only (monitoring access only)</Checkbox>
        </Form.Item>
        <Form.Item style={{ marginTop: "-4px" }}>
          <Button
            type="primary"
            htmlType="submit"
            icon={<PlusOutlined />}
            loading={addingDirector}
          >
            Add Director
          </Button>
        </Form.Item>
      </Form>

      <Table
        dataSource={directors}
        columns={columns}
        rowKey="username"
        pagination={false}
        locale={{
          emptyText: "No directors have been added to this tournament yet.",
        }}
      />
    </Modal>
  );
};
