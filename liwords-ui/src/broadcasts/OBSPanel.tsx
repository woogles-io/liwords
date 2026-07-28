import React, { useState } from "react";
import {
  Button,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Typography,
  App,
} from "antd";
import {
  OBS_SUFFIXES,
  OBS_FEED_SUFFIXES,
  OBS_USER_ONLY_SUFFIXES,
  type OBSSuffix,
} from "./constants";
import { OBSHelpButton } from "./OBSHelp";
import {
  OBSFieldPreview,
  OBS_SUFFIX_LABELS,
  FONT_FAMILY_MAP,
  defaultSizeForSuffix,
} from "./OBSFieldPreview";

// Font choices offered by the builder's form. The values are keys into
// FONT_FAMILY_MAP, which is what the preview and the emitted URL both use.
const FONT_OPTIONS = [
  { value: "mono", label: "Monospace (default)" },
  { value: "serif", label: "Serif" },
  { value: "sans", label: "Sans-serif" },
  { value: "inter", label: "Inter" },
  { value: "arial", label: "Arial" },
];

type OBSMode = "game" | "slot" | "streamSlot";

type OBSPanelProps = {
  /** Game UUID for direct per-game URLs. */
  gameID?: string;
  broadcastSlug?: string;
  slotName?: string;
  /** Username — required for stream-slot URLs. */
  username?: string;
  /**
   * Name of one of the user's own stream slots. Produces a URL that survives
   * across broadcasts, since it carries only the username and the slot name.
   */
  streamSlotName?: string;
  /**
   * True when that stream slot follows its owner's own latest annotation,
   * which is the only case where "the opponent" is a well-defined player.
   */
  streamSlotFollowsSelf?: boolean;
  /** Which mode to default to. Inferred from props when not specified. */
  defaultMode?: OBSMode;
  /** When true renders only a button; when false (default) renders a Card wrapper. */
  compact?: boolean;
};

export const OBSPanel: React.FC<OBSPanelProps> = ({
  gameID,
  broadcastSlug,
  slotName,
  username,
  streamSlotName,
  streamSlotFollowsSelf = false,
  defaultMode,
  compact = false,
}) => {
  const { notification } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [suffix, setSuffix] = useState<OBSSuffix>("score");
  const [bg, setBg] = useState("#ffffff");
  const [transparentBg, setTransparentBg] = useState(true);
  const [textColor, setTextColor] = useState("#000000");
  const [align, setAlign] = useState<"left" | "center" | "right">("center");
  const [size, setSize] = useState(defaultSizeForSuffix("score"));
  const [font, setFont] = useState("mono");
  const [bold, setBold] = useState(true);
  const [padding, setPadding] = useState(8);
  const [speed, setSpeed] = useState(80);
  const [blankColor, setBlankColor] = useState("#d33300");
  const [wrap, setWrap] = useState(0);

  // Determine which modes are available based on props.
  const hasSlot = !!(broadcastSlug && slotName);
  const hasStreamSlot = !!(username && streamSlotName);
  const hasGame = !!gameID;

  // Resolve the default mode: explicit prop > stream slot > broadcast slot > game.
  // Stream slot wins because its URL is the one that survives the event ending.
  const resolvedDefault: OBSMode =
    defaultMode ?? (hasStreamSlot ? "streamSlot" : hasSlot ? "slot" : "game");

  const [mode, setMode] = useState<OBSMode>(resolvedDefault);

  // Build the available mode options for the dropdown. The old "user alias"
  // URL shape is gone: "whichever game this user touched last" is now the
  // latest-annotation kind of a named stream slot, which resolves the same way
  // but is explicit and can be re-pointed later.
  const modeOptions: { value: OBSMode; label: string }[] = [];
  if (hasGame) modeOptions.push({ value: "game", label: "This game" });
  if (hasSlot)
    modeOptions.push({
      value: "slot",
      label: `Broadcast slot (${slotName})`,
    });
  if (hasStreamSlot)
    modeOptions.push({
      value: "streamSlot",
      label: `My stream slot (${streamSlotName})`,
    });

  const urlBase =
    mode === "slot"
      ? `/api/broadcasts/obs/${broadcastSlug}/${slotName}`
      : mode === "streamSlot"
        ? `/api/annotations/obs/user/${username}/${streamSlotName}`
        : `/api/annotations/obs/game/${gameID}`;

  // Tournament-standings fields need a broadcast feed behind them, which both
  // slot modes have and plain game mode doesn't. opponent_name is the mirror
  // image: it needs a single tracked player, which only a stream slot
  // following its own owner's annotations has.
  const isSuffixAvailable = (val: OBSSuffix, forMode: OBSMode) => {
    if (OBS_FEED_SUFFIXES.includes(val))
      return forMode === "slot" || forMode === "streamSlot";
    if (OBS_USER_ONLY_SUFFIXES.includes(val))
      return forMode === "streamSlot" && streamSlotFollowsSelf;
    return true;
  };
  const availableSuffixes = OBS_SUFFIXES.filter((s) =>
    isSuffixAvailable(s, mode),
  );

  const handleModeChange = (val: OBSMode) => {
    setMode(val);
    if (!isSuffixAvailable(suffix, val)) {
      setSuffix("score");
      setSize(defaultSizeForSuffix("score"));
    }
  };

  // Which optional controls to show, and which query params buildURL emits.
  // The preview renders its own sample text from the same field name.
  const isMarquee = suffix === "last_play";
  const isBlankField = suffix === "blank1" || suffix === "blank2";
  const isWrappable = suffix === "unseen_tiles";

  const handleSuffixChange = (val: OBSSuffix) => {
    setSuffix(val);
    setSize(defaultSizeForSuffix(val));
  };

  const buildURL = () => {
    const params = new URLSearchParams();
    if (transparentBg) params.set("bg", "transparent");
    else if (bg !== "#ffffff") params.set("bg", bg);
    if (textColor !== "#000000") params.set("color", textColor);
    if (align !== "center") params.set("align", align);
    const defSize = defaultSizeForSuffix(suffix);
    if (size !== defSize) params.set("size", String(size));
    if (font !== "mono") params.set("font", FONT_FAMILY_MAP[font]);
    if (!bold) params.set("bold", "0");
    if (padding !== 8) params.set("padding", String(padding));
    if (isMarquee && speed !== 80) params.set("speed", String(speed));
    if (isBlankField && blankColor !== "#d33300")
      params.set("blank", blankColor);
    if (isWrappable && wrap > 0) params.set("wrap", String(wrap));
    const qs = params.toString();
    return `${window.location.origin}${urlBase}/${suffix}${qs ? "?" + qs : ""}`;
  };

  const copyURL = () => {
    const url = buildURL();
    navigator.clipboard.writeText(url).then(() => {
      notification.success({
        message: "URL copied!",
        description: url,
        duration: 3,
      });
    });
  };

  const openButton = (
    <Button
      size={compact ? "small" : "middle"}
      onClick={() => setModalOpen(true)}
    >
      OBS Builder
    </Button>
  );

  return (
    <>
      {openButton}

      <Modal
        open={modalOpen}
        title={
          <Space>
            OBS URL Builder
            <OBSHelpButton section="fields" />
          </Space>
        }
        width={720}
        zIndex={1100}
        onCancel={() => setModalOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setModalOpen(false)}>Close</Button>
            <Button type="primary" onClick={copyURL}>
              Copy URL
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          {/* Context / mode selector — only shown when multiple sources are available */}
          {modeOptions.length > 1 && (
            <div>
              <Typography.Text strong>Source</Typography.Text>
              <br />
              <Select<OBSMode>
                value={mode}
                onChange={handleModeChange}
                style={{ width: "100%", marginTop: 4 }}
                options={modeOptions}
              />
            </div>
          )}
          {/* Field selector */}
          <div>
            <Typography.Text strong>Field</Typography.Text>
            <br />
            <Select<OBSSuffix>
              value={suffix}
              onChange={handleSuffixChange}
              style={{ width: "100%", marginTop: 4 }}
              options={availableSuffixes.map((s) => ({
                value: s,
                label: OBS_SUFFIX_LABELS[s],
              }))}
            />
          </div>

          {/* Customization form */}
          <Space wrap size="middle" align="start">
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Background
              </Typography.Text>
              <Space>
                <input
                  type="color"
                  value={bg}
                  disabled={transparentBg}
                  onChange={(e) => setBg(e.target.value)}
                  style={{
                    width: 60,
                    height: 32,
                    cursor: transparentBg ? "not-allowed" : "pointer",
                    padding: 2,
                    opacity: transparentBg ? 0.4 : 1,
                  }}
                />
                <Switch
                  size="small"
                  checked={transparentBg}
                  onChange={setTransparentBg}
                />
                <Typography.Text style={{ fontSize: 12 }}>
                  Transparent
                </Typography.Text>
              </Space>
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Text color
              </Typography.Text>
              <input
                type="color"
                value={textColor}
                onChange={(e) => setTextColor(e.target.value)}
                style={{ width: 60, height: 32, cursor: "pointer", padding: 2 }}
              />
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Alignment
              </Typography.Text>
              <Select
                value={align}
                onChange={setAlign}
                options={[
                  { value: "left", label: "Left" },
                  { value: "center", label: "Center" },
                  { value: "right", label: "Right" },
                ]}
                style={{ width: 110 }}
              />
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Size (px)
              </Typography.Text>
              <InputNumber
                value={size}
                min={8}
                max={200}
                onChange={(v) => setSize(v ?? defaultSizeForSuffix(suffix))}
                style={{ width: 80 }}
              />
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Font
              </Typography.Text>
              <Select
                value={font}
                onChange={setFont}
                options={FONT_OPTIONS}
                style={{ width: 180 }}
              />
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Bold
              </Typography.Text>
              <Switch checked={bold} onChange={setBold} />
            </div>
            <div>
              <Typography.Text
                style={{ fontSize: 12, display: "block", marginBottom: 4 }}
              >
                Padding (px)
              </Typography.Text>
              <InputNumber
                value={padding}
                min={0}
                max={100}
                onChange={(v) => setPadding(v ?? 8)}
                style={{ width: 80 }}
              />
            </div>
            {isMarquee && (
              <div>
                <Typography.Text
                  style={{ fontSize: 12, display: "block", marginBottom: 4 }}
                >
                  Scroll speed (px/s)
                </Typography.Text>
                <InputNumber
                  value={speed}
                  min={10}
                  max={500}
                  onChange={(v) => setSpeed(v ?? 80)}
                  style={{ width: 90 }}
                />
              </div>
            )}
            {isBlankField && (
              <div>
                <Typography.Text
                  style={{ fontSize: 12, display: "block", marginBottom: 4 }}
                >
                  Blank letter color
                </Typography.Text>
                <input
                  type="color"
                  value={blankColor}
                  onChange={(e) => setBlankColor(e.target.value)}
                  style={{
                    width: 60,
                    height: 32,
                    cursor: "pointer",
                    padding: 2,
                  }}
                />
              </div>
            )}
            {isWrappable && (
              <div>
                <Typography.Text
                  style={{ fontSize: 12, display: "block", marginBottom: 4 }}
                >
                  Wrap at (chars)
                </Typography.Text>
                <InputNumber
                  value={wrap || null}
                  min={1}
                  max={500}
                  placeholder="off"
                  onChange={(v) => setWrap(v ?? 0)}
                  style={{ width: 90 }}
                />
              </div>
            )}
          </Space>

          {/* Live preview */}
          <div>
            <Typography.Text strong>Preview</Typography.Text>
            <Typography.Text
              type="secondary"
              style={{ fontSize: 12, marginLeft: 8 }}
            >
              (sample data, not live)
            </Typography.Text>
            <OBSFieldPreview
              field={suffix}
              transparent={transparentBg}
              bg={bg}
              color={textColor}
              align={align}
              size={size}
              font={font}
              bold={bold}
              padding={padding}
              speed={speed}
              blankColor={blankColor}
              wrap={wrap}
            />
          </div>

          {/* URL display */}
          <div>
            <Typography.Text strong>URL</Typography.Text>
            <Typography.Paragraph
              copyable
              style={{ fontSize: 12, marginTop: 4, wordBreak: "break-all" }}
            >
              {buildURL()}
            </Typography.Paragraph>
          </div>
        </Space>
      </Modal>
    </>
  );
};
