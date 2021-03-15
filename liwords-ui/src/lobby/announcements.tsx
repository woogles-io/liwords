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
      title: 'Upcoming Tournament - HOPPY',
      link:
        'https://drive.google.com/file/d/1vi6eUYaYeL2az7-6eZthwevsMG0fr3fK/view',
      body: (
        <p>
          The HOPPY is a two-day tournament, brought to you by the organizers of
          the successful MERRY tournament! A two-day, 13-game event, starting on
          the first day of spring (Saturday, March 20, 2021). Premier and
          Classic divisions for both NWL20 and CSW19. All pairings and standings
          will be automatically handled by the Woogles platform!
        </p>
      ),
    },

    {
      title: 'Upcoming Tournament - Virtual CanAm 2021',
      link:
        'https://docs.google.com/spreadsheets/d/1s692UrJKbqymnuVECrtgLXxvhGVjwkuv2hJlsI4gVXQ/edit?fbclid=IwAR1SklJpVZwA6xYUPimpYKWK10cziNZ-TcEuKzXmutpOqKah0kcKOUpsacY#gid=0',
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
      title: 'Completed Tournament - Amazing Race 2',
      link: 'https://woogles.io/tournament/tarst2',
      body: (
        <p>
          Michael Fagen's Amazing Race 2 tournament is now concluded. Alec
          Sjöholm defended his title with his new teammate, Michael Fagen!
          Congrats!
        </p>
      ),
    },

    {
      title: "Completed Tournament - Brosowsky Brothers' Bonanza 2021",
      link: 'https://woogles.io/tournament/BBB',
      body: (
        <p>
          The Brosowsky Brothers' Bonanza is a one-day, 8-game event, with Open
          and Lite divisions for both NWL20 and CSW19! Check out the results.
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
