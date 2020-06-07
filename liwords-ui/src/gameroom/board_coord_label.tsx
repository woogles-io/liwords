import React from 'react';

type Props = {
  label: string;
};

const CoordLabel = (props: Props) => {
  return (
      <p
        className="coord-label"
      >
        {props.label}
      </p>
  );
};

export default CoordLabel;
