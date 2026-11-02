import React, { useLayoutEffect, useState } from "react";
import { hookLine, useLabHooks } from "./lab_hooks";

type Props = {
  gridDim: number;
};

// Whether a line of letters fits a square, measured in the font the squares
// draw with and the width they have on the page, again whenever the board
// changes size; undefined where the page cannot measure, as in tests.
const useLineFits = (grid: HTMLDivElement | null) => {
  const [fits, setFits] = useState<(line: string) => boolean>();
  useLayoutEffect(() => {
    if (!grid || typeof ResizeObserver === "undefined") return;
    const context = document.createElement("canvas").getContext("2d");
    if (!context) return;
    let current = true;
    const measure = () => {
      const cell = grid.firstElementChild;
      if (!current || !(cell instanceof HTMLElement)) return;
      const style = getComputedStyle(cell);
      context.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
      const room =
        cell.clientWidth -
        parseFloat(style.paddingLeft) -
        parseFloat(style.paddingRight);
      setFits(() => (line: string) => context.measureText(line).width <= room);
    };
    measure();
    // the board's font may load after the first measure
    document.fonts?.ready.then(measure);
    const observer = new ResizeObserver(measure);
    observer.observe(grid);
    return () => {
      current = false;
      observer.disconnect();
    };
  }, [grid]);
  return fits;
};

// The letters each empty square beside a tile takes, over the board, while the
// analyzer's lab shows hooks; a way no letter fits is marked "-". Those making
// the word across sit on top and those making the word down below, as many
// whole tiles as fit with how many more.
export const LabHooksOverlay = React.memo((props: Props) => {
  const hooks = useLabHooks();
  const [grid, setGrid] = useState<HTMLDivElement | null>(null);
  const fits = useLineFits(grid);
  if (!hooks || hooks.dim !== props.gridDim) return null;
  // Every square, by its index: the squares come as a sparse array, and a map
  // over its holes would skip them, drawing each square's letters on another.
  const squares = Array.from(
    { length: hooks.dim * hooks.dim },
    (_, i) => hooks.squares[i],
  );
  return (
    <div className="lab-hooks" ref={setGrid}>
      {squares.map((square, i) => (
        <div key={i} className="lab-hook">
          {square?.across && (
            <span className="lab-hook-across">
              {square.across.length ? hookLine(square.across, fits) : "-"}
            </span>
          )}
          {square?.down && (
            <span className="lab-hook-down">
              {square.down.length ? hookLine(square.down, fits) : "-"}
            </span>
          )}
        </div>
      ))}
    </div>
  );
});
