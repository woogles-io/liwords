import { Col, Row } from 'antd';
import React from 'react';
import { TopBar } from './navigation/topbar';

export const DonateSuccess = () => {
  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="donations donation-success">
        <h4>Thank you so much for your donation. &lt;3</h4>
        <p>
          <a href="https://woogles.io">Return to Woogles</a>
        </p>
      </div>
    </>
  );
};
