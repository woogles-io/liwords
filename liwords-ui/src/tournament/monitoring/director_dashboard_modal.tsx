import React, { useState, useMemo, useEffect } from "react";
import {
  Modal,
  Table,
  Tag,
  Button,
  Space,
  Typography,
  message,
  Input,
  Popconfirm,
} from "antd";
import {
  CameraOutlined,
  EyeOutlined,
  CopyOutlined,
  SearchOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { MonitoringData, StreamStatus } from "./types";
import {
  generateCameraViewUrl,
  generateScreenshotViewUrl,
  generateMultiStreamViewUrl,
} from "./vdo_ninja_utils";
import { useTournamentStoreContext } from "../../store/store";
import { TournamentService } from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../../utils/hooks/connect";

const { Title, Text, Link } = Typography;

type DirectorDashboardModalProps = {
  visible: boolean;
  onClose: () => void;
};

export const DirectorDashboardModal = ({
  visible,
  onClose,
}: DirectorDashboardModalProps) => {
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);
  const [searchText, setSearchText] = useState("");
  const [monitoringData, setMonitoringData] = useState<MonitoringData[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<Set<string>>(
    new Set(),
  );

  // Fetch monitoring data and poll every 2 seconds when modal is visible
  useEffect(() => {
    if (!visible) {
      return; // Stop polling when modal is not visible
    }

    if (!tournamentContext.metadata.id) {
      return;
    }

    const fetchMonitoringData = async () => {
      try {
        const response = await tClient.getTournamentMonitoring({
          tournamentId: tournamentContext.metadata.id,
        });

        // Convert to frontend format
        const data: MonitoringData[] = response.participants.map((p) => ({
          userId: p.userId,
          username: p.username,
          cameraKey: p.cameraKey,
          screenshotKey: p.screenshotKey,
          cameraStatus: p.cameraStatus,
          cameraTimestamp: p.cameraTimestamp
            ? new Date(
                Number(p.cameraTimestamp.seconds) * 1000 +
                  Number(p.cameraTimestamp.nanos) / 1000000,
              )
            : null,
          screenshotStatus: p.screenshotStatus,
          screenshotTimestamp: p.screenshotTimestamp
            ? new Date(
                Number(p.screenshotTimestamp.seconds) * 1000 +
                  Number(p.screenshotTimestamp.nanos) / 1000000,
              )
            : null,
        }));

        setMonitoringData(data);
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } catch (e: any) {
        flashError(e);
      }
    };

    // Fetch immediately when modal opens
    fetchMonitoringData();

    // Poll every 5 seconds while visible
    const interval = setInterval(fetchMonitoringData, 5000);

    return () => {
      clearInterval(interval);
    };
  }, [visible, tClient, tournamentContext.metadata.id]);

  const handleResetStream = async (
    userId: string,
    username: string,
    streamType: "camera" | "screenshot",
  ) => {
    try {
      await tClient.resetMonitoringStream({
        tournamentId: tournamentContext.metadata.id,
        userId: userId,
        streamType: streamType,
      });
      message.success(
        `Reset ${streamType} stream for ${username}. They will need to restart it.`,
      );
    } catch (e) {
      flashError(e);
    }
  };

  // Helper to render status tag based on StreamStatus
  const renderStatusTag = (
    status: StreamStatus,
    timestamp: Date | null | undefined,
  ) => {
    switch (status) {
      case StreamStatus.NOT_STARTED:
        return <Tag color="default">Not Started</Tag>;
      case StreamStatus.PENDING:
        return (
          <Tag color="orange">
            Pending {timestamp ? timestamp.toLocaleTimeString() : ""}
          </Tag>
        );
      case StreamStatus.ACTIVE:
        return (
          <Tag color="green">
            Active {timestamp ? timestamp.toLocaleTimeString() : ""}
          </Tag>
        );
      case StreamStatus.STOPPED:
        return (
          <Tag color="red">
            Stopped {timestamp ? timestamp.toLocaleTimeString() : ""}
          </Tag>
        );
      default:
        return <Tag color="default">Unknown</Tag>;
    }
  };

  const columns = [
    {
      title: "Player",
      dataIndex: "username",
      key: "username",
      render: (username: string) => <strong>{username}</strong>,
    },
    {
      title: "Camera",
      key: "camera",
      render: (_: unknown, record: MonitoringData) => {
        const hasStream =
          record.cameraStatus === StreamStatus.PENDING ||
          record.cameraStatus === StreamStatus.ACTIVE;
        return (
          <Space direction="vertical" size="small">
            <div>
              {renderStatusTag(record.cameraStatus, record.cameraTimestamp)}
            </div>
            {hasStream && record.cameraKey && (
              <>
                <Space>
                  <Button
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => {
                      const url = generateCameraViewUrl(
                        record.cameraKey!,
                        tournamentContext.metadata.slug,
                        false,
                      );
                      window.open(url, "_blank", "noopener,noreferrer");
                    }}
                  >
                    View
                  </Button>
                  <Popconfirm
                    title="Reset camera stream?"
                    description={`This will clear the stream status for ${record.username}. They will need to restart their camera stream. Use this if the stream appears stuck.`}
                    onConfirm={() =>
                      handleResetStream(
                        record.userId,
                        record.username,
                        "camera",
                      )
                    }
                    okText="Reset"
                    cancelText="Cancel"
                  >
                    <Button size="small" danger icon={<ReloadOutlined />}>
                      Reset
                    </Button>
                  </Popconfirm>
                </Space>
                <div style={{ fontSize: "12px" }}>
                  <Link
                    href={generateCameraViewUrl(
                      record.cameraKey!,
                      tournamentContext.metadata.slug,
                      false,
                    )}
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{ wordBreak: "break-all" }}
                  >
                    {generateCameraViewUrl(
                      record.cameraKey!,
                      tournamentContext.metadata.slug,
                      false,
                    )}
                  </Link>
                  <Button
                    type="link"
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={() => {
                      const url = generateCameraViewUrl(
                        record.cameraKey!,
                        tournamentContext.metadata.slug,
                        false,
                      );
                      navigator.clipboard.writeText(url);
                      message.success("Camera link copied to clipboard");
                    }}
                    style={{ padding: "0 4px", height: "auto" }}
                  >
                    Copy
                  </Button>
                </div>
              </>
            )}
          </Space>
        );
      },
    },
    {
      title: "Screen",
      key: "screenshot",
      render: (_: unknown, record: MonitoringData) => {
        const hasStream =
          record.screenshotStatus === StreamStatus.PENDING ||
          record.screenshotStatus === StreamStatus.ACTIVE;
        return (
          <Space direction="vertical" size="small">
            <div>
              {renderStatusTag(
                record.screenshotStatus,
                record.screenshotTimestamp,
              )}
            </div>
            {hasStream && record.screenshotKey && (
              <>
                <Space>
                  <Button
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => {
                      const url = generateScreenshotViewUrl(
                        record.screenshotKey!,
                        tournamentContext.metadata.slug,
                        false,
                      );
                      window.open(url, "_blank", "noopener,noreferrer");
                    }}
                  >
                    View
                  </Button>
                  <Popconfirm
                    title="Reset screen share?"
                    description={`This will clear the stream status for ${record.username}. They will need to restart their screen share. Use this if the stream appears stuck.`}
                    onConfirm={() =>
                      handleResetStream(
                        record.userId,
                        record.username,
                        "screenshot",
                      )
                    }
                    okText="Reset"
                    cancelText="Cancel"
                  >
                    <Button size="small" danger icon={<ReloadOutlined />}>
                      Reset
                    </Button>
                  </Popconfirm>
                </Space>
                <div style={{ fontSize: "12px" }}>
                  <Link
                    href={generateScreenshotViewUrl(
                      record.screenshotKey!,
                      tournamentContext.metadata.slug,
                      false,
                    )}
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{ wordBreak: "break-all" }}
                  >
                    {generateScreenshotViewUrl(
                      record.screenshotKey!,
                      tournamentContext.metadata.slug,
                      false,
                    )}
                  </Link>
                  <Button
                    type="link"
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={() => {
                      const url = generateScreenshotViewUrl(
                        record.screenshotKey!,
                        tournamentContext.metadata.slug,
                        false,
                      );
                      navigator.clipboard.writeText(url);
                      message.success("Screen link copied to clipboard");
                    }}
                    style={{ padding: "0 4px", height: "auto" }}
                  >
                    Copy
                  </Button>
                </div>
              </>
            )}
          </Space>
        );
      },
    },
  ];

  // Sort alphabetically and filter based on search
  const filteredData = useMemo(() => {
    const sorted = [...monitoringData].sort((a, b) =>
      a.username.toLowerCase().localeCompare(b.username.toLowerCase()),
    );

    if (!searchText) {
      return sorted;
    }

    return sorted.filter((data) =>
      data.username.toLowerCase().includes(searchText.toLowerCase()),
    );
  }, [monitoringData, searchText]);

  const handleViewSelectedStreams = () => {
    // Collect all stream keys from selected participants
    const streamKeys: string[] = [];

    selectedUserIds.forEach((userId) => {
      const participant = filteredData.find((p) => p.userId === userId);
      if (!participant) return;

      // Add camera key if stream exists (PENDING, ACTIVE, or STOPPED)
      if (
        participant.cameraKey &&
        participant.cameraStatus !== StreamStatus.NOT_STARTED
      ) {
        streamKeys.push(participant.cameraKey);
      }

      // Add screenshot key if stream exists (PENDING, ACTIVE, or STOPPED)
      if (
        participant.screenshotKey &&
        participant.screenshotStatus !== StreamStatus.NOT_STARTED
      ) {
        streamKeys.push(participant.screenshotKey);
      }
    });

    if (streamKeys.length === 0) {
      message.warning("No active streams found for selected participants");
      return;
    }

    const url = generateMultiStreamViewUrl(streamKeys, false);
    window.open(url, "_blank", "noopener,noreferrer");
    message.success(
      `Opening ${streamKeys.length} streams from ${selectedUserIds.size} participants`,
    );
  };

  return (
    <Modal
      open={visible}
      onCancel={onClose}
      footer={null}
      width="90vw"
      style={{ top: 20 }}
      bodyStyle={{
        maxHeight: "calc(100vh - 200px)",
        overflowY: "auto",
        display: "flex",
        flexDirection: "column",
      }}
      title={
        <Space>
          <CameraOutlined />
          <Title level={4} style={{ margin: 0 }}>
            Director Monitoring Dashboard
          </Title>
        </Space>
      }
    >
      <Text type="secondary" style={{ display: "block", marginBottom: "16px" }}>
        Monitor all participant streams. Click "View" to open a stream in a new
        window. Use "Reset" if a stream appears stuck (participant will need to
        restart). Status updates every 5 seconds.
      </Text>

      <Input
        placeholder="Search by username..."
        prefix={<SearchOutlined />}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        allowClear
        style={{ marginBottom: "16px", maxWidth: "400px" }}
      />

      {selectedUserIds.size > 0 && (
        <Space style={{ marginBottom: "16px" }}>
          <Button
            type="primary"
            icon={<EyeOutlined />}
            onClick={handleViewSelectedStreams}
          >
            View Selected Streams ({selectedUserIds.size})
          </Button>
          <Button onClick={() => setSelectedUserIds(new Set())}>
            Clear Selection
          </Button>
        </Space>
      )}

      <div style={{ flex: 1, overflowY: "auto", minHeight: 0 }}>
        <Table
          dataSource={filteredData}
          columns={columns}
          rowKey="userId"
          pagination={false}
          rowSelection={{
            selectedRowKeys: Array.from(selectedUserIds),
            onChange: (selectedKeys) => {
              setSelectedUserIds(new Set(selectedKeys as string[]));
            },
            getCheckboxProps: (record) => ({
              // Only allow selection if participant has at least one stream that's not NOT_STARTED
              disabled:
                record.cameraStatus === StreamStatus.NOT_STARTED &&
                record.screenshotStatus === StreamStatus.NOT_STARTED,
            }),
          }}
          locale={{
            emptyText: searchText
              ? `No participants found matching "${searchText}"`
              : "No participants are registered in this tournament yet.",
          }}
        />
      </div>
    </Modal>
  );
};
