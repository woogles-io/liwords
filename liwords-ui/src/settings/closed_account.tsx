import React from 'react';
import woogles from '../assets/woogles.png';
import { Row, Col } from 'antd';

type Props = {};

export const ClosedAccount = React.memo((props: Props) => {
  return (
    <div className="closed-account">
      <h3>Your account will be closed</h3>
      <div>
        The Woogles team has been notified of your request to close your
        account.
      </div>
      <Row>
        <Col>
          <img src={woogles} className="woogles" alt="Woogles" />
        </Col>
        <Col className="thanks">Thanks for using Woogles.io!</Col>
      </Row>
    </div>
  );
});
