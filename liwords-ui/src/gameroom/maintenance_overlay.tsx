import React from "react";
import { Spin } from "antd";
import {
  LoadingOutlined,
  CheckCircleOutlined,
  RadarChartOutlined,
} from "@ant-design/icons";

type Props = {
  message: string;
  isInGame?: boolean;
  isCorrespondence?: boolean;
  refreshCountdown?: number | null;
  isPinging?: boolean;
};

export const MaintenanceOverlay = ({
  message,
  isInGame,
  isCorrespondence,
  refreshCountdown,
  isPinging,
}: Props) => {
  const isRefreshing =
    refreshCountdown !== null && refreshCountdown !== undefined;

  const renderIcon = () => {
    if (isRefreshing) {
      return <CheckCircleOutlined style={{ fontSize: 32, color: "#52c41a" }} />;
    }
    if (isPinging) {
      return (
        <RadarChartOutlined style={{ fontSize: 32 }} className="pinging-icon" />
      );
    }
    return (
      <Spin indicator={<LoadingOutlined style={{ fontSize: 32 }} spin />} />
    );
  };

  const renderMessage = () => {
    if (isRefreshing) {
      return (
        <div className="maintenance-message">
          App is back! Refreshing in {refreshCountdown} second
          {refreshCountdown !== 1 ? "s" : ""}...
        </div>
      );
    }
    if (isPinging) {
      return (
        <div className="maintenance-message">Checking if app is back...</div>
      );
    }
    return (
      <>
        <div className="maintenance-message">{message}</div>
        {isInGame && (
          <div className="maintenance-submessage">
            Your game will resume automatically.
          </div>
        )}
        {isInGame && !isCorrespondence && (
          <div className="maintenance-submessage">
            Your clock has been paused and no time will be deducted.
          </div>
        )}
      </>
    );
  };

  return (
    <div className="maintenance-overlay">
      <div className="maintenance-content">
        {renderIcon()}
        {renderMessage()}
      </div>
    </div>
  );
};
