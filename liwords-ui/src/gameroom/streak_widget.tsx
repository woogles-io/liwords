import { Card, Col, Row } from "antd";
import React from "react";
import { Link } from "react-router";
import {
  StreakInfoResponse,
  StreakInfoResponse_SingleGameInfo,
} from "../gen/api/proto/game_service/game_service_pb";

type Props = {
  streakInfo: StreakInfoResponse;
};

type SGProps = {
  game: StreakInfoResponse_SingleGameInfo;
  p0win: number;
  p1win: number;
};

const SingleGame = (props: SGProps) => {
  const win = <p className="streak-win">1</p>;
  const loss = <p className="streak-loss">0</p>;
  const tie = <p className="streak-tie">½</p>;

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
    <div style={{ display: "inline-block", marginLeft: 10 }}>{cells}</div>
  );

  return (
    <span>
      <Link to={`/game/${encodeURIComponent(String(props.game.gameId ?? ""))}`}>
        {innerel}
      </Link>
    </span>
  );
};

export const StreakWidget = React.memo((props: Props) => {
  if (
    !props.streakInfo ||
    !props.streakInfo.streak ||
    props.streakInfo.streak.length === 0
  ) {
    return null;
  }
  // Determine which player is listed on top and which on bottom.
  const first = props.streakInfo.playersInfo[0].nickname;
  const second = props.streakInfo.playersInfo[1].nickname;

  let p0wins = 0;
  let p1wins = 0;

  const pergame = props.streakInfo.streak
    .slice(0) // reverse a shallow copy of the array.
    .reverse()
    .map((g) => {
      let p0win = 0;
      let p1win = 0;

      if (g.winner === 0) {
        p0win = 1;
        p1win = 0;
      } else if (g.winner === 1) {
        p0win = 0;
        p1win = 1;
      } else if (g.winner === -1) {
        p0win = 0.5;
        p1win = 0.5;
      }
      p0wins += p0win;
      p1wins += p1win;
      return <SingleGame game={g} key={g.gameId} p0win={p0win} p1win={p1win} />;
    });

  const pStyle = {
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap" as const,
  };

  return (
    <Card className="streak-widget" style={{ marginTop: 10 }}>
      <Row>
        <Col span={16} style={{ justifyContent: "right", textAlign: "right" }}>
          {pergame}
        </Col>
        <Col span={6}>
          <div style={{ marginLeft: 20 }}>
            <p style={pStyle}>{first}</p>
            <p style={pStyle}>{second}</p>
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
