import { Col, Row } from 'antd';
import React from 'react';
import { TopBar } from './topbar/topbar';

export const DonateSuccess = () => {
  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="donations">
        <p>Thank you so much for your donation. &lt;3</p>
        <p>
          Click <a href="https://woogles.io">to go back to Woogles</a>
        </p>
      </div>
    </>
  );
};
