import React from 'react';
import { Button } from 'antd';

type Props = {
  onRecall: () => void;
};

const GameControls = (props: Props) => (
  <>
    <Button style={{ width: 100, height: 40 }}>Exchange</Button>
    <Button danger style={{ width: 100, height: 40, marginLeft: 25 }}>
      Pass
    </Button>
    <Button
      onClick={props.onRecall}
      style={{ width: 100, height: 40, marginLeft: 25 }}
    >
      Recall tiles
    </Button>
    <Button style={{ width: 100, height: 40, marginLeft: 25 }}>
      Challenge
    </Button>
    <Button type="primary" style={{ width: 100, height: 40, marginLeft: 25 }}>
      Commit
    </Button>
  </>
);

export default GameControls;
