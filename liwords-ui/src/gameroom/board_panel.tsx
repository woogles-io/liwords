import React from 'react';

import { Board } from './board';

type Props = {
  compWidth: number;
  compHeight: number;
  gridLayout: Array<string>;
  showBonusLabels: boolean;
};

export const BoardPanel = (props: Props) => {
  return (
    <div
      style={{
        width: props.compWidth,
        height: props.compHeight,
        background: 'linear-gradient(180deg, #E2F8FF 0%, #FFFFFF 100%)',
        boxShadow: '0px 0px 30px rgba(0, 0, 0, 0.1)',
        borderRadius: '4px',
      }}
    >
      <Board
        compWidth={props.compWidth}
        gridLayout={props.gridLayout}
        showBonusLabels={false}
      />
    </div>
  );
};
