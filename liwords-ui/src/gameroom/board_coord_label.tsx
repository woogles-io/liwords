import React from 'react';

type Props = {
  x: number;
  y: number;
  label: string;
  rectWidth: number;
  rectHeight: number;
};

const CoordLabel = (props: Props) => {
  const transform = `translate(${props.x},${props.y})`;
  return (
    <g transform={transform}>
      <text
        x={props.rectWidth / 2}
        y={props.rectHeight / 2}
        textAnchor="middle"
        dominantBaseline="central"
        fontFamily="'Source Code Pro',monospace"
        fontSize="100%"
        fontWeight="600"
        stroke="#000000"
        fill="#000000"
        strokeWidth="0.35px"
      >
        {props.label}
      </text>
    </g>
  );
};

export default CoordLabel;
