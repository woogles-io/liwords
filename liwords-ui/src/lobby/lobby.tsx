import React, { useState } from 'react';
import axios from 'axios';
import { Row, Col, Button, Card, Input } from 'antd';

import { TopBar } from '../topbar/topbar';

const newGame = (seeker: string, acceptor: string) => {
  axios
    .post('/api/acceptedseek', {
      seeker,
      acceptor,
    })
    .then((response) => {
      console.log(response.data);
    });
};

const sendSeek = (game: SoughtGame) => {
  axios.post('/api/sendseek', {
    seeker: game.seeker,
    lexicon: game.lexicon,
    timeControl: game.timeControl,
    challengeRule: game.challengeRule,
  });
};

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  timeControl: string;
  challengeRule: string;
  // rating: number;
  // soughtID: string;
};

type Props = {
  soughtGames: Array<SoughtGame>;
  onConnect: (username: string) => void;
};

export const Lobby = (props: Props) => {
  const [username, setUsername] = useState('');

  const soughtGames = props.soughtGames.map((game, idx) => (
    <li
      key={`game${game.seeker}`}
      style={{ paddingTop: 20, cursor: 'pointer' }}
      onClick={(event: React.MouseEvent) => newGame(game.seeker, username)}
      role="button"
    >
      {game.seeker} wants to play {game.lexicon} ({game.timeControl}{' '}
      {game.challengeRule})
    </li>
  ));

  return (
    <div>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>

      <Row>
        <Col span={4} offset={10}>
          <Input
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <Button onClick={() => props.onConnect(username)}>Connect</Button>
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <h3>Lobby</h3>
        </Col>
      </Row>
      <Row>
        <Col span={8} offset={8}>
          <Card title="Join a game">
            <ul style={{ listStyleType: 'none' }}>{soughtGames}</ul>
          </Card>
        </Col>
      </Row>

      <Row>
        <Col span={24}>
          <Button
            onClick={() =>
              sendSeek({
                seeker: username,
                lexicon: 'CSW19',
                challengeRule: '5-pt',
                timeControl: '15/0',
                // rating: 0,
                // soughtID: username,
              })
            }
          >
            New Game
          </Button>
        </Col>
      </Row>
    </div>
  );
};
