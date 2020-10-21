import { Card, Col, Row } from 'antd';
import React from 'react';
import { Link } from 'react-router-dom';
import { useResetStoreContext } from '../store/store';
import { GameMetadata } from './game_info';

type Props = {
  recentGames: Array<GameMetadata>;
};

type SGProps = {
  game: GameMetadata;
  p0win: number;
  p1win: number;
};

const SingleGame = (props: SGProps) => {
  const { resetStore } = useResetStoreContext();
  const win = <p style={{ color: 'green' }}>1</p>;
  const loss = <p style={{ color: 'lightgray' }}>0</p>;
  const tie = <p style={{ color: 'blue' }}>½</p>;

  let cells;

  if (props.p0win === 1) {
    cells = (
      <>
        {win}
        {loss}
      </>
    );
  } else if (props.p1win === 1) {
    cells = (
      <>
        {loss}
        {win}
      </>
    );
  } else if (props.p0win === 0.5) {
    cells = (
      <>
        {tie}
        {tie}
      </>
    );
  }

  const innerel = (
    <div style={{ display: 'inline-block', marginLeft: 10 }}>{cells}</div>
  );

  return (
    <span>
      <Link
        to={`/game/${encodeURIComponent(String(props.game.game_id ?? ''))}`}
        onClick={resetStore}
      >
        {innerel}
      </Link>
    </span>
  );
};

export const StreakWidget = React.memo((props: Props) => {
  if (props.recentGames.length === 0) {
    return null;
  }
  // Determine which player is listed on top and which on bottom.
  let first = props.recentGames[0].players[0].nickname;
  let second = props.recentGames[0].players[1].nickname;
  if (second > first) {
    [first, second] = [second, first];
  }

  let p0wins = 0;
  let p1wins = 0;

  const pergame = props.recentGames
    .slice(0) // reverse a shallow copy of the array.
    .reverse()
    .map((g) => {
      let p0idx;
      let p1idx;
      let p0win = 0;
      let p1win = 0;
      if (first === g.players[0].nickname) {
        p0idx = 0;
        p1idx = 1;
      } else {
        p0idx = 1;
        p1idx = 0;
      }

      if ((g.winner === 0 && p0idx === 0) || (g.winner === 1 && p0idx === 1)) {
        p0win = 1;
        p1win = 0;
      } else if (
        (g.winner === 1 && p1idx === 1) ||
        (g.winner === 0 && p1idx === 0)
      ) {
        p0win = 0;
        p1win = 1;
      } else if (g.winner === -1) {
        p0win = 0.5;
        p1win = 0.5;
      }
      p0wins += p0win;
      p1wins += p1win;

      return (
        <SingleGame game={g} key={g.game_id} p0win={p0win} p1win={p1win} />
      );
    });

  return (
    <Card style={{ marginTop: 10 }}>
      <Row>
        <Col span={16} style={{ justifyContent: 'right', textAlign: 'right' }}>
          {pergame}
        </Col>
        <Col span={6}>
          <div style={{ marginLeft: 20, display: 'inline-block' }}>
            <p>{first}</p>
            <p>{second}</p>
          </div>
        </Col>
        <Col span={2}>
          <p>{p0wins}</p>
          <p>{p1wins}</p>
        </Col>
      </Row>
    </Card>
  );
});
