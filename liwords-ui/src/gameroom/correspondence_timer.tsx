import React from "react";
import { Tooltip } from "antd";

type Props = {
  timeRemaining: number; // milliseconds
  timeBank?: number | bigint; // milliseconds (bigint from proto int64)
  isOnTurn: boolean;
};

const formatTimeRemaining = (
  milliseconds: number,
): { text: string; className: string } => {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const totalMinutes = Math.floor(totalSeconds / 60);
  const totalHours = Math.floor(totalMinutes / 60);
  const totalDays = Math.round(totalHours / 24);

  // >= 48 hours: show days
  if (totalHours >= 48) {
    return {
      text: `${totalDays} day${totalDays !== 1 ? "s" : ""}`,
      className: "",
    };
  }

  // 24-48 hours: show "1 day"
  if (totalHours >= 24) {
    return { text: "1 day", className: "" };
  }

  // 2-24 hours: show hours
  if (totalHours >= 2) {
    return { text: `${totalHours} hr`, className: "" };
  }

  // 1-2 hours: show "1 hr" with warning color
  if (totalHours >= 1) {
    return { text: "1 hr", className: "warning" };
  }

  // < 1 hour: show minutes in red
  if (totalMinutes >= 1) {
    return { text: `${totalMinutes} min`, className: "urgent" };
  }

  // < 1 minute: show "<1 min" in red
  return { text: "<1 min", className: "urgent" };
};

const formatTimeBank = (milliseconds: number | bigint): string => {
  const ms =
    typeof milliseconds === "bigint" ? Number(milliseconds) : milliseconds;
  const totalSeconds = Math.floor(ms / 1000);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);

  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);

  return parts.length > 0 ? parts.join(" ") : "0m";
};

const formatDeadline = (milliseconds: number): string => {
  const deadline = new Date(Date.now() + milliseconds);
  return `Due: ${deadline.toLocaleString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  })}`;
};

export const CorrespondenceTimer = ({
  timeRemaining,
  timeBank,
  isOnTurn,
}: Props) => {
  const hasTimeBank = timeBank !== undefined && timeBank > 0;

  if (!isOnTurn) {
    return <div className="correspondence-timer">Opponent's turn</div>;
  }

  const { text, className } = formatTimeRemaining(timeRemaining);

  return (
    <div className="correspondence-timer">
      <Tooltip title={formatDeadline(timeRemaining)}>
        <div className={`time-remaining ${className}`}>{text}</div>
      </Tooltip>
      {hasTimeBank && (
        <div className="time-bank">Bank: {formatTimeBank(timeBank)}</div>
      )}
    </div>
  );
};
