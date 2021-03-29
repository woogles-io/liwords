import React from 'react';
import { Card } from 'antd';

type AnnouncementsProps = {};

export const Announcements = React.memo((props: AnnouncementsProps) => {
  // Todo: admin to add these and a backend to store and retrieve them
  const announcements = [
    {
      title: 'Where is my club or tournament?',
      link: 'https://woogles.io/clubs',
      body: (
        <p>
          Wondering where you can find a good club or tournament to play in? See
          more info here!
        </p>
      ),
    },

    {
      title: 'Upcoming Tournament - Virtual CanAm 2021',
      link: 'https://woogles.io/tournament/vcanam',
      body: (
        <p>
          The CanAm has been brought to Woogles virtually, with teams of
          Canadians and Americans vying for the virtual trophy! This is a
          two-day tournament on March 27 and 28, with CSW and NWL divisions. 2
          teams of 5 players for each division will play a triple round-robin
          against each other. Who will emerge victorious?
        </p>
      ),
    },

    {
      title: 'Ongoing Tournament - CoCo Blitz Championship',
      link: 'https://woogles.io/tournament/coco-blitz',
      body: (
        <p>
          The World Blitz Championship kicks off in late January and runs
          through late April. The fastest word gamers in the world will play
          3-minute games, starting with round robin pool play and culminating
          with a playoff bracket to determine the World Blitz Champion!
        </p>
      ),
    },

    {
      title: 'Finished Tournament - HOPPY',
      link: 'https://woogles.io/tournament/hoppy',
      body: (
        <p>
          The HOPPY, a 13-game tournament, took place on March 20 and 21st and
          was the largest tourney yet on Woogles, with 94 unique players! Click
          here to see the games and final results. Congratulations to the
          winners: Rodney Weis (NWL Classic), Jared Cappel (NWL Premier), Jack
          Moran (CSW Classic), and Ben Schoenbrun (CSW Premier)!
        </p>
      ),
    },
    {
      title: 'Want to help?',
      link: 'https://woogles.io/about',
      body: (
        <p>
          Woogles is a nonprofit, funded completely by donations, and committed
          to being ad-free and free for everyone. Want to make a donation and
          ensure its future?
        </p>
      ),
    },
    {
      title: 'Find a bug? Let us know',
      link: 'https://tinyurl.com/y4dkb2g6',
      body: (
        <p>
          We've made it easier to submit your bugs and feedback. Let us know if
          you find a problem or have a suggestion.
        </p>
      ),
    },
    {
      title: 'Woogles is live! Come join our Discord',
      link: 'https://discord.gg/GqkUqA7ENm',
      body: (
        <p>
          Welcome to our open beta. We still have a lot of features and designs
          to build. Please join our Discord server to discuss your thoughts.
          Happy Woogling!
        </p>
      ),
    },
  ];
  const renderAnnouncements = announcements.map((a, idx) => (
    <a href={a.link} target="_blank" rel="noopener noreferrer">
      <li key={idx}>
        <h4>{a.title}</h4>
        {a.body}
      </li>
    </a>
  ));
  return (
    <div className="announcements">
      <Card title="Announcements">
        <ul className="announcements-list">{renderAnnouncements}</ul>
      </Card>
    </div>
  );
});
