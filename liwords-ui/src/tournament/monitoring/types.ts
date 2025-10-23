// TypeScript types for tournament monitoring/invigilation
import { StreamStatus } from "../../gen/api/proto/ipc/tournament_pb";

export type DeviceType = "webcam" | "phone";

export type StreamType = "camera" | "screenshot";

export type MonitoringData = {
  userId: string;
  username: string;
  cameraKey?: string; // Only available to directors and self
  screenshotKey?: string; // Only available to directors and self
  cameraStatus: StreamStatus;
  cameraTimestamp?: Date | null;
  screenshotStatus: StreamStatus;
  screenshotTimestamp?: Date | null;
};

export type MonitoringStreamStatus = {
  camera: boolean;
  screenshot: boolean;
};

// Re-export StreamStatus for convenience
export { StreamStatus };
