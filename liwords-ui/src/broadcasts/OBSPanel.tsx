import React, { useState, useRef, useLayoutEffect } from "react";
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
import { OBS_SUFFIXES, type OBSSuffix } from "./constants";

// Human-readable labels for each OBS suffix
export const OBS_SUFFIX_LABELS: Record<OBSSuffix, string> = {
  score: "Combined Score",
  p1_score: "Player 1 Score",
  p2_score: "Player 2 Score",
  unseen_tiles: "Unseen Tiles",
  unseen_count: "Unseen Count",
  last_play: "Last Play (marquee)",
  blank1: "Blank Word 1",
  blank2: "Blank Word 2",
};

// Sample data shown in the preview (no real SSE needed)
const OBS_SAMPLE_DATA: Record<OBSSuffix, string> = {
  score: "345 - 298",
  p1_score: "345",
  p2_score: "298",
  unseen_tiles: "AAEIOU BCDFG HKLMN PRSTT ?",
  unseen_count: "28 tiles\n10 vowels | 17 consonants",
  last_play:
    "     LAST PLAY: Alice 8H GRAFTED 86 86 | to unite with a growing plant",
  blank1: "CoSTARS",
  blank2: "quiZzes",
};

const FONT_OPTIONS = [
  { value: "mono", label: "Monospace (default)" },
  { value: "serif", label: "Serif" },
  { value: "sans", label: "Sans-serif" },
  { value: "inter", label: "Inter" },
  { value: "arial", label: "Arial" },
];

export const FONT_FAMILY_MAP: Record<string, string> = {
  mono: "'Courier New', monospace",
  serif: "Georgia, serif",
  sans: "system-ui, sans-serif",
  inter: "Inter, system-ui, sans-serif",
  arial: "Arial, sans-serif",
};

export function defaultSizeForSuffix(suffix: OBSSuffix): number {
  if (suffix === "score" || suffix === "p1_score" || suffix === "p2_score")
    return 48;
  if (suffix === "blank1" || suffix === "blank2") return 36;
  if (suffix === "last_play") return 24;
  return 20;
}

function wrapAtWidth(text: string, maxWidth: number): string {
  if (!maxWidth) return text;
  const tokens = text.split(" ").filter((t) => t.length > 0);
  const lines: string[] = [];
  let cur = "";
  for (const tok of tokens) {
    if (cur === "") {
      cur = tok;
    } else if (cur.length + 1 + tok.length <= maxWidth) {
      cur += " " + tok;
    } else {
      lines.push(cur);
      cur = tok;
    }
  }
  if (cur) lines.push(cur);
  return lines.join("\n");
}

function BlankPreview({
  text,
  blankColor,
}: {
  text: string;
  blankColor: string;
}) {
  const parts: React.ReactNode[] = [];
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (ch >= "a" && ch <= "z") {
      parts.push(
        <span key={i} style={{ color: blankColor }}>
          {ch}
        </span>,
      );
    } else {
      parts.push(ch);
    }
  }
  return <>{parts}</>;
}

type OBSMode = "game" | "slot" | "user";

type OBSPanelProps = {
  /** Game UUID for direct per-game URLs. */
  gameID?: string;
  broadcastSlug?: string;
  slotName?: string;
  /** Username for user-alias URLs (follows the user's latest annotated game). */
  username?: string;
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
  defaultMode,
  compact = false,
}) => {
  const { notification } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [suffix, setSuffix] = useState<OBSSuffix>("score");
  const [bg, setBg] = useState("#ffffff");
  const [textColor, setTextColor] = useState("#000000");
  const [size, setSize] = useState(defaultSizeForSuffix("score"));
  const [font, setFont] = useState("mono");
  const [bold, setBold] = useState(true);
  const [padding, setPadding] = useState(8);
  const [speed, setSpeed] = useState(80);
  const [blankColor, setBlankColor] = useState("#d33300");
  const [wrap, setWrap] = useState(0);

  // Determine which modes are available based on props.
  const hasSlot = !!(broadcastSlug && slotName);
  const hasUser = !!username;
  const hasGame = !!gameID;

  // Resolve the default mode: explicit prop > broadcast slot > game > user alias.
  const resolvedDefault: OBSMode =
    defaultMode ??
    (hasSlot ? "slot" : hasGame ? "game" : hasUser ? "user" : "game");

  const [mode, setMode] = useState<OBSMode>(resolvedDefault);

  // Build the available mode options for the dropdown.
  const modeOptions: { value: OBSMode; label: string }[] = [];
  if (hasGame) modeOptions.push({ value: "game", label: "This game" });
  if (hasSlot)
    modeOptions.push({
      value: "slot",
      label: `Broadcast slot (${slotName})`,
    });
  if (hasUser)
    modeOptions.push({ value: "user", label: `My alias (${username})` });

  const urlBase =
    mode === "slot"
      ? `/api/broadcasts/obs/${broadcastSlug}/${slotName}`
      : mode === "user"
        ? `/api/annotations/obs/user/${username}`
        : `/api/annotations/obs/game/${gameID}`;

  const isMarquee = suffix === "last_play";
  const isBlankField = suffix === "blank1" || suffix === "blank2";
  const isWrappable = suffix === "unseen_tiles";
  const rawSampleText = OBS_SAMPLE_DATA[suffix];
  const sampleText =
    isWrappable && wrap > 0 ? wrapAtWidth(rawSampleText, wrap) : rawSampleText;

  const handleSuffixChange = (val: OBSSuffix) => {
    setSuffix(val);
    setSize(defaultSizeForSuffix(val));
  };

  const buildURL = () => {
    const params = new URLSearchParams();
    if (bg !== "#ffffff") params.set("bg", bg);
    if (textColor !== "#000000") params.set("color", textColor);
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

  const previewContainerStyle: React.CSSProperties = {
    background: bg,
    padding: `${padding}px`,
    overflow: "hidden",
    width: "100%",
    minHeight: 80,
    display: "flex",
    alignItems: "center",
    border: "1px solid #ccc",
    borderRadius: 4,
    marginTop: 16,
    marginBottom: 8,
  };

  const previewTextStyle: React.CSSProperties = {
    fontFamily: FONT_FAMILY_MAP[font],
    fontWeight: bold ? "bold" : "normal",
    color: textColor,
    fontSize: size,
    whiteSpace: "pre",
    lineHeight: 1.2,
    width: "100%",
  };

  // Measure the marquee span after it renders so we can derive the
  // animation duration in px/s.  useLayoutEffect fires synchronously
  // before the browser paints, so the two renders (first with
  // duration=null → animation off, second with computed duration) are
  // batched into a single paint by the browser.
  const marqueeSpanRef = useRef<HTMLSpanElement>(null);
  const [marqueeDuration, setMarqueeDuration] = useState<number | null>(null);
  useLayoutEffect(() => {
    if (!isMarquee || !marqueeSpanRef.current) {
      setMarqueeDuration(null);
      return;
    }
    const w = marqueeSpanRef.current.offsetWidth;
    if (w > 0) setMarqueeDuration(w / speed);
  }, [isMarquee, speed, sampleText]);

  const marqueeKeyframes = `
    @keyframes obs-mq-scroll {
      from { transform: translateX(0); }
      to   { transform: translateX(-100%); }
    }
  `;

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
        title="OBS URL Builder"
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
                onChange={setMode}
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
              options={OBS_SUFFIXES.map((s) => ({
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
              <input
                type="color"
                value={bg}
                onChange={(e) => setBg(e.target.value)}
                style={{ width: 60, height: 32, cursor: "pointer", padding: 2 }}
              />
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
            <div style={previewContainerStyle}>
              {isMarquee ? (
                <>
                  <style>{marqueeKeyframes}</style>
                  <div style={{ width: "100%", overflow: "hidden" }}>
                    <span
                      ref={marqueeSpanRef}
                      style={{
                        ...previewTextStyle,
                        width: "auto",
                        display: "inline-block",
                        whiteSpace: "nowrap",
                        paddingLeft: "100%",
                        ...(marqueeDuration !== null && {
                          animation: `obs-mq-scroll ${marqueeDuration}s linear infinite`,
                        }),
                      }}
                    >
                      {sampleText}
                    </span>
                  </div>
                </>
              ) : isBlankField ? (
                <span style={previewTextStyle}>
                  <BlankPreview text={sampleText} blankColor={blankColor} />
                </span>
              ) : (
                <span style={previewTextStyle}>{sampleText}</span>
              )}
            </div>
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
