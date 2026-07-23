import { useEffect } from "react";
import {
  useLocalStorage,
  useLocalStorageBool,
} from "../utils/use_local_storage";

// Chat display preferences, configured under Settings > Secret features and
// stored in localStorage (so they are per-device). They are applied as CSS
// custom properties that chat.scss reads: --chat-font-scale multiplies the
// message text size, and --chat-timestamp-visibility is the timestamp's base
// visibility (hover still reveals it). Because they go through useLocalStorage
// (useSyncExternalStore + the "storage" event), changing one in any tab updates
// every open tab live, without a refresh.

export const CHAT_FONT_SCALE_KEY = "chatFontScale";
export const CHAT_ALWAYS_TIMESTAMPS_KEY = "chatAlwaysShowTimestamps";

// Stored as strings so they can go through useLocalStorage directly.
export const CHAT_FONT_SCALE_OPTIONS: Array<{ label: string; value: string }> =
  [
    { label: "Normal", value: "1" },
    { label: "Large", value: "1.15" },
    { label: "Larger", value: "1.3" },
    { label: "Largest", value: "1.5" },
  ];

const applyChatFontScale = (scale: string): void => {
  const n = parseFloat(scale);
  document.body.style.setProperty(
    "--chat-font-scale",
    Number.isFinite(n) && n > 0 ? String(n) : "1",
  );
};

const applyAlwaysShowChatTimestamps = (always: boolean): void => {
  document.body.style.setProperty(
    "--chat-timestamp-visibility",
    always ? "visible" : "hidden",
  );
};

// Applies the chat display prefs as CSS variables and re-applies whenever they
// change in this or any other tab. Call it from the chat surface.
export const useApplyChatPrefs = (): void => {
  const [scale] = useLocalStorage(CHAT_FONT_SCALE_KEY, "1");
  const [alwaysTimestamps] = useLocalStorageBool(CHAT_ALWAYS_TIMESTAMPS_KEY);
  useEffect(() => {
    applyChatFontScale(scale);
  }, [scale]);
  useEffect(() => {
    applyAlwaysShowChatTimestamps(alwaysTimestamps);
  }, [alwaysTimestamps]);
};

// Backs the in-chat A-/A+ control. It reads and writes the same font-scale
// pref, so it stays in sync with the Settings dropdown and every open tab, and
// steps through the discrete sizes.
export const useChatFontScaleControl = (): {
  atMin: boolean;
  atMax: boolean;
  decrease: () => void;
  increase: () => void;
} => {
  const [scale, setScale] = useLocalStorage(CHAT_FONT_SCALE_KEY, "1");
  const values = CHAT_FONT_SCALE_OPTIONS.map((o) => o.value);
  const idx = Math.max(0, values.indexOf(scale));
  return {
    atMin: idx <= 0,
    atMax: idx >= values.length - 1,
    decrease: () => setScale(values[Math.max(0, idx - 1)]),
    increase: () => setScale(values[Math.min(values.length - 1, idx + 1)]),
  };
};
