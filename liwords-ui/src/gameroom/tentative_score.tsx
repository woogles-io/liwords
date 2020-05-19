import React from 'react';

const fontFamily = "'Roboto', sans-serif";

type Props = {
  score: number;
  width: number;
  height: number;
  x: number;
  y: number;
};

const TentativeScore = (props: Props) => {
  const textSize = (90 / 31) * props.width;
  const transform = `translate(${props.x}, ${props.y})`;
  return (
    <g transform={transform}>
      <ellipse
        cx={props.width / 2}
        cy={props.height / 2}
        rx={props.width / 2}
        ry={props.height / 2}
        style={{ fill: 'yellow', stroke: 'purple', strokeWidth: '1px' }}
      />
      <text
        x={props.width / 2}
        y={props.height / 2}
        textAnchor="middle"
        dominantBaseline="central"
        fontFamily={fontFamily}
        fontWeight={500}
        fontSize={`${textSize}%`}
        fill="#000000"
        strokeWidth={0}
      >
        {props.score}
      </text>
    </g>
  );
};

export default TentativeScore;
