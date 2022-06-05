import React from 'react';
import { Faller } from './faller';

type BlockProps = {
  isEmpty?: boolean;
  rune?: string;
  level?: number;
  left?: boolean;
  right?: boolean;
  top?: boolean;
  bottom?: boolean;
};
export const Block = React.memo((props: BlockProps) => {
  const { isEmpty, rune, level = 0, left, right, top, bottom } = props;
  const computedClassName = `block level-${level} ${isEmpty ? 'empty' : ''}
    ${left ? ' left' : ''}${right ? ' right' : ''}${top ? ' top' : ''}
    ${bottom ? ' bottom' : ''}`;
  return (
    <div className={computedClassName}>
      {!isEmpty && <div className="rune">{rune}</div>}
    </div>
  );
});

export const Well = React.memo(() => {
  return (
    <>
      <div className="well-wrapper">
        <div className="well-backdrop">
          <div className="well-row">
            <Block isEmpty />
            <Block rune="M" left bottom right level={0} />
          </div>
        </div>
        <Faller />
        <div className="well-overlay"></div>
      </div>
    </>
  );
});
