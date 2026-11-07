import { cleanup, render } from "@testing-library/react";
import { useMemo, useRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useGrowToWindowBottom } from "./grow_to_window_bottom";

// the window as tall as given, and the card's body at the top given, 190 px
// tall, with 1 px of the card's border below it
const layOut = (innerHeight: number, top: number) => {
  vi.stubGlobal("innerHeight", innerHeight);
  return vi
    .spyOn(HTMLElement.prototype, "getBoundingClientRect")
    .mockImplementation(function (this: HTMLElement) {
      const bottom = this.classList.contains("ant-card-body")
        ? top + 190
        : top + 191;
      return { top, bottom, height: 190 } as DOMRect;
    });
};

const Card = (props: { on: boolean }) => {
  const card = useRef<HTMLDivElement>(null);
  useGrowToWindowBottom(card, props.on);
  return (
    <div ref={card}>
      <div className="ant-card-body" />
    </div>
  );
};

const body = () => document.querySelector<HTMLElement>(".ant-card-body")!;

describe("useGrowToWindowBottom", () => {
  let rects: { mockRestore: () => void } | undefined;
  afterEach(() => {
    rects?.mockRestore();
    cleanup();
    vi.unstubAllGlobals();
  });

  it("gives the card's body its own height and its room down to a margin above the window's bottom", () => {
    rects = layOut(760, 499);
    render(<Card on />);
    expect(body()).toHaveClass("grow-to-bottom");
    expect(body().style.getPropertyValue("--grow-own")).toBe("190px");
    expect(body().style.getPropertyValue("--grow-room")).toBe("236px");
  });

  it("does nothing while off, and takes it all back once off", () => {
    rects = layOut(760, 499);
    const { rerender } = render(<Card on={false} />);
    expect(body()).not.toHaveClass("grow-to-bottom");
    rerender(<Card on />);
    expect(body().style.getPropertyValue("--grow-room")).toBe("236px");
    rerender(<Card on={false} />);
    expect(body()).not.toHaveClass("grow-to-bottom");
    expect(body().style.getPropertyValue("--grow-own")).toBe("");
    expect(body().style.getPropertyValue("--grow-room")).toBe("");
  });

  it("measures again as the window is resized", () => {
    rects = layOut(760, 499);
    render(<Card on />);
    vi.stubGlobal("innerHeight", 1000);
    window.dispatchEvent(new Event("resize"));
    expect(body().style.getPropertyValue("--grow-room")).toBe("476px");
  });

  it("tells the page with a resize as it turns on and off", () => {
    rects = layOut(760, 499);
    const heard = vi.fn();
    window.addEventListener("resize", heard);
    const { rerender } = render(<Card on={false} />);
    expect(heard).not.toHaveBeenCalled();
    rerender(<Card on />);
    expect(heard).toHaveBeenCalledTimes(1);
    rerender(<Card on={false} />);
    expect(heard).toHaveBeenCalledTimes(2);
    window.removeEventListener("resize", heard);
  });

  it("has the window below what the page pins, on a page laid out as on a phone", () => {
    vi.stubGlobal("innerHeight", 844);
    rects = vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.classList.contains("sticky-player-card-container")) {
          return { top: 0, bottom: 166, height: 166 } as DOMRect;
        }
        const bottom = this.classList.contains("ant-card-body") ? 1957 : 1958;
        return { top: 1767, bottom, height: 190 } as DOMRect;
      });
    const shown = vi
      .spyOn(HTMLElement.prototype, "getClientRects")
      .mockImplementation(() => [{}] as unknown as DOMRectList);
    render(
      <div>
        <div className="sticky-player-card-container" />
        <Card on />
      </div>,
    );
    expect(body().style.getPropertyValue("--grow-room")).toBe("629px");
    shown.mockRestore();
  });

  it("has its room past half the window, on a page laid out for a wider window", () => {
    rects = layOut(1200, 499);
    render(<Card on />);
    expect(body().style.getPropertyValue("--grow-room")).toBe("676px");
  });

  it("has its own height as its room at least", () => {
    rects = layOut(200, 499);
    render(<Card on />);
    expect(body().style.getPropertyValue("--grow-room")).toBe("190px");
  });
});

describe("useGrowToWindowBottom with a card above that gives up height", () => {
  // the card above 355 px tall at its full height; the growing card's body
  // as in layOut, its room 236 px in a window 760 px tall
  const layOutBelow = () => {
    vi.stubGlobal("innerHeight", 760);
    return vi
      .spyOn(HTMLElement.prototype, "getBoundingClientRect")
      .mockImplementation(function (this: HTMLElement) {
        if (this.id === "above") {
          return { top: 0, bottom: 355, height: 355 } as DOMRect;
        }
        const bottom = this.classList.contains("ant-card-body") ? 689 : 690;
        return { top: 499, bottom, height: 190 } as DOMRect;
      });
  };

  // a list that holds hidden px more than it shows
  const List = (props: { hidden: number }) => (
    <div
      className="list"
      ref={(el) => {
        if (el) {
          Object.defineProperty(el, "scrollHeight", {
            configurable: true,
            value: 150 + props.hidden,
          });
          Object.defineProperty(el, "clientHeight", {
            configurable: true,
            value: 150,
          });
        }
      }}
    />
  );

  const Below = (props: { spare: number; hidden: number; on?: boolean }) => {
    const card = useRef<HTMLDivElement>(null);
    const giver = useMemo(
      () => ({
        card: () => document.getElementById("above"),
        spare: () => props.spare,
        list: ".list",
      }),
      [props.spare],
    );
    useGrowToWindowBottom(card, props.on ?? true, giver);
    return (
      <div>
        <div id="above" />
        <div ref={card}>
          <div className="ant-card-body">
            <List hidden={props.hidden} />
          </div>
        </div>
      </div>
    );
  };

  const above = () => document.getElementById("above")!;

  let rects: { mockRestore: () => void } | undefined;
  afterEach(() => {
    rects?.mockRestore();
    cleanup();
    vi.unstubAllGlobals();
  });

  it("takes what the list needs past its room from the card above", () => {
    rects = layOutBelow();
    // the body needs 190 + 250 = 440 px, 204 px past its room of 236
    render(<Below spare={300} hidden={250} />);
    expect(above()).toHaveClass("gives-room");
    expect(above().style.getPropertyValue("--give-height")).toBe("151px");
    expect(body().style.getPropertyValue("--grow-room")).toBe("440px");
  });

  it("takes no more than the card above can spare", () => {
    rects = layOutBelow();
    render(<Below spare={100} hidden={250} />);
    expect(above().style.getPropertyValue("--give-height")).toBe("255px");
    expect(body().style.getPropertyValue("--grow-room")).toBe("336px");
  });

  it("takes nothing while the list fits its room", () => {
    rects = layOutBelow();
    render(<Below spare={300} hidden={40} />);
    expect(above()).not.toHaveClass("gives-room");
    expect(above().style.getPropertyValue("--give-height")).toBe("");
    expect(body().style.getPropertyValue("--grow-room")).toBe("236px");
  });

  it("keeps what it took as the list gets shorter", () => {
    rects = layOutBelow();
    const { rerender } = render(<Below spare={300} hidden={250} />);
    rerender(<Below spare={300} hidden={0} />);
    window.dispatchEvent(new Event("resize"));
    expect(above().style.getPropertyValue("--give-height")).toBe("151px");
  });

  it("gives it back once the card above can spare nothing", () => {
    rects = layOutBelow();
    const { rerender } = render(<Below spare={300} hidden={250} />);
    rerender(<Below spare={0} hidden={250} />);
    expect(above()).not.toHaveClass("gives-room");
    expect(body().style.getPropertyValue("--grow-room")).toBe("236px");
  });

  it("measures again as the card above shows another of its tabs", async () => {
    rects = layOutBelow();
    let spare = 0;
    const Tabbed = () => {
      const card = useRef<HTMLDivElement>(null);
      const giver = useMemo(
        () => ({
          card: () => document.getElementById("above"),
          spare: () => spare,
          list: ".list",
        }),
        [],
      );
      useGrowToWindowBottom(card, true, giver);
      return (
        <div>
          <div id="above">
            <div className="pane" />
          </div>
          <div ref={card}>
            <div className="ant-card-body">
              <List hidden={250} />
            </div>
          </div>
        </div>
      );
    };
    render(<Tabbed />);
    expect(above()).not.toHaveClass("gives-room");
    spare = 300;
    document.querySelector(".pane")!.classList.add("active");
    await Promise.resolve();
    expect(above().style.getPropertyValue("--give-height")).toBe("151px");
  });

  it("gives it all back once off", () => {
    rects = layOutBelow();
    const { rerender } = render(<Below spare={300} hidden={250} />);
    rerender(<Below spare={300} hidden={250} on={false} />);
    expect(above()).not.toHaveClass("gives-room");
    expect(above().style.getPropertyValue("--give-height")).toBe("");
  });

  it("takes nothing, and keeps its own height, on a page laid out as on a phone", () => {
    rects = layOutBelow();
    const shown = vi
      .spyOn(HTMLElement.prototype, "getClientRects")
      .mockImplementation(() => [{}] as unknown as DOMRectList);
    render(
      <div>
        <div className="sticky-player-card-container" />
        <Below spare={300} hidden={250} />
      </div>,
    );
    expect(above()).not.toHaveClass("gives-room");
    expect(body()).not.toHaveClass("grow-to-bottom");
    expect(body().style.getPropertyValue("--grow-room")).toBe("");
    shown.mockRestore();
  });
});
