import React from 'react';

import CoordLabel from './board_coord_label';

type Props = {
  gridDim: number;
};

const BoardCoordLabels = React.memo((props: Props) => {
  const horizLabels = [];
  const vertLabels = [];

  // COLUMN labels.
  for (let i = 0; i < props.gridDim; i += 1) {
    horizLabels.push(
      <CoordLabel
        label={String.fromCharCode(i + 'A'.charCodeAt(0))}
        key={`collbl${i}`}
      />
    );
  }

  // ROW labels
  for (let i = 0; i < props.gridDim; i += 1) {
    vertLabels.push(<CoordLabel label={String(i + 1)} key={`rowlbl${i}`} />);
  }
  return (
    <>
      <div className="coord-labels horiz">{horizLabels}</div>
      <div className="coord-labels vert">{vertLabels}</div>
    </>
  );
});

export default BoardCoordLabels;
