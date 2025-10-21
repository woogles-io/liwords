import React from "react";
import { Card, Space, Typography } from "antd";
import { CameraOutlined, DesktopOutlined } from "@ant-design/icons";
import { useTournamentStoreContext } from "../../store/store";
import { useLoginStateStoreContext } from "../../store/store";
import { Link } from "react-router";

const { Text } = Typography;

export const MonitoringWidget = () => {
  const { tournamentContext } = useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();

  // Don't show widget if tournament doesn't require monitoring
  if (!tournamentContext.metadata.monitored) {
    return null;
  }

  // Get current user's monitoring data
  const userMonitoringData =
    tournamentContext.monitoringData?.[loginState.userID];

  const cameraActive = !!userMonitoringData?.cameraStartedAt;
  const screenshotActive = !!userMonitoringData?.screenshotStartedAt;

  // Show green if both active, yellow if one active, red if none active
  let status: "success" | "warning" | "error" = "error";
  if (cameraActive && screenshotActive) {
    status = "success";
  } else if (cameraActive || screenshotActive) {
    status = "warning";
  }

  const statusColor =
    status === "success"
      ? "#52c41a"
      : status === "warning"
        ? "#faad14"
        : "#f5222d";

  return (
    <Card
      size="small"
      style={{
        position: "fixed",
        bottom: "20px",
        right: "20px",
        width: "250px",
        zIndex: 1000,
        boxShadow: "0 2px 8px rgba(0,0,0,0.15)",
        borderLeft: `4px solid ${statusColor}`,
      }}
    >
      <Link
        to={`${tournamentContext.metadata.slug}/monitoring`}
        style={{ textDecoration: "none", color: "inherit" }}
      >
        <Space direction="vertical" size="small" style={{ width: "100%" }}>
          <Text strong style={{ fontSize: "12px" }}>
            Monitoring Status
          </Text>
          <Space size="small">
            <CameraOutlined
              style={{ color: cameraActive ? "#52c41a" : "#d9d9d9" }}
            />
            <Text style={{ fontSize: "11px" }}>
              {cameraActive ? "Camera active" : "Camera not shared"}
            </Text>
          </Space>
          <Space size="small">
            <DesktopOutlined
              style={{ color: screenshotActive ? "#52c41a" : "#d9d9d9" }}
            />
            <Text style={{ fontSize: "11px" }}>
              {screenshotActive ? "Screen active" : "Screen not shared"}
            </Text>
          </Space>
          {(!cameraActive || !screenshotActive) && (
            <Text
              type="secondary"
              style={{ fontSize: "10px", fontStyle: "italic" }}
            >
              Click to set up
            </Text>
          )}
        </Space>
      </Link>
    </Card>
  );
};
