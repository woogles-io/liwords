import { cleanup, render } from "@testing-library/react";
import { useRef } from "react";
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
