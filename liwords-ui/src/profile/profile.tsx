import React, { useEffect, useState } from 'react';
import { useParams, useLocation, Link } from 'react-router-dom';
import { notification, Card, Table, Row, Col } from 'antd';
import axios, { AxiosError } from 'axios';
import { TopBar } from '../topbar/topbar';

type ProfileResponse = {
  first_name: string;
  last_name: string;
  country_code: string;
  title: string;
  about: string;
  ratings_json: string;
  stats_json: string;
};

const errorCatcher = (e: AxiosError) => {
  if (e.response) {
    notification.warning({
      message: 'Fetch Error',
      description: e.response.data.msg,
      duration: 4,
    });
  }
};

type StatItem = {
  n: string; // name
  t: number; // total
  a: Array<number>; // averages
};

type Rating = {
  r: number;
  rd: number;
  v: number;
};

type ProfileRatings = { [variant: string]: Rating };

type RatingsProps = {
  ratings: ProfileRatings;
};

type ProfileStats = {
  [variant: string]: {
    i1: string; // us
    i2: string; // opp
    d1: { [key: string]: StatItem }; // us
    d2: { [key: string]: StatItem }; // opp
    n: Array<StatItem>; // notable
  };
};

type StatsProps = {
  stats: ProfileStats;
};

const variantToName = (variant: string) => {
  const arr = variant.split('.');
  // get rid of the middle element (classic) for now
  const timectrl = {
    ultrablitz: 'Ultra-Blitz!',
    blitz: 'Blitz',
    rapid: 'Rapid',
    regular: 'Regular',
  }[arr[2] as 'ultrablitz' | 'blitz' | 'rapid' | 'regular']; // cmon typescript

  return `${arr[0]} (${timectrl})`;
};

const RatingsCard = (props: RatingsProps) => {
  const variants = Object.keys(props.ratings);
  console.log('ratings', props.ratings, variants);
  const dataSource = variants.map((v) => ({
    key: v,
    name: variantToName(v),
    rating: props.ratings[v].r.toFixed(2),
    deviation: props.ratings[v].rd.toFixed(2),
    volatility: props.ratings[v].v.toFixed(2),
  }));
  console.log('datasource', dataSource);

  const columns = [
    {
      title: 'Variant',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Rating',
      dataIndex: 'rating',
      key: 'rating',
    },
    {
      title: 'Deviation',
      dataIndex: 'deviation',
      key: 'deviation',
    },
    {
      title: 'Volatility',
      dataIndex: 'volatility',
      key: 'volatility',
    },
  ];

  return (
    <Card title="Ratings">
      <Table dataSource={dataSource} columns={columns} scroll={{ x: 500 }} />
    </Card>
  );
};

const StatsCard = (props: StatsProps) => {
  const variants = Object.keys(props.stats);

  console.log('stats', props.stats, variants);

  const dataSource = variants.map((v) => ({
    key: v,
    name: variantToName(v),
    games: props.stats[v].d1.Games.t,
    wld: `${props.stats[v].d1.Wins.t}-${props.stats[v].d1.Losses.t}-${props.stats[v].d1.Draws.t}`,
    avgScore: (props.stats[v].d1.Score.t / props.stats[v].d1.Games.t).toFixed(
      2
    ),
    avgPerTurn: (props.stats[v].d1.Score.t / props.stats[v].d1.Turns.t).toFixed(
      2
    ),
    avgScoreAgainst: (
      props.stats[v].d2.Score.t / props.stats[v].d2.Games.t
    ).toFixed(2),
    bingosPerGame: (
      props.stats[v].d1.Bingos.t / props.stats[v].d1.Games.t
    ).toFixed(2),
  }));

  const columns = [
    {
      title: 'Variant',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Games',
      dataIndex: 'games',
      key: 'games',
    },
    {
      title: 'W-L-D',
      dataIndex: 'wld',
      key: 'wld',
    },
    {
      title: 'Avg Score',
      dataIndex: 'avgScore',
      key: 'avgScore',
    },
    {
      title: 'Avg Per Turn',
      dataIndex: 'avgPerTurn',
      key: 'avgPerTurn',
    },
    {
      title: 'Opp Avg Score',
      dataIndex: 'avgScoreAgainst',
      key: 'avgScoreAgainst',
    },
    {
      title: 'Bingos Per Game',
      dataIndex: 'bingosPerGame',
      key: 'bingosPerGame',
    },
  ];

  return (
    <Card title="stats">
      <Table dataSource={dataSource} columns={columns} />
    </Card>
  );
};

type Props = {
  loggedIn: boolean;
  myUsername: string;
  connectedToSocket: boolean;
};

export const UserProfile = (props: Props) => {
  const { username } = useParams();
  const location = useLocation();
  // Show username's profile
  const [ratings, setRatings] = useState({});
  const [stats, setStats] = useState({});
  useEffect(() => {
    axios
      .post<ProfileResponse>('/twirp/user_service.ProfileService/GetProfile', {
        username,
      })
      .then((resp) => {
        console.log('prof', resp, JSON.parse(resp.data.ratings_json).Data);
        setRatings(JSON.parse(resp.data.ratings_json).Data);
        setStats(JSON.parse(resp.data.stats_json).Data);
      })
      .catch(errorCatcher);
  }, [username, location.pathname]);

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar
            username={props.myUsername}
            loggedIn={props.loggedIn}
            connectedToSocket={props.connectedToSocket}
          />
        </Col>
      </Row>

      <Row>
        <Col span={12} offset={12}>
          <h2>Profile for {username}</h2>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={12}>
          <RatingsCard ratings={ratings} />
        </Col>
        <Col span={12}>
          <StatsCard stats={stats} />
        </Col>
      </Row>
      <Row style={{ marginTop: 20 }}>
        <Col span={12} offset={12}>
          <Link to="/password/change">
            <big>Change your password</big>
          </Link>
        </Col>
      </Row>
    </>
  );
};
