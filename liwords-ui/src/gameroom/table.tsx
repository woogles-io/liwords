import React from 'react';
import { Row, Col } from 'antd';

import { useParams } from 'react-router-dom';
import { BoardPanel } from './board_panel';
import { TopBar } from '../topbar/topbar';
import { Chat } from './chat';

const gutter = 16;
const boardspan = 12;
const maxspan = 24; // from ant design
const navbarHeightAndGutter = 84; // 72 + 12 spacing

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

type RouterProps = {
  gameID: string;
};

type Props = {
  windowWidth: number;
  windowHeight: number;
};

export const Table = (props: Props) => {
  // Calculate the width of the board.
  // If the pixel width is 1440,
  // The width of the drawable part is 12/24 * 1440 = 720
  // Minus gutters makes it 704

  // The height is more important, as the buttons and tiles go down
  // so don't make the board so tall that these elements become invisible.

  let boardPanelWidth = (boardspan / maxspan) * props.windowWidth - gutter;
  // Shrug; determine this better:
  let boardPanelHeight = boardPanelWidth + 96;
  const viewableHeight = props.windowHeight - navbarHeightAndGutter;

  // XXX: this all needs to be tweaked.
  if (boardPanelHeight > viewableHeight) {
    boardPanelHeight = viewableHeight;
    boardPanelWidth = boardPanelHeight - 96;
  }

  const { gameID } = useParams();

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <Row gutter={gutter} style={{ marginTop: 12 }}>
        <Col span={6}>
          <Chat gameID={gameID} />
        </Col>
        <Col span={boardspan}>
          <BoardPanel
            compWidth={boardPanelWidth}
            compHeight={boardPanelHeight}
            gridLayout={gridLayout}
            showBonusLabels={false}
            currentRack="ABEOPXZ"
            lastPlayedLetters={{}}
            tilesLayout={oxyTilesLayout}
            gameID={gameID}
          />
        </Col>
        <Col span={6}>
          <div>scorecard and stuff</div>
        </Col>
      </Row>
    </div>
  );
};
