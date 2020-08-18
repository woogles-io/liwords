import React from 'react';
import { Card, Row, Col } from 'antd';

type Props = {
  message: string;
};

export const GameEndMessage = (props: Props) => (
  <Card className="end-message" size="small">
    {props.message.split('\n').map((line, idx) => (
      <Row key={`l-${idx + 0}`} justify="center">
        <Col>{line}</Col>
      </Row>
    ))}
  </Card>
);
