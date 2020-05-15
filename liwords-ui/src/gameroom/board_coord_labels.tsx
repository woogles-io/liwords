import React from 'react';

import CoordLabel from './board_coord_label';

type Props = {
  gridDim: number;
  boardSquareDim: number;
  colLabelHeight: number;
  colLabelGutter: number;
  rowLabelWidth: number;
  rowLabelGutter: number;
};

const BoardCoordLabels = (props: Props) => {
  const labels = [];
  const horizPadding = props.rowLabelWidth + 2 * props.rowLabelGutter;
  const vertPadding = props.colLabelHeight + 2 * props.colLabelGutter;

  // COLUMN labels.
  for (let i = 0; i < props.gridDim; i += 1) {
    labels.push(
      <CoordLabel
        rectHeight={vertPadding}
        rectWidth={props.boardSquareDim}
        x={i * props.boardSquareDim + horizPadding}
        y={0}
        label={String.fromCharCode(i + 'A'.charCodeAt(0))}
        key={`collbl${i}`}
      />
    );
  }

  // ROW labels
  for (let i = 0; i < props.gridDim; i += 1) {
    labels.push(
      <CoordLabel
        rectHeight={props.boardSquareDim}
        rectWidth={horizPadding}
        x={0}
        y={i * props.boardSquareDim + vertPadding}
        label={String(i + 1)}
        key={`rowlbl${i}`}
      />
    );
  }
  return <>{labels}</>;
};

export default BoardCoordLabels;
