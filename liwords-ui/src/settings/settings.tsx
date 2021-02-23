import React, { useEffect } from 'react';
import { notification, Row, Col } from 'antd';
import { useMountedState } from '../utils/mounted';
import { TopBar } from '../topbar/topbar';
import { ChangePassword } from './change_password';
import { PersonalInfo } from './personal_info';
import { Preferences } from './preferences';
import { BlockedPlayers } from './blocked_players';
import { LogOut } from './log_out_woogles';
import { Support } from './support_woogles';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { useLoginStateStoreContext } from '../store/store';
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

type ProfileResponse = {
  avatar_url: string;
  full_name: string;
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

export const Settings = React.memo((props: Props) => {
  const { loginState } = useLoginStateStoreContext();
  const { username: viewer } = loginState;
  const { useState } = useMountedState();
  const [category, setCategory] = useState(Category.PersonalInfo);
  const [player, setPlayer] = useState<Partial<PlayerMetadata> | undefined>(
    undefined
  );

  type CategoryProps = {
    title: string;
    category: Category;
  };

  useEffect(() => {
    if (viewer === '') return;
    axios
      .post<ProfileResponse>(
        toAPIUrl('user_service.ProfileService', 'GetProfile'),
        {
          username: viewer,
        }
      )
      .then((resp) => {
        setPlayer({
          avatar_url: resp.data.avatar_url,
          full_name: resp.data.full_name,
        });
      })
      .catch(errorCatcher);
  }, [viewer]);

  const CategoryChoice = React.memo((props: CategoryProps) => {
    return (
      <div
        className={category === props.category ? 'choice active' : 'choice'}
        onClick={() => {
          setCategory(props.category);
        }}
      >
        {props.title}
      </div>
    );
  });

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
          {category === Category.LogOut ? <LogOut player={player} /> : null}
          {category === Category.Support ? <Support /> : null}
        </div>
      </div>
    </>
  );
});
