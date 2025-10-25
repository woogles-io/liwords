import React, { useState, useEffect, useCallback } from "react";
import {
  Button,
  Radio,
  Card,
  Alert,
  Space,
  Typography,
  Divider,
  Modal,
  Row,
  Col,
} from "antd";
import {
  CameraOutlined,
  DesktopOutlined,
  MobileOutlined,
} from "@ant-design/icons";
import { QRCodeSVG } from "qrcode.react";
import { useTournamentStoreContext } from "../../store/store";
import { useLoginStateStoreContext } from "../../store/store";
import {
  generateCameraShareUrl,
  generateScreenshotShareUrl,
  generatePhoneQRUrl,
  generateCameraViewUrl,
  generateScreenshotViewUrl,
} from "./vdo_ninja_utils";
import { MonitoringData, StreamStatus } from "./types";
import { TournamentService } from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { flashError, useClient } from "../../utils/hooks/connect";
import { ActionType } from "../../actions/actions";

const { Paragraph, Text } = Typography;

type Props = {
  visible: boolean;
  onClose: () => void;
};

export const MonitoringModal = ({ visible, onClose }: Props) => {
  const { tournamentContext, dispatchTournamentContext } =
    useTournamentStoreContext();
  const { loginState } = useLoginStateStoreContext();
  const tClient = useClient(TournamentService);

  // Read UI state from tournament context (persists across navigation!)
  const uiState = tournamentContext.monitoringUIState;
  const deviceType = uiState?.deviceType || "phone";

  // Get keys from backend data (backend is source of truth)
  const ownData = tournamentContext.monitoringData?.[loginState.userID];
  const cameraKey = ownData?.cameraKey || "";
  const screenshotKey = ownData?.screenshotKey || "";
  const keysReady = !!(cameraKey && screenshotKey);

  // Update context when UI state changes
  const updateUIState = useCallback(
    (updates: Partial<typeof uiState>) => {
      dispatchTournamentContext({
        actionType: ActionType.SetMonitoringUIState,
        payload: updates,
      });
    },
    [dispatchTournamentContext],
  );

  // Initialize keys and fetch monitoring data when modal opens
  useEffect(() => {
    if (!visible || !tournamentContext.metadata.id || !loginState.loggedIn) {
      return;
    }

    const initializeAndFetch = async () => {
      try {
        // First, ensure keys are initialized on the backend
        await tClient.initializeMonitoringKeys({
          tournamentId: tournamentContext.metadata.id,
        });

        // Then fetch the monitoring data (which now includes the initialized keys)
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

        // Update tournament context with monitoring data
        dispatchTournamentContext({
          actionType: ActionType.SetMonitoringData,
          payload: data.reduce(
            (acc, d) => {
              acc[d.userId] = d;
              return acc;
            },
            {} as { [userId: string]: MonitoringData },
          ),
        });

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } catch (e: any) {
        flashError(e);
      }
    };

    initializeAndFetch();
  }, [
    visible,
    tournamentContext.metadata.id,
    loginState.loggedIn,
    tClient,
    dispatchTournamentContext,
  ]);

  const handleStartCamera = async () => {
    const url = generateCameraShareUrl(
      cameraKey,
      tournamentContext.metadata.slug,
      loginState.username,
    );
    window.open(
      url,
      "monitoring-camera",
      "width=600,height=600,menubar=no,toolbar=no",
    );

    // Send API call to mark stream as PENDING (webhooks will mark as ACTIVE)
    try {
      await tClient.requestMonitoringStream({
        tournamentId: tournamentContext.metadata.id,
        streamType: "camera",
      });
    } catch (e) {
      flashError(e);
    }
  };

  const handleStartScreenshot = async () => {
    const url = generateScreenshotShareUrl(
      screenshotKey,
      tournamentContext.metadata.slug,
      loginState.username,
    );
    window.open(
      url,
      "monitoring-screenshot",
      "width=600,height=600,menubar=no,toolbar=no",
    );

    // Send API call to mark stream as PENDING (webhooks will mark as ACTIVE)
    try {
      await tClient.requestMonitoringStream({
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
    loginState.username,
  );

  // Derive streaming status from backend data (backend is source of truth)
  // Only disable buttons when stream is actually ACTIVE (confirmed by webhook)
  // Allow re-clicking if PENDING (in case window was closed or didn't open)
  const cameraStreaming = ownData?.cameraStatus === StreamStatus.ACTIVE;
  const screenshotStreaming = ownData?.screenshotStatus === StreamStatus.ACTIVE;

  // Check if user is logged in
  if (!loginState.loggedIn) {
    return (
      <Modal
        title="Login Required"
        open={visible}
        onCancel={onClose}
        footer={[
          <Button key="close" onClick={onClose}>
            Close
          </Button>,
        ]}
      >
        <Alert
          message="Login Required"
          description="You must be logged in to access monitoring setup."
          type="error"
          showIcon
        />
      </Modal>
    );
  }

  return (
    <Modal
      title={
        <>
          <CameraOutlined /> Tournament Monitoring Setup
        </>
      }
      open={visible}
      onCancel={onClose}
      width="90vw"
      footer={[
        <Button key="close" onClick={onClose}>
          Close
        </Button>,
      ]}
      style={{ top: 20 }}
      bodyStyle={{ maxHeight: "calc(100vh - 200px)", overflowY: "auto" }}
      zIndex={1100}
    >
      <Alert
        message="Monitoring Required"
        description="This tournament requires all participants to share their camera and screen during games. Please set up your monitoring streams below."
        type="info"
        showIcon
        style={{ marginBottom: "24px" }}
      />

      {/* Invigilation Explanation */}
      <Card
        title="About Tournament Invigilation"
        style={{ marginBottom: "24px" }}
      >
        <Paragraph>
          To ensure fair play and maintain the integrity of competitive
          OMGWords, this tournament uses invigilation - a monitoring system
          where tournament directors can observe participants during games. You
          must use a laptop or desktop computer to play in this tournament, with
          a connected webcam or cell phone camera for monitoring.
        </Paragraph>
        <Paragraph>
          Invigilation helps prevent cheating and ensures all players compete on
          a level playing field. This is especially important in tournaments
          with prizes or ratings at stake. By participating, you're helping us
          create a trustworthy competitive environment for everyone.
        </Paragraph>
        <Paragraph>
          <strong>Important:</strong> All participants must share{" "}
          <strong>both their camera AND screen</strong> during tournament games.
          You must complete all three steps below to participate in this
          tournament.
        </Paragraph>
        <Paragraph>
          The setup process below will guide you through sharing your camera and
          screen with tournament directors. Thank you for your cooperation in
          keeping our tournaments fair and enjoyable!
        </Paragraph>
      </Card>

      <Card title="Step 1: Choose Your Device" style={{ marginBottom: "24px" }}>
        <Radio.Group
          value={deviceType}
          onChange={(e) => updateUIState({ deviceType: e.target.value })}
          style={{ width: "100%" }}
        >
          <Space direction="vertical" style={{ width: "100%" }}>
            <Radio value="phone">
              <MobileOutlined /> Use cell phone camera
            </Radio>
            <Radio value="webcam">
              <DesktopOutlined /> Use webcam on this computer
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
            <Alert
              message="Important: External Webcam Required"
              description="You must use an external webcam for monitoring. Built-in laptop/computer cameras cannot be repositioned to show your hands and side view, which is required for proper invigilation."
              type="warning"
              showIcon
              style={{ marginBottom: "16px" }}
            />

            <Row gutter={24}>
              <Col xs={24} md={14}>
                <Paragraph>
                  • Use an <strong>external webcam</strong> (not your computer's
                  built-in camera)
                  <br />• Place your webcam so it faces you from the side
                  <br />• Position it away from your computer to capture your
                  hands and monitor/screen
                  <br />• Use a <strong>tripod or camera holder</strong> to keep
                  the webcam stable and ensure consistent stream quality
                  <br />• Ensure your monitor/screen and hands are clearly
                  visible
                </Paragraph>

                <Paragraph strong style={{ marginTop: "16px" }}>
                  Click the button below. When the window opens:
                </Paragraph>
                <Paragraph>
                  • Select the <strong>Video Source</strong> (your camera)
                  <br />• Select the <strong>Audio Source</strong> (your
                  microphone)
                  <br />• Click <strong>Start</strong>
                  <br />• <strong>Minimize</strong> the widget and don't close
                  it
                  <br />• Return to this page and continue the instructions
                </Paragraph>
              </Col>

              <Col xs={24} md={10}>
                <div style={{ textAlign: "center", padding: "8px" }}>
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
              </Col>
            </Row>

            <Button
              type="primary"
              size="large"
              icon={<CameraOutlined />}
              onClick={handleStartCamera}
              disabled={!keysReady}
              loading={!keysReady}
              block
              style={{ marginTop: "16px" }}
            >
              {!keysReady
                ? "Loading..."
                : cameraStreaming
                  ? "Camera Stream Active"
                  : ownData?.cameraStatus === StreamStatus.PENDING
                    ? "Start Camera Stream (Waiting...)"
                    : "Start Camera Stream"}
            </Button>
            {ownData?.cameraStatus === StreamStatus.PENDING && (
              <Alert
                message="Waiting for camera stream to connect. If the window didn't open, click the button again."
                type="warning"
                showIcon
                style={{ marginTop: "8px" }}
              />
            )}
            {cameraStreaming && (
              <Alert
                message="Camera is streaming. To stop, click the hangup button in the VDO.Ninja widget."
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
            <Row gutter={24}>
              <Col xs={24} md={14}>
                <Paragraph>
                  • Use your phone in <strong>landscape mode/position</strong>{" "}
                  (this makes it easier for monitors to get a better view)
                  <br />• Use a <strong>tripod or phone holder</strong> to keep
                  your device stable and ensure consistent stream quality
                  <br />• We recommend you <strong>
                    plug your phone in
                  </strong>{" "}
                  so it doesn't run out of battery.
                  <br />• Place your phone so the camera faces you from the side
                  <br />• Position it away from your computer to capture your
                  hands and monitor/screen
                  <br />• Ensure your monitor/screen and hands are clearly
                  visible
                </Paragraph>
              </Col>

              <Col xs={24} md={10}>
                <div style={{ textAlign: "center", padding: "8px" }}>
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
              </Col>
            </Row>

            <Paragraph strong style={{ marginTop: "16px" }}>
              Scan this QR code with your phone camera:
            </Paragraph>
            {keysReady ? (
              <>
                <div style={{ textAlign: "center", padding: "20px" }}>
                  <QRCodeSVG value={phoneQRUrl} size={256} />
                </div>
                <Paragraph
                  type="secondary"
                  style={{ textAlign: "center", fontSize: "12px" }}
                >
                  Or manually visit:{" "}
                  <a
                    href={phoneQRUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Open on phone
                  </a>
                </Paragraph>
              </>
            ) : (
              <div style={{ textAlign: "center", padding: "20px" }}>
                <Alert
                  message="Loading QR code..."
                  type="info"
                  showIcon
                  style={{ maxWidth: "300px", margin: "0 auto" }}
                />
              </div>
            )}

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
              <br />• The stream will automatically connect
            </Paragraph>

            {cameraStreaming && (
              <Alert
                message="Camera is streaming. Keep your camera active during the tournament."
                type="success"
                showIcon
                style={{ marginTop: "16px" }}
              />
            )}
            {ownData?.cameraStatus === StreamStatus.PENDING && (
              <Alert
                message="Waiting for phone camera to connect. If it didn't connect, try scanning the QR code again."
                type="warning"
                showIcon
                style={{ marginTop: "16px" }}
              />
            )}
          </Card>
        </>
      )}

      {/* Step 3: Screen Share - shown for both webcam and phone modes */}
      <Card
        title="Step 3: Start Screen Share (on this computer)"
        style={{ marginBottom: "24px" }}
      >
        <Paragraph strong>
          Click the button below. When the window opens:
        </Paragraph>
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
          disabled={!keysReady}
          loading={!keysReady}
          block
          style={{ marginTop: "16px" }}
        >
          {!keysReady
            ? "Loading..."
            : screenshotStreaming
              ? "Screen Share Active"
              : ownData?.screenshotStatus === StreamStatus.PENDING
                ? "Start Screen Share (Waiting...)"
                : "Start Screen Share"}
        </Button>
        {ownData?.screenshotStatus === StreamStatus.PENDING && (
          <Alert
            message="Waiting for screen share to connect. If the window didn't open, click the button again."
            type="warning"
            showIcon
            style={{ marginTop: "8px" }}
          />
        )}
        {screenshotStreaming && (
          <Alert
            message="Screen is sharing. To stop, click the hangup button in the VDO.Ninja widget."
            type="success"
            showIcon
            style={{ marginTop: "8px" }}
          />
        )}
      </Card>

      {(cameraStreaming || screenshotStreaming) && (
        <Card title="Stream Status" style={{ marginBottom: "24px" }}>
          <Space direction="vertical" style={{ width: "100%" }}>
            <Text>
              <CameraOutlined
                style={{
                  marginRight: "8px",
                  color: cameraStreaming ? "green" : "gray",
                }}
              />
              Camera: {cameraStreaming ? "✓ Streaming" : "Not streaming"}
            </Text>
            <Text>
              <DesktopOutlined
                style={{
                  marginRight: "8px",
                  color: screenshotStreaming ? "green" : "gray",
                }}
              />
              Screen: {screenshotStreaming ? "✓ Streaming" : "Not streaming"}
            </Text>
            <Divider />
            <Paragraph type="secondary" style={{ fontSize: "12px" }}>
              Directors can now see your streams. To stop streaming, click the
              hangup button in the VDO.Ninja widget. The system will detect this
              automatically.
            </Paragraph>
          </Space>
        </Card>
      )}

      {/* Preview section */}
      {(cameraStreaming || screenshotStreaming) && (
        <Card title="Preview (Low Quality)" style={{ marginBottom: "24px" }}>
          <Paragraph type="secondary">
            You can verify your streams are working by opening these preview
            links in a new tab:
          </Paragraph>
          <Space direction="vertical">
            {cameraStreaming && (
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
            {screenshotStreaming && (
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
    </Modal>
  );
};
