import React from 'react';
import { Row, Col } from 'antd';

import { BoardPanel } from './board_panel';

const gutter = 16;
const boardspan = 12;
const maxspan = 24; // from ant design

type Props = {
  windowWidth: number;
};

// XXX: This should come from the backend.
const gridLayout = [
  "=  '   =   '  =",
  ' -   "   "   - ',
  "  -   ' '   -  ",
  "'  -   '   -  '",
  '    -     -    ',
  ' "   "   "   " ',
  "  '   ' '   '  ",
  "=  '   -   '  =",
  "  '   ' '   '  ",
  ' "   "   "   " ',
  '    -     -    ',
  "'  -   '   -  '",
  "  -   ' '   -  ",
  ' -   "   "   - ',
  "=  '   =   '  =",
];

const tilesLayout = [
  '         RADIO ',
  '         E     ',
  '      R SI     ',
  '      U E      ',
  '    ZINGARO    ',
  '    o   T      ',
  '    N          ',
  '   WASTE       ',
  '    T          ',
  '    I          ',
  '    O          ',
  '    N          ',
  '               ',
  '               ',
  '               ',
];

const oxyTilesLayout = [
  ' PACIFYING     ',
  ' IS            ',
  'YE             ',
  ' REQUALIFIED   ',
  'H L            ',
  'EDS            ',
  'NO   T         ',
  ' RAINWASHING   ',
  'UM   O         ',
  'T  E O         ',
  ' WAKEnERS      ',
  ' OnETIME       ',
  'OOT  E B       ',
  'N      U       ',
  ' JACULATING    ',
];

export const Table = (props: Props) => {
  // Calculate the width of the board.
  // If the pixel width is 1440,
  // The width of the drawable part is 12/24 * 1440 = 720
  // Minus gutters makes it 704
  const boardPanelWidth = (boardspan / maxspan) * props.windowWidth - gutter;
  // Shrug; determine this better:
  const boardPanelHeight = boardPanelWidth + 96;

  return (
    <div>
      <Row gutter={gutter}>
        <Col span={6}>
          <div>lefts</div>
        </Col>
        <Col span={boardspan}>
          <BoardPanel
            compWidth={boardPanelWidth}
            compHeight={boardPanelHeight}
            gridLayout={gridLayout}
            showBonusLabels={false}
            currentRack="AEINQ?T"
            lastPlayedLetters={{}}
            tilesLayout={tilesLayout}
          />
        </Col>
        <Col span={6}>
          <div>scorecard and stuff</div>
        </Col>
      </Row>
    </div>
  );
};
