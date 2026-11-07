import { RefObject, useLayoutEffect } from "react";
import { refitOnPageChanges, roomClass, roomInWindow } from "./window_room";

const GROW = roomClass("grow-to-bottom");

// The class a card above a growing one gets while it gives up height to it,
// with the height it keeps as --give-height.
const GIVE = "gives-room";

// A card above a growing one that may give up height for what the growing
// card lists, on a page laid out for a wider window than a phone's: the card,
// found from the growing one; how much of its height it can spare, measured
// at its full height; and the element in the growing card's body that scrolls
// what it lists.
export type Giver = {
  card: (growing: HTMLElement) => HTMLElement | null;
  spare: (card: HTMLElement) => number;
  list: string;
};

// While on, a card's body is no shorter than its own height and as tall as
// what it holds needs, up to its room in the window (window_room.ts): the
// body gets the class grow-to-bottom, its own height as --grow-own, and that
// room as --grow-room, measured as it turns on and again whenever the window
// or the page changes size, or one of the page's affixes pins or lets go.
// Given a giver, the card above gives up what the body needs past that room,
// as far as it can spare, and keeps the most it gave until the window changes
// size, so that the card above does not move as what the body lists changes;
// on a page laid out as on a phone, the card then keeps its own height.
// Turning it on or off moves what the card holds, so the page hears a resize
// then, and whatever sizes itself to the window measures again.
export const useGrowToWindowBottom = (
  card: RefObject<HTMLElement | null>,
  on: boolean,
  giver?: Giver,
) => {
  useLayoutEffect(() => {
    const root = card.current;
    const body = root?.querySelector<HTMLElement>(":scope > .ant-card-body");
    if (!on || !root || !body) {
      return;
    }
    const above = giver?.card(root) ?? null;
    let held = 0;
    // what fit sets, taken off
    const unfit = () => {
      above?.classList.remove(GIVE);
      above?.style.removeProperty("--give-height");
      body.classList.remove(GROW);
      body.style.removeProperty("--grow-own");
      body.style.removeProperty("--grow-room");
    };
    const fit = () => {
      // measured with the card above at its full height, and the body grown,
      // so that what it needs is what it lists and not its own height
      above?.classList.remove(GIVE);
      body.classList.add(GROW);
      const { own, phone, room } = roomInWindow(body, root);
      if (giver && phone) {
        unfit();
        return;
      }
      let give = 0;
      if (above && giver) {
        const list = body.querySelector<HTMLElement>(giver.list);
        const need = list
          ? body.getBoundingClientRect().height +
            list.scrollHeight -
            list.clientHeight
          : own;
        give = Math.max(
          0,
          Math.min(
            giver.spare(above),
            Math.max(held, need - Math.max(own, room)),
          ),
        );
        held = give;
      }
      if (above && give > 0) {
        const full = above.getBoundingClientRect().height;
        above.style.setProperty("--give-height", `${full - give}px`);
        above.classList.add(GIVE);
      } else {
        above?.style.removeProperty("--give-height");
      }
      body.style.setProperty("--grow-own", `${own}px`);
      body.style.setProperty("--grow-room", `${Math.max(own, room + give)}px`);
    };
    fit();
    // a resize by the window itself, not one the page tells, lets the card
    // above take back what it gave
    const resized = (e: Event) => {
      if (e.isTrusted) {
        held = 0;
      }
    };
    window.addEventListener("resize", resized);
    const stop = refitOnPageChanges(fit);
    // what the body lists changes as it is analyzed, and the card above
    // changes size as it collapses or expands
    const listed = new MutationObserver(fit);
    listed.observe(body, {
      childList: true,
      subtree: true,
      characterData: true,
    });
    const sized =
      above && typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(fit)
        : null;
    // what the card above can spare changes as it shows another of its tabs;
    // the class this hook gives the card itself is left out, or each fit
    // would set off the next
    const tabbed = new MutationObserver((records) => {
      if (records.some((r) => r.target !== above)) {
        fit();
      }
    });
    if (above) {
      sized?.observe(above);
      tabbed.observe(above, { attributeFilter: ["class"], subtree: true });
    }
    window.dispatchEvent(new Event("resize"));
    return () => {
      window.removeEventListener("resize", resized);
      stop();
      listed.disconnect();
      sized?.disconnect();
      tabbed.disconnect();
      unfit();
      window.dispatchEvent(new Event("resize"));
    };
  }, [card, on, giver]);
};
