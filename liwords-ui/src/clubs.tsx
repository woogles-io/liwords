import { Col, Divider, Row } from 'antd';
import React from 'react';
import { TopBar } from './topbar/topbar';
import './clubs.scss';
export const Clubs = (props: {}) => {
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="event-calendar">
        <p>All events are shown in your local timezone: {tz}</p>
        <h3>Club Directory</h3>
        <iframe
          src={`https://calendar.google.com/calendar/embed?src=c_qv1pd8072ui5aal0u0jsnh2n34%40group.calendar.google.com&showTitle=0&mode=AGENDA&ctz=${encodeURIComponent(
            tz
          )}`}
          style={{ border: 0 }}
          height="600"
          width="800"
          frameBorder="0"
          scrolling="yes"
          title="club calendar"
        />

        <Divider />

        <h3>Upcoming Tournaments</h3>

        <iframe
          src={`https://calendar.google.com/calendar/embed?src=c_q7s63i04spmcd7o1qep0vo0920%40group.calendar.google.com&showTitle=0&mode=AGENDA&ctz=${encodeURIComponent(
            tz
          )}`}
          style={{ border: 0 }}
          width="800"
          height="600"
          frameBorder="0"
          scrolling="yes"
          title="tournament calendar"
        />
      </div>
    </>
  );
};
