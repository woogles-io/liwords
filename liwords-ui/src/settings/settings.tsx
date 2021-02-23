import React, { useCallback } from 'react';
import { Row, Col } from 'antd';
// import axios, { AxiosError } from 'axios';
import { useMountedState } from '../utils/mounted';
import { TopBar } from '../topbar/topbar';
import { ChangePassword } from './change_password';
import { PersonalInfo } from './personal_info';
import { Preferences } from './preferences';
import { BlockedPlayers } from './blocked_players';
import { LogOut } from './log_out_woogles';
import { Support } from './support_woogles';
import { PlayerMetadata } from '../gameroom/game_info';

import './settings.scss';

type Props = {};

enum Category {
  PersonalInfo = 1,
  ChangePassword,
  Preferences,
  BlockedPlayers,
  LogOut,
  Support,
}

export const Settings = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [category, setCategory] = useState(Category.PersonalInfo);

  const handleShowPersonalInfo = useCallback(() => {
    setCategory(Category.PersonalInfo);
  }, []);
  const handleShowChangePassword = useCallback(() => {
    setCategory(Category.ChangePassword);
  }, []);
  const handleShowPreferences = useCallback(() => {
    setCategory(Category.Preferences);
  }, []);
  const handleShowBlockedPlayers = useCallback(() => {
    setCategory(Category.BlockedPlayers);
  }, []);
  const handleShowLogOut = useCallback(() => {
    setCategory(Category.LogOut);
  }, []);
  const handleShowSupport = useCallback(() => {
    setCategory(Category.Support);
  }, []);

  type CategoryProps = {
    title: string;
    category: Category;
  };

  const CategoryChoice = React.memo((props: CategoryProps) => {
    return (
      <div
        className={category == props.category ? 'choice active' : 'choice'}
        onClick={() => {
          setCategory(props.category);
        }}
      >
        {props.title}
      </div>
    );
  });

  const player = {
    avatar_url:
      'https://woogles-uploads.s3.amazonaws.com/7rugV2GrytwcweCwQAf4Rk-87Dmj6qwj.jpg',
    full_name: 'Bob Smith',
  };

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="settings">
        <div className="categories">
          <CategoryChoice
            title="Personal Info"
            category={Category.PersonalInfo}
          />
          <CategoryChoice
            title="Change Password"
            category={Category.ChangePassword}
          />
          <CategoryChoice title="Preferences" category={Category.Preferences} />
          <CategoryChoice
            title="Blocked players list"
            category={Category.BlockedPlayers}
          />
          <CategoryChoice
            title="Log out of Woogles.io"
            category={Category.LogOut}
          />
          <CategoryChoice
            title="Support Woogles.io"
            category={Category.Support}
          />
        </div>
        <div className="category">
          {category === Category.PersonalInfo ? (
            <PersonalInfo player={player} />
          ) : null}
          {category === Category.ChangePassword ? <ChangePassword /> : null}
          {category === Category.Preferences ? <Preferences /> : null}
          {category === Category.BlockedPlayers ? <BlockedPlayers /> : null}
          {category === Category.LogOut ? <LogOut /> : null}
          {category === Category.Support ? <Support /> : null}
        </div>
      </div>
    </>
  );
});
