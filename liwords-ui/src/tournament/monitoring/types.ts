// TypeScript types for tournament monitoring/invigilation

export type DeviceType = "webcam" | "phone";

export type StreamType = "camera" | "screenshot";

export type MonitoringData = {
  userId: string;
  username: string;
  cameraKey?: string; // Only available to directors and self
  screenshotKey?: string; // Only available to directors and self
  cameraStartedAt?: Date | null;
  screenshotStartedAt?: Date | null;
};

export type MonitoringStreamStatus = {
  camera: boolean;
  screenshot: boolean;
};
