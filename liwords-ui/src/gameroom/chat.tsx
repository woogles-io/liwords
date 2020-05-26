import React from 'react';
import { Card } from 'antd';

type Props = {
  gameID: string;
};

export const Chat = (props: Props) => {
  return (
    <Card style={{ textAlign: 'left' }}>
      <div>GAME ID: {props.gameID}</div>
      <div>césar: yay</div>
      <div>conrad: codar</div>
      <div>conrad played 8G FOO for 23 pts</div>
      <div> more chat here...</div>
    </Card>
  );
};
