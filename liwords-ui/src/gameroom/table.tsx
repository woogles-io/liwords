import React from 'react';
import { Row, Col } from 'antd';

import { Board } from './board';

const gutter = 16;
const boardspan = 12;
const maxspan = 24; // from ant design

type Props = {
  windowWidth: number;
};

export const Table = (props: Props) => {
  // Calculate the width of the board.
  // If the pixel width is 1440,
  // The width of the drawable part is 12/24 * 1440 = 720
  // Minus gutters makes it 704

  <div>
    <Row gutter={gutter}>
      <Col span={6}>
        <div>lefts</div>
      </Col>
      <Col span={boardspan}>
        <Board compWidth={(boardspan / maxspan) * props.windowWidth - gutter} />
      </Col>
      <Col span={6}>
        <div>scorecard and stuff</div>
      </Col>
    </Row>
  </div>;
};
