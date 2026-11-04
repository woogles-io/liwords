import { RefObject, useLayoutEffect } from "react";
import { refitOnPageChanges, roomClass, roomInWindow } from "./window_room";

const GROW = roomClass("grow-to-bottom");

// While on, a card's body is no shorter than its own height and as tall as
// what it holds needs, up to its room in the window (window_room.ts): the
// body gets the class grow-to-bottom, its own height as --grow-own, and that
// room as --grow-room, measured as it turns on and again whenever the window
// or the page changes size, or one of the page's affixes pins or lets go.
// Turning it on or off moves what the card holds, so the page hears a resize
// then, and whatever sizes itself to the window measures again.
export const useGrowToWindowBottom = (
  card: RefObject<HTMLElement | null>,
  on: boolean,
) => {
  useLayoutEffect(() => {
    const root = card.current;
    const body = root?.querySelector<HTMLElement>(":scope > .ant-card-body");
    if (!on || !root || !body) {
      return;
    }
    const fit = () => {
      const { own, room } = roomInWindow(body, root);
      body.style.setProperty("--grow-own", `${own}px`);
      body.style.setProperty("--grow-room", `${Math.max(own, room)}px`);
      body.classList.add(GROW);
    };
    fit();
    const stop = refitOnPageChanges(fit);
    window.dispatchEvent(new Event("resize"));
    return () => {
      stop();
      body.classList.remove(GROW);
      body.style.removeProperty("--grow-own");
      body.style.removeProperty("--grow-room");
      window.dispatchEvent(new Event("resize"));
    };
  }, [card, on]);
};
