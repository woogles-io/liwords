import React, { useEffect, useState, useCallback } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Input,
  message,
  Image,
  Tag,
  Card,
  Spin,
} from "antd";
import { CheckOutlined, CloseOutlined, EyeOutlined } from "@ant-design/icons";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  OrganizationService,
  GetPendingVerificationsRequestSchema,
  ApproveVerificationRequestSchema,
  RejectVerificationRequestSchema,
  VerificationRequestInfo,
} from "../gen/api/proto/user_service/user_service_pb";
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

const { TextArea } = Input;

const organizationNames: Record<string, string> = {
  naspa: "NASPA",
  wespa: "WESPA",
  absp: "ABSP",
};

export const VerificationQueue = () => {
  const [verifications, setVerifications] = useState<VerificationRequestInfo[]>(
    [],
  );
  const [loading, setLoading] = useState(false);
  const [imageModalVisible, setImageModalVisible] = useState(false);
  const [selectedImage, setSelectedImage] = useState<string | null>(null);
  const [imageLoading, setImageLoading] = useState(false);
  const [actionModalVisible, setActionModalVisible] = useState(false);
  const [actionType, setActionType] = useState<"approve" | "reject">("approve");
  const [selectedRequest, setSelectedRequest] =
    useState<VerificationRequestInfo | null>(null);
  const [notes, setNotes] = useState("");

  const orgClient = useClient(OrganizationService);

  const fetchVerifications = useCallback(async () => {
    setLoading(true);
    try {
      const response = await orgClient.getPendingVerifications(
        create(GetPendingVerificationsRequestSchema, {}),
      );
      setVerifications(response.requests);
    } catch (e) {
      flashError(e);
    } finally {
      setLoading(false);
    }
  }, [orgClient]);

  useEffect(() => {
    fetchVerifications();
  }, [fetchVerifications]);

  const handleViewImage = async (requestId: bigint) => {
    setImageModalVisible(true);
    setImageLoading(true);
    setSelectedImage(null);
    try {
      const response = await orgClient.getVerificationImageUrl({
        requestId: requestId,
      });
      setSelectedImage(response.imageUrl);
    } catch (e) {
      flashError(e);
    } finally {
      setImageLoading(false);
    }
  };

  const handleAction = (
    request: VerificationRequestInfo,
    type: "approve" | "reject",
  ) => {
    setSelectedRequest(request);
    setActionType(type);
    setNotes("");
    setActionModalVisible(true);
  };

  const handleConfirmAction = async () => {
    if (!selectedRequest) return;

    try {
      if (actionType === "approve") {
        await orgClient.approveVerification(
          create(ApproveVerificationRequestSchema, {
            requestId: selectedRequest.requestId,
            notes,
          }),
        );
        message.success("Verification approved");
      } else {
        await orgClient.rejectVerification(
          create(RejectVerificationRequestSchema, {
            requestId: selectedRequest.requestId,
            notes,
          }),
        );
        message.success("Verification rejected");
      }
      setActionModalVisible(false);
      fetchVerifications();
    } catch (e) {
      flashError(e);
    }
  };

  const columns = [
    {
      title: "Submitted",
      dataIndex: "submittedAt",
      key: "submittedAt",
      render: (timestamp: Timestamp | undefined) => {
        if (!timestamp) return "-";
        // Convert protobuf Timestamp (seconds + nanos) to JS Date
        const date = new Date(Number(timestamp.seconds) * 1000);
        return date.toLocaleString("en-US", {
          year: "numeric",
          month: "short",
          day: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        });
      },
      width: 180,
    },
    {
      title: "Username",
      dataIndex: "username",
      key: "username",
      width: 150,
    },
    {
      title: "Organization",
      dataIndex: "organizationCode",
      key: "organizationCode",
      render: (code: string) => organizationNames[code] || code,
      width: 120,
    },
    {
      title: "Member ID",
      dataIndex: "memberId",
      key: "memberId",
      width: 120,
    },
    {
      title: "Full Name",
      dataIndex: "fullName",
      key: "fullName",
      width: 200,
    },
    {
      title: "Title",
      dataIndex: "title",
      key: "title",
      render: (title: string) =>
        title ? <Tag color="blue">{title}</Tag> : "-",
      width: 100,
    },
    {
      title: "ID Photo",
      key: "image",
      render: (_: unknown, record: VerificationRequestInfo) => (
        <Button
          icon={<EyeOutlined />}
          size="small"
          onClick={() => handleViewImage(record.requestId)}
        >
          View
        </Button>
      ),
      width: 100,
    },
    {
      title: "Actions",
      key: "actions",
      render: (_: unknown, record: VerificationRequestInfo) => (
        <Space>
          <Button
            type="primary"
            icon={<CheckOutlined />}
            size="small"
            onClick={() => handleAction(record, "approve")}
          >
            Approve
          </Button>
          <Button
            danger
            icon={<CloseOutlined />}
            size="small"
            onClick={() => handleAction(record, "reject")}
          >
            Reject
          </Button>
        </Space>
      ),
      width: 180,
    },
  ];

  return (
    <Card title="Identity Verification Queue" style={{ margin: "20px" }}>
      <Table
        dataSource={verifications}
        columns={columns}
        rowKey={(record) => record.requestId.toString()}
        loading={loading}
        pagination={{
          pageSize: 20,
          showSizeChanger: false,
        }}
        scroll={{ x: "max-content" }}
      />

      <Modal
        title="View ID Photo"
        open={imageModalVisible}
        onCancel={() => setImageModalVisible(false)}
        footer={null}
        width={800}
      >
        {imageLoading ? (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <Spin size="large" />
            <p style={{ marginTop: 16 }}>Loading image...</p>
          </div>
        ) : selectedImage ? (
          <div>
            <Image
              src={selectedImage}
              alt="ID Photo"
              style={{ width: "100%" }}
              preview={true}
              fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
            />
            <div style={{ marginTop: 10, fontSize: 12, color: "#666" }}>
              URL: {selectedImage}
            </div>
          </div>
        ) : (
          <p>No image available</p>
        )}
      </Modal>

      <Modal
        title={`${actionType === "approve" ? "Approve" : "Reject"} Verification`}
        open={actionModalVisible}
        onOk={handleConfirmAction}
        onCancel={() => setActionModalVisible(false)}
        okText={actionType === "approve" ? "Approve" : "Reject"}
        okButtonProps={{
          danger: actionType === "reject",
        }}
      >
        {selectedRequest && (
          <div>
            <p>
              <strong>Username:</strong> {selectedRequest.username}
            </p>
            <p>
              <strong>Organization:</strong>{" "}
              {organizationNames[selectedRequest.organizationCode] ||
                selectedRequest.organizationCode}
            </p>
            <p>
              <strong>Member ID:</strong> {selectedRequest.memberId}
            </p>
            <p>
              <strong>Full Name:</strong> {selectedRequest.fullName}
            </p>
            {selectedRequest.title && (
              <p>
                <strong>Title:</strong>{" "}
                <Tag color="blue">{selectedRequest.title}</Tag>
              </p>
            )}
            <div style={{ marginTop: 16 }}>
              <label>
                <strong>Notes (optional):</strong>
              </label>
              <TextArea
                rows={4}
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Add any notes about this verification..."
              />
            </div>
          </div>
        )}
      </Modal>
    </Card>
  );
};
