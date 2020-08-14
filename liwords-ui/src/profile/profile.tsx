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
    d1: Array<StatItem>; // us
    d2: Array<StatItem>; // opp
    n: Array<StatItem>; // notable
  };
};

type StatsProps = {
  stats: ProfileStats;
};

const RatingsCard = (props: RatingsProps) => {
  const variants = Object.keys(props.ratings);
  console.log('ratings', props.ratings, variants);
  const dataSource = variants.map((v) => ({
    key: v,
    name: v,
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
    name: v,
    // Obviously hardcoded numbers here...
    wins: props.stats[v].d1[27].t,
    losses: props.stats[v].d1[14].t,
    draws: props.stats[v].d1[8].t,
    avgScore: props.stats[v].d1[20].a[0].toFixed(2),
    avgPerTurn: props.stats[v].d1[20].a[1].toFixed(2),
    bingosPerGame: props.stats[v].d1[1].a[0].toFixed(2),
  }));

  const columns = [
    {
      title: 'Variant',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Wins',
      dataIndex: 'wins',
      key: 'wins',
    },
    {
      title: 'Losses',
      dataIndex: 'losses',
      key: 'losses',
    },
    {
      title: 'Draws',
      dataIndex: 'draws',
      key: 'draws',
    },
    {
      title: 'Average Score',
      dataIndex: 'avgScore',
      key: 'avgScore',
    },
    {
      title: 'Average Per Turn',
      dataIndex: 'avgPerTurn',
      key: 'avgPerTurn',
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
          <TopBar username={props.myUsername} loggedIn={props.loggedIn} />
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
