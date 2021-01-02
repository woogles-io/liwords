import React from 'react';
import { Card } from 'antd';

type AnnouncementsProps = {};

export const Announcements = React.memo((props: AnnouncementsProps) => {
  // Todo: admin to add these and a backend to store and retrieve them
  const announcements = [
    {
      title: 'Where is my club or tournament?',
      body: (
        <a href="/clubs" target="_blank" rel="noopener noreferrer">
          <p>
            Wondering where you can find a good club or tournament to play in?
            See more info here!
          </p>
        </a>
      ),
    },
    {
      title: 'Upcoming Tournament - M.E.R.R.Y.',
      body: (
        <a
          href="https://woogles.io/tournament/merry"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The second annual MERRY tournament will be held on Woogles on
            January 2, with an early bird on January 1. The main event is 8
            games, with CSW19 and NWL18 divisions. See more info by clicking
            here!
          </p>
        </a>
      ),
    },
    {
      title: 'Upcoming Tournament - 11th Annual Duke PBMT Charity Tournament',
      body: (
        <a
          href="https://sites.google.com/site/trianglescrabble/pbmt-2021"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            8 games Saturday, January 16, NWL20 or CSW19, starting at 10 AM. All
            proceeds from this online tourney will go to support families in the
            Pediatric Blood and Marrow Transplantation Program at the Duke
            Comprehensive Cancer Center. Last year's event raised over $10,000
            for this worthy cause!
          </p>
        </a>
      ),
    },

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
