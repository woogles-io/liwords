import React from 'react';
import { Card } from 'antd';

type Props = {
  message: string;
};

export const GameEndMessage = (props: Props) => (
  <Card className="end-message" size="small">
    {props.message.split('\n').map((line, idx) => (
      <p key={`l-${idx + 0}`}>{line}</p>
    ))}
  </Card>
);
