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
      title: 'Upcoming Tournament - WYSC',
      body: (
        <a
          href="http://youthscrabble.org/WYC2020/"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The World Youth Championship runs for three consecutive weekends,
            starting January 23, 2021! Watch as teams of talented youth around
            the world play our favorite game. Who will emerge victorious?
          </p>
        </a>
      ),
    },

    {
      title: 'Upcoming Tournament - Amazing Race 2',
      body: (
        <a
          href="http://youthscrabble.org/WYC2020/"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            Michael Fagen's Amazing Race 2 tournament will be partially played
            on Woogles. You can play your games in this room.
          </p>
        </a>
      ),
    },

    {
      title: 'Upcoming Tournament - CoCo Blitz Championships',
      body: (
        <a
          href="https://www.cocoscrabble.org/blitz-champs"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The World Blitz Championship starts on the week of January 25, 2021!
            Play 3-minute games against some of the fastest word gamers in the
            world; starting with round play and proceeding to elimination
            brackets. Three-minute clocks will be used throughout!
          </p>
        </a>
      ),
    },

    {
      title: 'Upcoming Tournament - MISCO',
      body: (
        <a
          href="https://mindgamesincorporated.com/events/mgi-international-scrabble-classics-online-misco/"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The biggest online money event of the century! Please join the
            MISCO; $1000 top prize with a $20 entry fee. See the flyer for more
            information. Four consecutive weekends, starting on March 6, 2021.
          </p>
        </a>
      ),
    },

    {
      title: 'Tournament Complete - Virtual Crescent City Cup',
      body: (
        <a
          href="https://woogles.io/tournament/vccc"
          target="_blank"
          rel="noopener noreferrer"
        >
          <p>
            The Virtual Crescent City Cup, the first one to use our brand new
            tournament interface, was a resounding success! Congratulations to
            Billy Nakamura (NWL Open), Lukeman Owolabi (CSW Open), Gideon
            Brosowsky (NWL Lite), and Terry Kang Rau (CSW Lite) for winning
            their respective divisions! All 850+ games were recorded and can be
            found by clicking on this text.
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
          href="https://discord.gg/GqkUqA7ENm"
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
