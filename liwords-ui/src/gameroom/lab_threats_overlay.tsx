import React from "react";
import { useLabHooks } from "./lab_hooks";
import { useLabThreats } from "./lab_threats";

type Props = {
  gridDim: number;
};

// What the opponent may play next, over the board, while the analyzer's lab
// shows threats: each empty square tinted by how hot it is, with its number;
// the play picked in the list as the opponent's tiles, apart from any the
// player has placed; and the squares of the play pointed at, outlined. While
// the panel picks a square, a click on one narrows the list to it, and a
// click on it again widens the list back. While the hooks show too, their
// letters take the square's room, and the tint alone says how hot it is.
export const LabThreatsOverlay = React.memo((props: Props) => {
  const threats = useLabThreats();
  const hooks = useLabHooks();
  if (!threats || threats.dim !== props.gridDim) return null;
  const squares = Array.from(
    { length: threats.dim * threats.dim },
    (_, i) => i,
  );
  return (
    <div className={`lab-threats${threats.picking ? " lab-picking" : ""}`}>
      {squares.map((i) => {
        const heat = threats.heat[i];
        const ghost = threats.ghost.get(i);
        const classes = [
          "lab-threat",
          threats.outline.has(i) ? "lab-outlined" : "",
          threats.square === i ? "lab-narrowed" : "",
          ghost !== undefined ? "lab-ghost" : "",
        ]
          .filter(Boolean)
          .join(" ");
        return (
          <div
            key={i}
            className={classes}
            onClick={
              threats.picking
                ? () => threats.pick(threats.square === i ? null : i)
                : undefined
            }
          >
            {heat && ghost === undefined && (
              <React.Fragment>
                {/* the least hot square still shows, and the hottest leaves
                    the square's own color showing through */}
                <span
                  className="lab-heat"
                  style={{ opacity: 0.15 + 0.6 * heat.level }}
                />
                {!hooks && <span className="lab-heat-label">{heat.label}</span>}
              </React.Fragment>
            )}
            {ghost}
          </div>
        );
      })}
    </div>
  );
});
