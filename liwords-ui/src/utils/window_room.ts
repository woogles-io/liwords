// The gap kept between a card and the window's bottom, and on a phone also
// between the card and what the page pins above it.
const MARGIN = 24;

// What the game page pins at the window's top as it is scrolled, below its
// sticky player cards: the rack, and the examiner's controls.
const AFFIXES = ".rack-affix, .examiner-controls";

// The window's safe areas, top and bottom, as its CSS has them.
const safeAreas = () => {
  const probe = document.createElement("div");
  probe.style.cssText =
    "position: fixed; visibility: hidden; padding-top: env(safe-area-inset-top, 0px); padding-bottom: env(safe-area-inset-bottom, 0px)";
  document.body.appendChild(probe);
  const style = getComputedStyle(probe);
  const px =
    (parseFloat(style.paddingTop) || 0) +
    (parseFloat(style.paddingBottom) || 0);
  probe.remove();
  return px;
};

// How far down the window the game page pins what stays in sight once it is
// scrolled: its sticky player cards, and those of its affixes that have
// pinned.
const pinnedHeight = () => {
  let bottom = 0;
  document
    .querySelectorAll<HTMLElement>(".sticky-player-card-container")
    .forEach((el) => {
      if (el.getClientRects().length) {
        bottom = Math.max(
          bottom,
          (parseFloat(getComputedStyle(el).top) || 0) +
            el.getBoundingClientRect().height,
        );
      }
    });
  document.querySelectorAll<HTMLElement>(AFFIXES).forEach((el) => {
    const pinned = el.querySelector<HTMLElement>(":scope > .ant-affix");
    if (pinned && getComputedStyle(pinned).position === "fixed") {
      bottom = Math.max(bottom, pinned.getBoundingClientRect().bottom);
    }
  });
  return bottom;
};

// The classes that size a card's body to its room, each named by roomClass by
// what gives it; all are taken off the body while its own height is measured,
// so that no two of them measure through each other.
const roomClasses = new Set<string>();
export const roomClass = (name: string) => {
  roomClasses.add(name);
  return name;
};

// Whether the page is laid out as on a phone, by what it shows only there: its
// sticky player cards.
const phoneLayout = () =>
  [
    ...document.querySelectorAll<HTMLElement>(".sticky-player-card-container"),
  ].some((el) => el.getClientRects().length > 0);

// What a card's body has of the window, measured with the page at its top:
// its own height, without any class of roomClass's; whether the page is laid
// out as on a phone; and its room. On a phone, whose page scrolls, that is the
// window below what the page pins once scrolled to the card, less the card's
// head and borders, the safe areas, and a margin above and below; elsewhere,
// from where the card starts down to a margin above the window's bottom, so
// that the card never runs past it.
export const roomInWindow = (body: HTMLElement, card: HTMLElement) => {
  const held = [...roomClasses].filter((name) => body.classList.contains(name));
  body.classList.remove(...held);
  const own = body.getBoundingClientRect();
  const frame = card.getBoundingClientRect();
  body.classList.add(...held);
  // the card's head and border above its body; its border below, and its
  // margin
  const above = own.top - frame.top;
  const below =
    frame.bottom -
    own.bottom +
    (parseFloat(getComputedStyle(card).marginBottom) || 0);
  const phone = phoneLayout();
  return {
    own: own.height,
    phone,
    room: phone
      ? window.innerHeight -
        pinnedHeight() -
        above -
        below -
        safeAreas() -
        2 * MARGIN
      : window.innerHeight - (own.top + window.scrollY) - below - MARGIN,
  };
};

// Calls fit again whenever the window or the page changes size, or one of the
// page's affixes pins or lets go; returns what stops it.
export const refitOnPageChanges = (fit: () => void) => {
  window.addEventListener("resize", fit);
  // what lies above the card can change size after it is first measured
  const page =
    typeof ResizeObserver === "undefined" ? null : new ResizeObserver(fit);
  page?.observe(document.body);
  // an affix pins as its class changes, and the page shows the rack anew as
  // the game is stepped through
  const affix = new MutationObserver((records) => {
    if (records.some((r) => (r.target as Element).closest(AFFIXES))) {
      fit();
    }
  });
  affix.observe(document.body, { attributeFilter: ["class"], subtree: true });
  return () => {
    window.removeEventListener("resize", fit);
    page?.disconnect();
    affix.disconnect();
  };
};
