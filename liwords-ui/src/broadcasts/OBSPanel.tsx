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
import {
  OBS_SUFFIXES,
  OBS_SLOT_ONLY_SUFFIXES,
  OBS_USER_ONLY_SUFFIXES,
  type OBSSuffix,
} from "./constants";

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
  p1_name: "Player 1 Name",
  p2_name: "Player 2 Name",
  combined_names: "Both Names (P1 - P2)",
  p1_record: "Player 1 Record (W-L)",
  p2_record: "Player 2 Record (W-L)",
  p1_place: "Player 1 Place",
  p2_place: "Player 2 Place",
  p1_spread: "Player 1 Spread",
  p2_spread: "Player 2 Spread",
  p1_rating: "Player 1 Rating",
  p2_rating: "Player 2 Rating",
  division: "Division",
  tournament: "Tournament Name",
  round: "Round",
  table: "Table Number",
  opponent_name: "Opponent Name",
};

// Sample data shown in the preview (no real SSE needed). Numeric fields use
// the same space-padding the backend applies (see obs.go/obs_tournament.go):
// score right-justifies the left number and left-justifies the right one so
// both hug the " - " while padding lands on the outer edges; rating/spread
// are simple fixed-width right-justify since they're standalone fields.
const OBS_SAMPLE_DATA: Record<OBSSuffix, string> = {
  score: " 45 - 7  ",
  p1_score: "345",
  p2_score: "298",
  unseen_tiles: "AAEIOU BCDFG HKLMN PRSTT ?",
  unseen_count: "28 tiles\n10 vowels | 17 consonants",
  last_play:
    "     LAST PLAY: Alice 8H GRAFTED 86 86 | to unite with a growing plant",
  blank1: "CoSTARS",
  blank2: "quiZzes",
  p1_name: "Alice Smith",
  p2_name: "Bob Jones",
  combined_names: "Alice Smith - Bob Jones",
  p1_record: "6-1",
  p2_record: "5-2",
  p1_place: "2nd",
  p2_place: "4th",
  p1_spread: "+245",
  p2_spread: " -30",
  p1_rating: "1875",
  p2_rating: " 802",
  division: "Championship",
  tournament: "Albany Open 2026",
  round: "7 of 31",
  table: "12",
  opponent_name: "Bob Jones",
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
  if (
    suffix === "p1_name" ||
    suffix === "p2_name" ||
    suffix === "combined_names"
  )
    return 32;
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

  // Tournament-standings fields only resolve in slot mode (the feed is tied
  // to a slot's tournament); opponent_name only makes sense relative to a
  // single tracked player, which only user mode has.
  const isSuffixAvailable = (val: OBSSuffix, forMode: OBSMode) => {
    if (OBS_SLOT_ONLY_SUFFIXES.includes(val)) return forMode === "slot";
    if (OBS_USER_ONLY_SUFFIXES.includes(val)) return forMode === "user";
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

  const previewContainerStyle: React.CSSProperties = {
    background: transparentBg
      ? "repeating-conic-gradient(#e0e0e0 0% 25%, #ffffff 0% 50%) 0 0 / 16px 16px"
      : bg,
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
    textAlign: align,
  };

  // Measure the marquee's inner element (which holds two duplicated copies
  // of the text, back to back, for a seamless loop — see obs_handler.go's
  // .mq-inner/.mq-seg) after it renders, to derive the animation duration in
  // px/s. Duration is based on ONE copy's width (half the measured total).
  // useLayoutEffect fires synchronously before the browser paints, so the
  // two renders (first with duration=null → animation off, second with
  // computed duration) are batched into a single paint by the browser.
  const marqueeInnerRef = useRef<HTMLDivElement>(null);
  const [marqueeAnimation, setMarqueeAnimation] = useState<string | null>(null);
  useLayoutEffect(() => {
    if (!isMarquee || !marqueeInnerRef.current) {
      setMarqueeAnimation(null);
      return;
    }
    const copyWidth = marqueeInnerRef.current.offsetWidth / 2;
    if (copyWidth <= 0) {
      setMarqueeAnimation(null);
      return;
    }
    // Chain a one-shot obs-mq-intro (the 1em head start) into the infinite
    // obs-mq-scroll loop, handing off at the exact position/time obs-mq-scroll
    // expects. Baking the head start into obs-mq-scroll's own keyframes
    // instead would break its loop math — it only repeats seamlessly when it
    // travels exactly one copy-width per cycle, so a jump would reappear at
    // every restart. See obs_handler.go's setMarqueeSpeed for the Go twin.
    const introDur = size / speed; // 1em resolves to the size (px) value
    const loopDur = copyWidth / speed;
    setMarqueeAnimation(
      `obs-mq-intro ${introDur}s linear 1 forwards, obs-mq-scroll ${loopDur}s linear ${introDur}s infinite`,
    );
  }, [isMarquee, speed, sampleText, size]);

  const marqueeKeyframes = `
    @keyframes obs-mq-intro {
      from { transform: translateX(1em); }
      to   { transform: translateX(0); }
    }
    @keyframes obs-mq-scroll {
      from { transform: translateX(0); }
      to   { transform: translateX(-50%); }
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
            <div style={previewContainerStyle}>
              {isMarquee ? (
                <>
                  <style>{marqueeKeyframes}</style>
                  <div style={{ width: "100%", overflow: "hidden" }}>
                    <div
                      ref={marqueeInnerRef}
                      style={{
                        display: "inline-flex",
                        whiteSpace: "nowrap",
                        // Sets the base for the keyframes' 1em head-start so
                        // it tracks the configured text size.
                        fontSize: size,
                        ...(marqueeAnimation !== null && {
                          animation: marqueeAnimation,
                        }),
                      }}
                    >
                      {[0, 1].map((i) => (
                        <span
                          key={i}
                          style={{
                            ...previewTextStyle,
                            width: "auto",
                            flex: "0 0 auto",
                            paddingRight: "2em",
                          }}
                        >
                          {sampleText}
                        </span>
                      ))}
                    </div>
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
