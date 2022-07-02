import React, { useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { notification } from 'antd';
import { useMountedState } from '../utils/mounted';
import { TopBar } from '../navigation/topbar';
import { ChangePassword } from './change_password';
import { PersonalInfo } from './personal_info';
import { CloseAccount } from './close_account';
import { ClosedAccount } from './closed_account';
import { Preferences } from './preferences';
import { BlockedPlayers } from './blocked_players';
import { LogOut } from './log_out_woogles';
import { Support } from './support_woogles';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { useLoginStateStoreContext } from '../store/store';
import { PlayerMetadata } from '../gameroom/game_info';
import { useNavigate } from 'react-router-dom';
import { useResetStoreContext } from '../store/store';

import './settings.scss';
import { Secret } from './secret';
import { HeartFilled } from '@ant-design/icons';

enum Category {
  PersonalInfo = 1,
  ChangePassword,
  Preferences,
  BlockedPlayers,
  Secret,
  LogOut,
  Support,
  NoUser,
}

type PersonalInfoResponse = {
  avatar_url: string;
  birth_date: string;
  full_name: string;
  first_name: string;
  last_name: string;
  country_code: string;
  email: string;
  about: string;
  silent_mode: boolean;
};

const getInitialCategory = (categoryShortcut: string, loggedIn: boolean) => {
  // We don't want to keep /donate or any other shortcuts in the url after reading it on first load
  // These are just shortcuts for backwards compatibility so we can send existing urls to the new
  // settings pages
  if (!loggedIn && categoryShortcut === 'donate') {
    // Don't redirect if they aren't logged in and just donating
    return Category.Support;
  }
  window.history.replaceState({}, 'settings', '/settings');
  switch (categoryShortcut) {
    case 'donate':
    case 'support':
      return Category.Support;
    case 'personal':
      return Category.PersonalInfo;
    case 'password':
      return Category.ChangePassword;
    case 'preferences':
      return Category.Preferences;
    case 'secret':
      return Category.Secret;
    case 'blocked':
      return Category.BlockedPlayers;
    case 'logout':
      return Category.LogOut;
  }
  // to be streaming-friendly, PersonalInfo should not be the default tab.
  return Category.Preferences;
};

export const Settings = React.memo(() => {
  const { loginState } = useLoginStateStoreContext();
  const { userID, username: viewer, loggedIn } = loginState;
  const { useState } = useMountedState();
  const { resetStore } = useResetStoreContext();
  let { section } = useParams();
  if (!section) {
    section = '';
  }
  const [category, setCategory] = useState(
    getInitialCategory(section, loggedIn)
  );
  const [player, setPlayer] = useState<Partial<PlayerMetadata> | undefined>(
    undefined
  );
  const [personalInfo, setPersonalInfo] = useState<PersonalInfo | undefined>(
    undefined
  );
  const [showCloseAccount, setShowCloseAccount] = useState(false);
  const [showClosedAccount, setShowClosedAccount] = useState(false);
  const [accountClosureError, setAccountClosureError] = useState('');
  const navigate = useNavigate();

  const errorCatcher = (e: AxiosError) => {
    console.log(e);
    if (e.response) {
      notification.warning({
        message: 'Fetch Error',
        description: e.response.data.msg,
        duration: 4,
      });
    }
  };

  useEffect(() => {
    if (viewer === '' || (!loggedIn && category === Category.Support)) return;
    if (!loggedIn) {
      setCategory(Category.NoUser);
      return;
    }
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
          user_id: userID, // for name-based avatar initial to work
        });
        setPersonalInfo({
          birthDate: resp.data.birth_date,
          email: resp.data.email,
          firstName: resp.data.first_name,
          lastName: resp.data.last_name,
          countryCode: resp.data.country_code,
          about: resp.data.about,
          silentMode: resp.data.silent_mode,
        });
      })
      .catch(errorCatcher);
  }, [viewer, loggedIn, category, userID]);

  type CategoryProps = {
    title: string | React.ReactNode;
    category: Category;
  };

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
    navigate('/settings');
  }, [navigate]);

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
        navigate('/');
      })
      .catch((e) => {
        console.log(e);
      });
  }, [navigate, resetStore]);

  const updatedAvatar = useCallback(
    (avatarUrl: string) => {
      setPlayer({ ...player, avatar_url: avatarUrl });
    },
    [player]
  );

  const startClosingAccount = useCallback(() => {
    setAccountClosureError('');
    setShowCloseAccount(true);
  }, []);

  const closeAccountNow = useCallback(
    (pw: string) => {
      axios
        .post(
          toAPIUrl(
            'user_service.AuthenticationService',
            'NotifyAccountClosure'
          ),
          { password: pw },
          {
            withCredentials: true,
          }
        )
        .then(() => {
          setShowCloseAccount(false);
          setShowClosedAccount(true);
          handleLogout();
        })
        .catch((e) => {
          if (e.response) {
            // From Twirp
            setAccountClosureError(e.response.data.msg);
          } else {
            setAccountClosureError('unknown error, see console');
            console.log(e);
          }
        });
    },
    [handleLogout]
  );

  const logIn = <div className="log-in">Log in to see your settings</div>;

  const categoriesColumn = (
    <div className="categories">
      <CategoryChoice title="Personal info" category={Category.PersonalInfo} />
      <CategoryChoice
        title="Change password"
        category={Category.ChangePassword}
      />
      <CategoryChoice title="Preferences" category={Category.Preferences} />
      <CategoryChoice
        title="Blocked players list"
        category={Category.BlockedPlayers}
      />
      <CategoryChoice title="Secret features" category={Category.Secret} />
      <CategoryChoice title="Log out" category={Category.LogOut} />
      <CategoryChoice
        title={
          <>
            <HeartFilled />
            Support Woogles
          </>
        }
        category={Category.Support}
      />
    </div>
  );

  return (
    <>
      <TopBar />
      {loggedIn ? (
        <div className="settings">
          {categoriesColumn}
          <div className="category">
            {category === Category.PersonalInfo ? (
              showCloseAccount ? (
                <CloseAccount
                  closeAccountNow={closeAccountNow}
                  player={player}
                  err={accountClosureError}
                  cancel={() => {
                    setShowCloseAccount(false);
                  }}
                />
              ) : showClosedAccount ? (
                <ClosedAccount />
              ) : personalInfo ? (
                <PersonalInfo
                  player={player}
                  personalInfo={personalInfo}
                  updatedAvatar={updatedAvatar}
                  startClosingAccount={startClosingAccount}
                />
              ) : null
            ) : null}
            {category === Category.ChangePassword ? <ChangePassword /> : null}
            {category === Category.Preferences ? <Preferences /> : null}
            {category === Category.Secret ? <Secret /> : null}
            {category === Category.BlockedPlayers ? <BlockedPlayers /> : null}
            {category === Category.LogOut ? (
              <LogOut player={player} handleLogout={handleLogout} />
            ) : null}
            {category === Category.Support ? (
              <Support handleContribute={handleContribute} />
            ) : null}
            {category === Category.NoUser ? logIn : null}
          </div>
        </div>
      ) : null}
      {!loggedIn && category === Category.Support ? (
        <div className="settings stand-alone">
          <Support handleContribute={handleContribute} />
        </div>
      ) : (
        !loggedIn && <div className="settings loggedOut">{logIn}</div>
      )}
    </>
  );
});
