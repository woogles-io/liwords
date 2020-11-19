import React from 'react';
import { Card } from 'antd';

type AnnouncementsProps = {};

export const Announcements = React.memo((props: AnnouncementsProps) => {
  // Todo: admin to add these and a backend to store and retrieve them
  const announcements = [
    {
      title: 'Upcoming Tournament - Crescent City Cup',
      body: (
        <a
          href="https://sites.google.com/site/nolascrabble"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The 10th Annual Crescent City Cup will be held on Woogles on January
            17th and 18th. Twenty games with Open and Lite divisions for both
            CSW19 and NWL2020. Register now.
          </p>
        </a>
      ),
    },
    {
      title: 'Live Tournament - The WETO',
      body: (
        <a
          href="https://woogles.io/tournament/weto"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            A single elimination best 3 out of 5 CSW tournament on Woogles.io
            with an automatic side tournament, November 16 - December 20, 2020.
            If you are registered, click here to attend the tournament.
          </p>
        </a>
      ),
    },
    {
      title: 'First tournament complete!',
      body: (
        <a
          href="https://woogles.io/tournament/dells2020"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The 37th Annual Wisconsin Dells Tourney was held on Woogles November
            14th and 15th. If you missed it, take a look at the games here.
            Congratulations to the winners, Lisa Odom (NWL) and Geoff Thevenot
            (CSW)!
          </p>
        </a>
      ),
    },
    {
      title: 'Want to help?',
      body: (
        <a
          href="https://woogles.io/about"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            Woogles is a nonprofit, funded completely by donations, and
            committed to being ad-free and free for everyone. Want to make a
            donation and ensure its future?
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
