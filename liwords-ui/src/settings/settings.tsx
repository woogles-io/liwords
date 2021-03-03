import React, { useEffect, useCallback } from 'react';
import { notification, Row, Col } from 'antd';
import { useMountedState } from '../utils/mounted';
import { TopBar } from '../topbar/topbar';
import { ChangePassword } from './change_password';
import { PersonalInfo } from './personal_info';
import { CloseAccount } from './close_account';
import { Preferences } from './preferences';
import { BlockedPlayers } from './blocked_players';
import { LogOut } from './log_out_woogles';
import { Support } from './support_woogles';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { useLoginStateStoreContext } from '../store/store';
import { PlayerMetadata } from '../gameroom/game_info';
import { useHistory } from 'react-router-dom';
import { useResetStoreContext } from '../store/store';

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

type PersonalInfoResponse = {
  avatar_url: string;
  full_name: string;
  first_name: string;
  last_name: string;
  country_code: string;
  email: string;
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
  const { resetStore } = useResetStoreContext();
  const [category, setCategory] = useState(Category.PersonalInfo);
  const [player, setPlayer] = useState<Partial<PlayerMetadata> | undefined>(
    undefined
  );
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [countryCode, setCountryCode] = useState('');
  const [email, setEmail] = useState('');
  const [showCloseAccount, setShowCloseAccount] = useState(false);
  const history = useHistory();

  type CategoryProps = {
    title: string;
    category: Category;
  };

  useEffect(() => {
    if (viewer === '') return;
    axios
      .post<PersonalInfoResponse>(
        toAPIUrl('user_service.ProfileService', 'GetPersonalInfo'),
        {
          username: viewer,
        }
      )
      .then((resp) => {
        setPlayer({
          avatar_url: resp.data.avatar_url,
          full_name: resp.data.full_name,
        });
        setFirstName(resp.data.first_name);
        setLastName(resp.data.last_name);
        setCountryCode(resp.data.country_code);
        setEmail(resp.data.email);
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

  const handleContribute = useCallback(() => {
    history.push('/'); // Need a Contribute page
  }, [history]);

  const handleLogout = useCallback(() => {
    axios
      .post(toAPIUrl('user_service.AuthenticationService', 'Logout'), {
        withCredentials: true,
      })
      .then(() => {
        notification.info({
          message: 'Success',
          description: 'You have been logged out.',
        });
        resetStore();
        history.push('/');
      })
      .catch((e) => {
        console.log(e);
      });
  }, [history, resetStore]);

  const updatedAvatar = useCallback(
    (avatarUrl: string) => {
      setPlayer({ ...player, avatar_url: avatarUrl });
    },
    [player]
  );

  const startClosingAccount = useCallback(() => {
    setShowCloseAccount(true);
  }, []);

  const closeAccountNow = useCallback(() => {
    console.log('CLOSE ACCOUNT');
  }, []);

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
            showCloseAccount ? (
              <CloseAccount
                closeAccountNow={closeAccountNow}
                player={player}
                cancel={() => {
                  setShowCloseAccount(false);
                }}
              />
            ) : (
              <PersonalInfo
                player={player}
                personalInfo={{
                  email: email,
                  firstName: firstName,
                  lastName: lastName,
                  countryCode: countryCode,
                }}
                updatedAvatar={updatedAvatar}
                startClosingAccount={startClosingAccount}
              />
            )
          ) : null}
          {category === Category.ChangePassword ? <ChangePassword /> : null}
          {category === Category.Preferences ? <Preferences /> : null}
          {category === Category.BlockedPlayers ? <BlockedPlayers /> : null}
          {category === Category.LogOut ? (
            <LogOut player={player} handleLogout={handleLogout} />
          ) : null}
          {category === Category.Support ? (
            <Support handleContribute={handleContribute} />
          ) : null}
        </div>
      </div>
    </>
  );
});
