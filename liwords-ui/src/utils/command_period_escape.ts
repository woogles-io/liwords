import { isMac } from "./cwgame/common";

// On a Mac, Cmd+. is Esc: it cancels, as it does in the Mac's own dialogs.
// The page hears it as an Escape keydown sent where the Cmd+. was, so every
// dialog, editor, and field that Esc closes or cancels takes it too, with no
// handler of its own changed. Other platforms are left alone (Win+. opens
// the emoji panel there).
export const commandPeriodEscape = (e: KeyboardEvent) => {
  if (e.key !== "." || !e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) {
    return;
  }
  // while an input method composes, Cmd+. is its own to cancel the composition
  if (e.isComposing) return;
  e.preventDefault();
  e.stopPropagation();
  (e.target ?? document).dispatchEvent(
    new KeyboardEvent("keydown", {
      key: "Escape",
      code: "Escape",
      // antd's dialogs read the key code
      keyCode: 27,
      which: 27,
      bubbles: true,
      cancelable: true,
      composed: true,
    }),
  );
};

export const installCommandPeriodEscape = () => {
  if (!isMac()) return;
  document.addEventListener("keydown", commandPeriodEscape, true);
};
