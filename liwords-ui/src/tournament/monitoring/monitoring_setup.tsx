import React, { useState, useEffect } from "react";
import { Button, Radio, Card, Alert, Space, Typography, Divider } from "antd";
import {
  CameraOutlined,
  DesktopOutlined,
  MobileOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router";
import { QRCodeSVG } from "qrcode.react";
import { useTournamentStoreContext } from "../../store/store";
import { useLoginStateStoreContext } from "../../store/store";
import {
  generateMonitoringKey,
  getScreenshotKey,
  generateCameraShareUrl,
  generateScreenshotShareUrl,
  generatePhoneQRUrl,
  generateCameraViewUrl,
  generateScreenshotViewUrl,
} from "./vdo_ninja_utils";
import { DeviceType, MonitoringData } from "./types";
import { DirectorDashboard } from "./director_dashboard";
import { TournamentService } from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../../utils/hooks/connect";

const { Title, Paragraph, Text } = Typography;

export const MonitoringSetup = () => {
  const navigate = useNavigate();
  const { tournamentContext } = useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const tClient = useClient(TournamentService);

  const [deviceType, setDeviceType] = useState<DeviceType>("webcam");
  const [cameraKey, setCameraKey] = useState<string>("");
  const [screenshotKey, setScreenshotKey] = useState<string>("");
  const [cameraActive, setCameraActive] = useState(false);
  const [screenshotActive, setScreenshotActive] = useState(false);
  const [cameraWindow, setCameraWindow] = useState<Window | null>(null);
  const [screenshotWindow, setScreenshotWindow] = useState<Window | null>(null);
  const [phoneConfirmed, setPhoneConfirmed] = useState(false);
  const [monitoringData, setMonitoringData] = useState<MonitoringData[]>([]);
  const [accessDenied, setAccessDenied] = useState(false);

  // Generate keys on mount
  useEffect(() => {
    if (!tournamentContext.metadata.id || !loginState.userID) {
      return;
    }
    const baseKey = generateMonitoringKey(
      tournamentContext.metadata.id,
      loginState.userID,
    );
    setCameraKey(baseKey);
    setScreenshotKey(getScreenshotKey(baseKey));
  }, [tournamentContext.metadata.id, loginState.userID]);

  // Poll for monitoring data every 5 seconds to restore state
  useEffect(() => {
    // Don't fetch if tournament ID is not loaded yet
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
          cameraStartedAt: p.cameraStartedAt
            ? new Date(
                Number(p.cameraStartedAt.seconds) * 1000 +
                  Number(p.cameraStartedAt.nanos) / 1000000,
              )
            : null,
          screenshotStartedAt: p.screenshotStartedAt
            ? new Date(
                Number(p.screenshotStartedAt.seconds) * 1000 +
                  Number(p.screenshotStartedAt.nanos) / 1000000,
              )
            : null,
        }));

        setMonitoringData(data);

        // Restore own state from backend data
        const ownData = data.find((d) => d.userId === loginState.userID);
        if (ownData) {
          if (ownData.cameraStartedAt && !cameraActive && !phoneConfirmed) {
            // Camera was started in a previous session, restore state
            setCameraActive(true);
            setPhoneConfirmed(true);
          }
          if (ownData.screenshotStartedAt && !screenshotActive) {
            // Screenshot was started in a previous session, restore state
            setScreenshotActive(true);
          }
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } catch (e: any) {
        // Check if it's a permission denied error
        if (e.code === "permission_denied") {
          setAccessDenied(true);
        } else {
          flashError(e);
        }
      }
    };

    // Fetch immediately on mount
    fetchMonitoringData();

    // Then poll every 5 seconds
    const interval = setInterval(fetchMonitoringData, 5000);

    return () => clearInterval(interval);
  }, [
    tClient,
    tournamentContext.metadata.id,
    loginState.userID,
    cameraActive,
    phoneConfirmed,
    screenshotActive,
  ]);

  // Check if windows are closed periodically
  useEffect(() => {
    const checkInterval = setInterval(async () => {
      if (cameraWindow && cameraWindow.closed) {
        setCameraActive(false);
        setCameraWindow(null);

        // Notify backend that camera stream has stopped
        try {
          await tClient.stopMonitoringStream({
            tournamentId: tournamentContext.metadata.id,
            streamType: "camera",
          });
        } catch (e) {
          flashError(e);
        }
      }
      if (screenshotWindow && screenshotWindow.closed) {
        setScreenshotActive(false);
        setScreenshotWindow(null);

        // Notify backend that screenshot stream has stopped
        try {
          await tClient.stopMonitoringStream({
            tournamentId: tournamentContext.metadata.id,
            streamType: "screenshot",
          });
        } catch (e) {
          flashError(e);
        }
      }
    }, 1000);

    return () => clearInterval(checkInterval);
  }, [cameraWindow, screenshotWindow, tClient, tournamentContext.metadata.id]);

  const handleStartCamera = async () => {
    const url = generateCameraShareUrl(
      cameraKey,
      tournamentContext.metadata.slug,
    );
    const win = window.open(
      url,
      "monitoring-camera",
      "width=600,height=600,menubar=no,toolbar=no",
    );
    if (win) {
      setCameraWindow(win);
      setCameraActive(true);

      // Send API call to mark stream as started
      try {
        await tClient.startMonitoringStream({
          tournamentId: tournamentContext.metadata.id,
          streamType: "camera",
          streamKey: cameraKey,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  const handleStartScreenshot = async () => {
    const url = generateScreenshotShareUrl(
      screenshotKey,
      tournamentContext.metadata.slug,
    );
    const win = window.open(
      url,
      "monitoring-screenshot",
      "width=600,height=600,menubar=no,toolbar=no",
    );
    if (win) {
      setScreenshotWindow(win);
      setScreenshotActive(true);

      // Send API call to mark stream as started
      try {
        await tClient.startMonitoringStream({
          tournamentId: tournamentContext.metadata.id,
          streamType: "screenshot",
          streamKey: screenshotKey,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  const handleManualStopCamera = async () => {
    setCameraActive(false);
    setPhoneConfirmed(false);
    setCameraWindow(null);

    try {
      await tClient.stopMonitoringStream({
        tournamentId: tournamentContext.metadata.id,
        streamType: "camera",
      });
    } catch (e) {
      flashError(e);
    }
  };

  const handleManualStopScreenshot = async () => {
    setScreenshotActive(false);
    setScreenshotWindow(null);

    try {
      await tClient.stopMonitoringStream({
        tournamentId: tournamentContext.metadata.id,
        streamType: "screenshot",
      });
    } catch (e) {
      flashError(e);
    }
  };

  const phoneQRUrl = generatePhoneQRUrl(
    cameraKey,
    tournamentContext.metadata.slug,
  );

  // Check if current user is a director
  const isDirector = tournamentContext.directors.includes(loginState.username);

  // Show access denied message if user is not a participant
  if (accessDenied) {
    return (
      <div style={{ maxWidth: "800px", margin: "0 auto", padding: "24px" }}>
        <Alert
          message="Access Denied"
          description="You must be registered in this tournament to access the monitoring page."
          type="error"
          showIcon
          style={{ marginBottom: "16px" }}
        />
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`${tournamentContext.metadata.slug}`)}
        >
          Back to Tournament
        </Button>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: "800px", margin: "0 auto", padding: "24px" }}>
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`${tournamentContext.metadata.slug}`)}
        style={{ marginBottom: "16px" }}
      >
        Back to Tournament
      </Button>

      <Title level={2}>
        <CameraOutlined /> Tournament Monitoring Setup
      </Title>

      <Alert
        message="Monitoring Required"
        description="This tournament requires all participants to share their camera and screen during games. Please set up your monitoring streams below."
        type="info"
        showIcon
        style={{ marginBottom: "24px" }}
      />

      {/* Director Dashboard - shown at top for directors */}
      {isDirector && <DirectorDashboard monitoringData={monitoringData} />}

      {/* Recovered session alerts */}
      {cameraActive && !cameraWindow && (
        <Alert
          message="Camera Stream Recovered"
          description={
            <Space direction="vertical" style={{ width: "100%" }}>
              <Text>
                Your camera stream is active from a previous session. If you
                want to stop it, click the button below.
              </Text>
              <Button danger onClick={handleManualStopCamera}>
                Stop Camera Stream
              </Button>
            </Space>
          }
          type="warning"
          showIcon
          style={{ marginBottom: "16px" }}
        />
      )}

      {screenshotActive && !screenshotWindow && (
        <Alert
          message="Screen Share Recovered"
          description={
            <Space direction="vertical" style={{ width: "100%" }}>
              <Text>
                Your screen share is active from a previous session. If you want
                to stop it, click the button below.
              </Text>
              <Button danger onClick={handleManualStopScreenshot}>
                Stop Screen Share
              </Button>
            </Space>
          }
          type="warning"
          showIcon
          style={{ marginBottom: "16px" }}
        />
      )}

      <Card title="Step 1: Choose Your Device" style={{ marginBottom: "24px" }}>
        <Radio.Group
          value={deviceType}
          onChange={(e) => setDeviceType(e.target.value)}
          style={{ width: "100%" }}
        >
          <Space direction="vertical" style={{ width: "100%" }}>
            <Radio value="webcam">
              <DesktopOutlined /> Use webcam on this computer
            </Radio>
            <Radio value="phone">
              <MobileOutlined /> Use cell phone camera
            </Radio>
          </Space>
        </Radio.Group>
      </Card>

      {deviceType === "webcam" && (
        <>
          <Card
            title="Step 2: Position and Start Your Camera"
            style={{ marginBottom: "24px" }}
          >
            <Paragraph>
              • Place your webcam so it faces you from the side
              <br />• Ensure your monitor/screen and hands are visible
            </Paragraph>

            <div style={{ textAlign: "center", margin: "16px 0" }}>
              <img
                src="/approved-camera-position.jpg"
                alt="Approved camera position showing player from the side with monitor visible"
                style={{
                  maxWidth: "100%",
                  height: "auto",
                  borderRadius: "4px",
                }}
              />
            </div>

            <Paragraph strong style={{ marginTop: "16px" }}>
              When the widget opens:
            </Paragraph>
            <Paragraph>
              • Select the <strong>Video Source</strong> (your camera)
              <br />• Select the <strong>Audio Source</strong> (your microphone)
              <br />• Click <strong>Start</strong>
              <br />• <strong>Minimize</strong> the widget and don't close it
              <br />• Return to this page and continue the instructions
            </Paragraph>

            <Button
              type="primary"
              size="large"
              icon={<CameraOutlined />}
              onClick={handleStartCamera}
              disabled={cameraActive}
              block
              style={{ marginTop: "16px" }}
            >
              {cameraActive ? "Camera Stream Active" : "Start Camera Stream"}
            </Button>
            {cameraActive && (
              <Alert
                message="Keep the camera stream window open during the tournament. You can minimize it, but don't close it."
                type="success"
                showIcon
                style={{ marginTop: "8px" }}
              />
            )}
          </Card>
        </>
      )}

      {deviceType === "phone" && (
        <>
          <Card
            title="Step 2: Position and Start Your Phone Camera"
            style={{ marginBottom: "24px" }}
          >
            <Paragraph>
              • Place your phone so the camera faces you from the side
              <br />• Ensure your monitor/screen and hands are visible
            </Paragraph>

            <div style={{ textAlign: "center", margin: "16px 0" }}>
              <img
                src="/approved-camera-position.jpg"
                alt="Approved camera position showing player from the side with monitor visible"
                style={{
                  maxWidth: "100%",
                  height: "auto",
                  borderRadius: "4px",
                }}
              />
            </div>

            <Paragraph strong style={{ marginTop: "16px" }}>
              Scan this QR code with your phone camera:
            </Paragraph>
            <div style={{ textAlign: "center", padding: "20px" }}>
              <QRCodeSVG value={phoneQRUrl} size={256} />
            </div>
            <Paragraph
              type="secondary"
              style={{ textAlign: "center", fontSize: "12px" }}
            >
              Or manually visit:{" "}
              <a href={phoneQRUrl} target="_blank" rel="noopener noreferrer">
                Open on phone
              </a>
            </Paragraph>

            <Paragraph strong style={{ marginTop: "16px" }}>
              When the vdo.ninja window opens on your phone:
            </Paragraph>
            <Paragraph>
              • Select the <strong>Video Source</strong> (choose your{" "}
              <strong>FRONT</strong> cell phone camera)
              <br />• Select the <strong>Audio Source</strong> (Default audio
              source)
              <br />• Click <strong>Start</strong>
              <br />• Don't close the browser tab on your phone
              <br />• Return to this page on your computer and{" "}
              <strong>click the button</strong>
              below to confirm
            </Paragraph>

            <div style={{ marginTop: "16px" }}>
              <Button
                type="primary"
                size="large"
                icon={<CameraOutlined />}
                onClick={async () => {
                  setPhoneConfirmed(true);

                  // Send API call to mark stream as started
                  try {
                    await tClient.startMonitoringStream({
                      tournamentId: tournamentContext.metadata.id,
                      streamType: "camera",
                      streamKey: cameraKey,
                    });
                  } catch (e) {
                    flashError(e);
                  }
                }}
                disabled={phoneConfirmed}
                block
              >
                {phoneConfirmed
                  ? "Phone Camera Confirmed"
                  : "Confirm Phone Camera is Streaming"}
              </Button>
              {phoneConfirmed && (
                <Alert
                  message="Keep your phone camera streaming during the tournament."
                  type="success"
                  showIcon
                  style={{ marginTop: "8px" }}
                />
              )}
            </div>
          </Card>
        </>
      )}

      {/* Step 3: Screen Share - shown for both webcam and phone modes */}
      <Card
        title="Step 3: Start Screen Share (on this computer)"
        style={{ marginBottom: "24px" }}
      >
        <Paragraph strong>When the widget opens:</Paragraph>
        <Paragraph>
          • Click <strong>SELECT SCREEN TO SHARE</strong> in the widget
          <br />• Click <strong>CHOOSE YOUR ENTIRE SCREEN</strong> tab
          <br />• Tick the <strong>"Share audio"</strong> box at the bottom left
          <br />
          • Select the screen where Woogles is running (click "Entire screen")
          <br />• Click the <strong>Share</strong> button
          <br />• <strong>Minimize</strong> the window and don't close it
          <br />• A small window which says "No Audio Source was detected" might
          pop up - click <strong>OK</strong> to close
        </Paragraph>

        <Button
          type="primary"
          size="large"
          icon={<DesktopOutlined />}
          onClick={handleStartScreenshot}
          disabled={screenshotActive}
          block
          style={{ marginTop: "16px" }}
        >
          {screenshotActive ? "Screen Share Active" : "Start Screen Share"}
        </Button>
        {screenshotActive && (
          <Alert
            message="Keep the screen share window open during the tournament. You can minimize it, but don't close it."
            type="success"
            showIcon
            style={{ marginTop: "8px" }}
          />
        )}
      </Card>

      {(cameraActive || screenshotActive || phoneConfirmed) && (
        <Card title="Stream Status" style={{ marginBottom: "24px" }}>
          <Space direction="vertical" style={{ width: "100%" }}>
            <Text>
              <CameraOutlined
                style={{
                  marginRight: "8px",
                  color: cameraActive || phoneConfirmed ? "green" : "gray",
                }}
              />
              Camera:{" "}
              {cameraActive || phoneConfirmed ? "✓ Shared" : "Not shared"}
            </Text>
            <Text>
              <DesktopOutlined
                style={{
                  marginRight: "8px",
                  color: screenshotActive ? "green" : "gray",
                }}
              />
              Screen: {screenshotActive ? "✓ Shared" : "Not shared"}
            </Text>
            <Divider />
            <Paragraph type="secondary" style={{ fontSize: "12px" }}>
              Directors can now see your streams. You can navigate back to the
              tournament page - your streams will stay active as long as{" "}
              {deviceType === "phone" ? "your phone keeps streaming and " : ""}
              the windows remain open.
            </Paragraph>
          </Space>
        </Card>
      )}

      {/* Preview section - could show low-res vdo.ninja view */}
      {(cameraActive || screenshotActive || phoneConfirmed) && (
        <Card title="Preview (Low Quality)" style={{ marginBottom: "24px" }}>
          <Paragraph type="secondary">
            You can verify your streams are working by opening these preview
            links in a new tab:
          </Paragraph>
          <Space direction="vertical">
            {(cameraActive || phoneConfirmed) && (
              <a
                href={generateCameraViewUrl(
                  cameraKey,
                  tournamentContext.metadata.slug,
                )}
                target="_blank"
                rel="noopener noreferrer"
              >
                Preview Camera Stream
              </a>
            )}
            {screenshotActive && (
              <a
                href={generateScreenshotViewUrl(
                  screenshotKey,
                  tournamentContext.metadata.slug,
                )}
                target="_blank"
                rel="noopener noreferrer"
              >
                Preview Screen Share
              </a>
            )}
          </Space>
        </Card>
      )}
    </div>
  );
};
