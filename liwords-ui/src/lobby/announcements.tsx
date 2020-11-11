import React from 'react';
import { Card } from 'antd';

type AnnouncementsProps = {};

export const Announcements = React.memo((props: AnnouncementsProps) => {
  // Todo: admin to add these and a backend to store and retrieve them
  const announcements = [
    {
      title: 'Upcoming Tournament - The WETO',
      body: (
        <a
          href="https://docs.google.com/document/d/1jqtNAnAbXChW86QyGUggJH5xn6FxhE8BJYWmo9mHXMw"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            A single elimination best 3 out of 5 CSW tournament on Woogles.io
            with an automatic side tournament, November 16 - December 20, 2020.
            $15.00
          </p>
        </a>
      ),
    },
    {
      title: 'Upcoming Tournament - 37th Annual Wisconsin Dells Tourney',
      body: (
        <a
          href="https://docs.google.com/document/d/1d38HOaiQANqK5WAVzKKLd_v_ZR9Jw45OnOek1MVU0io"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            13 games on Woogles.io, NWL and CSW divisions, November 14 and 15,
            2020. Free!
          </p>
        </a>
      ),
    },
    {
      title: 'Find a bug? Let us know',
      body: (
        <a
          href="https://tinyurl.com/y4dkb2g6"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            We've made it easier to submit your bugs and feedback. Let us know
            if you find a problem or have a suggestion.
          </p>
        </a>
      ),
    },
    {
      title: 'Woogles is live! Come join our Discord',
      body: (
        <a
          href="https://discord.gg/5yCJjmW"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            Welcome to our open beta. We still have a lot of features and
            designs to build. Please join our Discord server to discuss your
            thoughts. Happy Woogling!
          </p>
        </a>
      ),
    },
  ];
  const renderAnnouncements = announcements.map((a, idx) => (
    <li key={idx}>
      <h4>{a.title}</h4>
      {a.body}
    </li>
  ));
  return (
    <div className="announcements">
      <Card title="Announcements">
        <ul className="announcements-list">{renderAnnouncements}</ul>
      </Card>
    </div>
  );
});
