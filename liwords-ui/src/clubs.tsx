import { Col, Divider, Row } from 'antd';
import React from 'react';
import { TopBar } from './topbar/topbar';

export const Clubs = (props: {}) => {
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <Row>
        <Col span={24} offset={6}>
          <p>All events are shown in your local timezone: {tz}</p>
        </Col>
      </Row>
      <Row>
        <Col span={24} offset={1}>
          <h3>Club Directory</h3>
        </Col>
      </Row>
      <Row>
        <Col span={24}>
          <iframe
            src={`https://calendar.google.com/calendar/embed?src=c_qv1pd8072ui5aal0u0jsnh2n34%40group.calendar.google.com&ctz=${encodeURIComponent(
              tz
            )}`}
            style={{ border: 0 }}
            width="800"
            height="600"
            frameBorder="0"
            scrolling="no"
            title="Club Calendar"
          />
        </Col>
      </Row>
      <Divider />
      <Row>
        <Col span={24} offset={1}>
          <h3>Upcoming Tournaments</h3>
        </Col>
      </Row>
      <Row>
        <Col span={24}>
          <iframe
            src={`https://calendar.google.com/calendar/embed?src=c_q7s63i04spmcd7o1qep0vo0920%40group.calendar.google.com&ctz=${encodeURIComponent(
              tz
            )}`}
            style={{ border: 0 }}
            width="800"
            height="600"
            frameBorder="0"
            scrolling="no"
            title="Upcoming Tournaments"
          />
        </Col>
      </Row>
    </>
  );
};
