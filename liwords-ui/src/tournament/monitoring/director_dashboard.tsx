import React, { useState, useMemo } from "react";
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Typography,
  message,
  Input,
} from "antd";
import {
  CameraOutlined,
  EyeOutlined,
  CopyOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { MonitoringData } from "./types";
import {
  generateCameraViewUrl,
  generateScreenshotViewUrl,
} from "./vdo_ninja_utils";
import { useTournamentStoreContext } from "../../store/store";

const { Title, Text, Link } = Typography;

type DirectorDashboardProps = {
  monitoringData: MonitoringData[];
};

export const DirectorDashboard = ({
  monitoringData,
}: DirectorDashboardProps) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [searchText, setSearchText] = useState("");

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
      render: (_: unknown, record: MonitoringData) => (
        <Space direction="vertical" size="small">
          {record.cameraStartedAt ? (
            <>
              <div>
                <Tag color="green">
                  Shared {new Date(record.cameraStartedAt).toLocaleTimeString()}
                </Tag>
              </div>
              {record.cameraKey && (
                <>
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
            </>
          ) : (
            <Tag color="default">Not shared</Tag>
          )}
        </Space>
      ),
    },
    {
      title: "Screen",
      key: "screenshot",
      render: (_: unknown, record: MonitoringData) => (
        <Space direction="vertical" size="small">
          {record.screenshotStartedAt ? (
            <>
              <div>
                <Tag color="green">
                  Shared{" "}
                  {new Date(record.screenshotStartedAt).toLocaleTimeString()}
                </Tag>
              </div>
              {record.screenshotKey && (
                <>
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
            </>
          ) : (
            <Tag color="default">Not shared</Tag>
          )}
        </Space>
      ),
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

  return (
    <Card
      title={
        <Space>
          <CameraOutlined />
          <Title level={4} style={{ margin: 0 }}>
            Director Monitoring Dashboard
          </Title>
        </Space>
      }
      style={{ marginTop: "24px", marginBottom: "24px" }}
    >
      <Text type="secondary" style={{ display: "block", marginBottom: "16px" }}>
        Monitor all participant streams. Click "View" to open a stream in a new
        window. Status updates every 5 seconds.
      </Text>

      <Input
        placeholder="Search by username..."
        prefix={<SearchOutlined />}
        value={searchText}
        onChange={(e) => setSearchText(e.target.value)}
        allowClear
        style={{ marginBottom: "16px", maxWidth: "400px" }}
      />

      <Table
        dataSource={filteredData}
        columns={columns}
        rowKey="userId"
        pagination={false}
        locale={{
          emptyText: searchText
            ? `No participants found matching "${searchText}"`
            : "No monitoring data available yet. Participants will appear here once they start sharing their streams.",
        }}
      />
    </Card>
  );
};
