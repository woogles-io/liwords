import React, { useLayoutEffect, useRef, useState } from "react";
import { type OBSSuffix } from "./constants";

/**
 * A faithful, non-live rendering of one OBS field, using the same sample data
 * and the same styling rules the real browser-source page applies.
 *
 * Shared by the OBS URL builder (as its live preview) and by the manual, so
 * documentation shows exactly what a viewer would get rather than a screenshot
 * that quietly goes out of date. The marquee maths below is the twin of
 * setMarqueeSpeed in pkg/broadcasts/obs_handler.go and must stay in sync.
 */

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

export type OBSFieldPreviewProps = {
  field: OBSSuffix;
  /** Checkerboard behind the text, as OBS shows transparency. Default true. */
  transparent?: boolean;
  bg?: string;
  color?: string;
  align?: "left" | "center" | "right";
  /** Defaults to the field's own default size, as the server does. */
  size?: number;
  /** Key into FONT_FAMILY_MAP. */
  font?: string;
  bold?: boolean;
  padding?: number;
  /** Marquee scroll speed in px/sec (last_play only). */
  speed?: number;
  /** Colour for blank-designated letters (blank1/blank2 only). */
  blankColor?: string;
  /** Wrap width in characters (unseen_tiles only). */
  wrap?: number;
  minHeight?: number;
};

export const OBSFieldPreview: React.FC<OBSFieldPreviewProps> = ({
  field,
  transparent = true,
  bg = "#ffffff",
  color = "#000000",
  align = "center",
  size,
  font = "mono",
  bold = true,
  padding = 8,
  speed = 80,
  blankColor = "#d33300",
  wrap = 0,
  minHeight = 80,
}) => {
  const resolvedSize = size ?? defaultSizeForSuffix(field);
  const isMarquee = field === "last_play";
  const isBlankField = field === "blank1" || field === "blank2";
  const isWrappable = field === "unseen_tiles";

  const rawSampleText = OBS_SAMPLE_DATA[field];
  const sampleText =
    isWrappable && wrap > 0 ? wrapAtWidth(rawSampleText, wrap) : rawSampleText;

  const containerStyle: React.CSSProperties = {
    background: transparent
      ? "repeating-conic-gradient(#e0e0e0 0% 25%, #ffffff 0% 50%) 0 0 / 16px 16px"
      : bg,
    padding: `${padding}px`,
    overflow: "hidden",
    width: "100%",
    minHeight,
    display: "flex",
    alignItems: "center",
    border: "1px solid #ccc",
    borderRadius: 4,
  };

  const textStyle: React.CSSProperties = {
    fontFamily: FONT_FAMILY_MAP[font],
    fontWeight: bold ? "bold" : "normal",
    color,
    fontSize: resolvedSize,
    whiteSpace: "pre",
    lineHeight: 1.2,
    width: "100%",
    textAlign: align,
  };

  // Measure the marquee's inner element (which holds two duplicated copies of
  // the text, back to back, for a seamless loop -- see obs_handler.go's
  // .mq-inner/.mq-seg) after it renders, to derive the animation duration in
  // px/s. Duration is based on ONE copy's width (half the measured total).
  // useLayoutEffect fires synchronously before the browser paints, so the two
  // renders (first with duration=null, animation off, then with the computed
  // duration) are batched into a single paint by the browser.
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
    // expects. Baking the head start into obs-mq-scroll's own keyframes instead
    // would break its loop math -- it only repeats seamlessly when it travels
    // exactly one copy-width per cycle, so a jump would reappear at every
    // restart. See obs_handler.go's setMarqueeSpeed for the Go twin.
    const introDur = resolvedSize / speed; // 1em resolves to the size (px) value
    const loopDur = copyWidth / speed;
    setMarqueeAnimation(
      `obs-mq-intro ${introDur}s linear 1 forwards, obs-mq-scroll ${loopDur}s linear ${introDur}s infinite`,
    );
  }, [isMarquee, speed, sampleText, resolvedSize]);

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

  return (
    <div style={containerStyle}>
      {isMarquee ? (
        <>
          <style>{marqueeKeyframes}</style>
          <div style={{ width: "100%", overflow: "hidden" }}>
            <div
              ref={marqueeInnerRef}
              style={{
                display: "inline-flex",
                whiteSpace: "nowrap",
                // Sets the base for the keyframes' 1em head-start so it tracks
                // the configured text size.
                fontSize: resolvedSize,
                ...(marqueeAnimation !== null && {
                  animation: marqueeAnimation,
                }),
              }}
            >
              {[0, 1].map((i) => (
                <span
                  key={i}
                  style={{
                    ...textStyle,
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
        <span style={textStyle}>
          <BlankPreview text={sampleText} blankColor={blankColor} />
        </span>
      ) : (
        <span style={textStyle}>{sampleText}</span>
      )}
    </div>
  );
};
