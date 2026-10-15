import React from "react";

import CoordLabel from "./board_coord_label";

type Props = {
  gridDim: number;
  // The squares are upside down: label them from the far end.
  flipped?: boolean;
};

const BoardCoordLabels = React.memo((props: Props) => {
  const horizLabels = [];
  const vertLabels = [];

  // COLUMN labels.
  for (let i = 0; i < props.gridDim; i += 1) {
    const col = props.flipped ? props.gridDim - 1 - i : i;
    horizLabels.push(
      <CoordLabel
        label={String.fromCharCode(col + "A".charCodeAt(0))}
        key={`collbl${i}`}
      />,
    );
  }

  // ROW labels
  for (let i = 0; i < props.gridDim; i += 1) {
    const row = props.flipped ? props.gridDim - 1 - i : i;
    vertLabels.push(<CoordLabel label={String(row + 1)} key={`rowlbl${i}`} />);
  }
  return (
    <>
      <div className="coord-labels horiz">{horizLabels}</div>
      <div className="coord-labels vert">{vertLabels}</div>
    </>
  );
});

export default BoardCoordLabels;
