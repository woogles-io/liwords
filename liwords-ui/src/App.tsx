import React, { useCallback, useEffect, useRef } from 'react';
import {
  Navigate,
  Route,
  Routes,
  useLocation,
  useSearchParams,
} from 'react-router-dom';
import { useMountedState } from './utils/mounted';
import './App.scss';
import axios from 'axios';
import 'antd/dist/antd.min.css';

import { Table as GameTable } from './gameroom/table';
import { SinglePuzzle } from './puzzles/puzzle';
import TileImages from './gameroom/tile_images';
import { Lobby } from './lobby/lobby';
import {
  useExcludedPlayersStoreContext,
  useLoginStateStoreContext,
  useResetStoreContext,
  useModeratorStoreContext,
  FriendUser,
  useFriendsStoreContext,
} from './store/store';

import { LiwordsSocket } from './socket/socket';
import { Team } from './about/team';
import { Register } from './lobby/register';
import { UserProfile } from './profile/profile';
import { Settings } from './settings/settings';
import { PasswordChange } from './lobby/password_change';
import { PasswordReset } from './lobby/password_reset';
import { NewPassword } from './lobby/new_password';
import { postJsonObj, toAPIUrl } from './api/api';
import { encodeToSocketFmt } from './utils/protobuf';
import { Clubs } from './clubs';
import { TournamentRoom } from './tournament/room';
import { Admin } from './admin/admin';
import { DonateSuccess } from './donate_success';
import { TermsOfService } from './about/termsOfService';
import { ChatMessage } from './gen/api/proto/ipc/chat_pb';
import { MessageType } from './gen/api/proto/ipc/ipc_pb';
import Footer from './navigation/footer';

type Blocks = {
  user_ids: Array<string>;
};

type ModsResponse = {
  admin_user_ids: Array<string>;
  mod_user_ids: Array<string>;
};

type FriendsResponse = {
  users: Array<FriendUser>;
};

const useDarkMode = localStorage?.getItem('darkMode') === 'true';
document?.body?.classList?.add(`mode--${useDarkMode ? 'dark' : 'default'}`);

const userTile = localStorage?.getItem('userTile');
if (userTile) {
  document?.body?.classList?.add(`tile--${userTile}`);
}

const userBoard = localStorage?.getItem('userBoard');
if (userBoard) {
  document?.body?.classList?.add(`board--${userBoard}`);
}
const bnjyTile = localStorage?.getItem('bnjyMode') === 'true';
if (bnjyTile) {
  document?.body?.classList?.add(`bnjyMode`);
}

// A temporary component until we auto-redirect with Cloudfront
const HandoverSignedCookie = () => {
  // No render.
  const [searchParams] = useSearchParams();
  const jwt = searchParams.get('jwt');
  const ls = searchParams.get('ls');
  const path = searchParams.get('path');

  const cookieSetFunc = useCallback(async () => {
    await postJsonObj(
      'user_service.AuthenticationService',
      'InstallSignedCookie',
      { jwt },
      () => {
        // we successfully transferred the cookie.
        if (ls) {
          const lsobj = JSON.parse(ls);
          console.log('got localstorage', lsobj);
          for (const k in lsobj) {
            localStorage.setItem(k, lsobj[k]);
          }
        }
        if (path) {
          window.location.replace(path);
        }
      }
    );
  }, [jwt, ls, path]);

  useEffect(() => {
    cookieSetFunc();
  }, [cookieSetFunc]);

  return null;
};

const App = React.memo(() => {
  const { useState } = useMountedState();

  const {
    setExcludedPlayers,
    setExcludedPlayersFetched,
    pendingBlockRefresh,
    setPendingBlockRefresh,
  } = useExcludedPlayersStoreContext();

  const { loginState } = useLoginStateStoreContext();
  const { loggedIn, userID } = loginState;

  const { setAdmins, setModerators, setModsFetched } =
    useModeratorStoreContext();

  const { setFriends, pendingFriendsRefresh, setPendingFriendsRefresh } =
    useFriendsStoreContext();

  const { resetStore } = useResetStoreContext();

  // See store.tsx for how this works.
  const [socketId, setSocketId] = useState(0);
  const resetSocket = useCallback(() => setSocketId((n) => (n + 1) | 0), []);

  const [liwordsSocketValues, setLiwordsSocketValues] = useState({
    sendMessage: (msg: Uint8Array) => {},
    justDisconnected: false,
  });
  const { sendMessage } = liwordsSocketValues;

  const location = useLocation();
  const knownLocation = useRef(location.pathname); // Remember the location on first render.
  const isCurrentLocation = knownLocation.current === location.pathname;
  useEffect(() => {
    if (!isCurrentLocation) {
      resetStore();
    }
  }, [isCurrentLocation, resetStore]);

  const getFullBlocks = useCallback(() => {
    void userID; // used only as effect dependency
    (async () => {
      let toExclude = new Set<string>();
      try {
        if (loggedIn) {
          const resp = await axios.post<Blocks>(
            toAPIUrl('user_service.SocializeService', 'GetFullBlocks'),
            {},
            { withCredentials: true }
          );
          toExclude = new Set<string>(resp.data.user_ids);
        }
      } catch (e) {
        console.log(e);
      } finally {
        setExcludedPlayers(toExclude);
        setExcludedPlayersFetched(true);
        setPendingBlockRefresh(false);
      }
    })();
  }, [
    loggedIn,
    userID,
    setExcludedPlayers,
    setExcludedPlayersFetched,
    setPendingBlockRefresh,
  ]);

  useEffect(() => {
    getFullBlocks();
  }, [getFullBlocks]);

  useEffect(() => {
    if (pendingBlockRefresh) {
      getFullBlocks();
    }
  }, [getFullBlocks, pendingBlockRefresh]);

  const getMods = useCallback(() => {
    axios
      .post<ModsResponse>(
        toAPIUrl('user_service.SocializeService', 'GetModList'),
        {},
        {}
      )
      .then((resp) => {
        setAdmins(new Set<string>(resp.data.admin_user_ids));
        setModerators(new Set<string>(resp.data.mod_user_ids));
      })
      .catch((e) => {
        console.log(e);
      })
      .finally(() => {
        setModsFetched(true);
      });
  }, [setAdmins, setModerators, setModsFetched]);

  useEffect(() => {
    getMods();
  }, [getMods]);

  const getFriends = useCallback(() => {
    if (loggedIn) {
      axios
        .post<FriendsResponse>(
          toAPIUrl('user_service.SocializeService', 'GetFollows'),
          {},
          {}
        )
        .then((resp) => {
          console.log('Fetched friends:', resp);
          const friends: { [uuid: string]: FriendUser } = {};
          resp.data.users.forEach((f: FriendUser) => {
            friends[f.uuid] = f;
          });
          setFriends(friends);
        })
        .catch((e) => {
          console.log(e);
        })
        .finally(() => setPendingFriendsRefresh(false));
    }
  }, [setFriends, setPendingFriendsRefresh, loggedIn]);

  useEffect(() => {
    getFriends();
  }, [getFriends]);

  useEffect(() => {
    if (pendingFriendsRefresh) {
      getFriends();
    }
  }, [getFriends, pendingFriendsRefresh]);

  const sendChat = useCallback(
    (msg: string, chan: string) => {
      const evt = new ChatMessage();
      evt.setMessage(msg);

      // const chan = isObserver ? 'gametv' : 'game';
      // evt.setChannel(`chat.${chan}.${gameID}`);
      evt.setChannel(chan);
      sendMessage(
        encodeToSocketFmt(MessageType.CHAT_MESSAGE, evt.serializeBinary())
      );
    },
    [sendMessage]
  );

  // Avoid useEffect in the new path triggering xhr twice.
  if (!isCurrentLocation) return null;

  return (
    <div className="App">
      <LiwordsSocket
        key={socketId}
        resetSocket={resetSocket}
        setValues={setLiwordsSocketValues}
      />
      <Routes>
        <Route
          path="/"
          element={
            <Lobby
              sendSocketMsg={sendMessage}
              sendChat={sendChat}
              DISCONNECT={resetSocket}
            />
          }
        />
        <Route
          path="tournament/:partialSlug/*"
          element={
            <TournamentRoom sendSocketMsg={sendMessage} sendChat={sendChat} />
          }
        />
        <Route
          path="club/:partialSlug/*"
          element={
            <TournamentRoom sendSocketMsg={sendMessage} sendChat={sendChat} />
          }
        />
        <Route path="clubs" element={<Clubs />} />
        <Route
          path="game/:gameID"
          element={
            <GameTable sendSocketMsg={sendMessage} sendChat={sendChat} />
          }
        />
        <Route path="puzzle" element={<SinglePuzzle sendChat={sendChat} />}>
          <Route
            path=":puzzleID"
            element={<SinglePuzzle sendChat={sendChat} />}
          />
        </Route>

        <Route path="about" element={<Team />} />
        <Route path="team" element={<Team />} />
        <Route path="terms" element={<TermsOfService />} />
        <Route path="register" element={<Register />} />
        <Route path="password">
          <Route path="change" element={<PasswordChange />} />
          <Route path="reset" element={<PasswordReset />} />
          <Route path="new" element={<NewPassword />} />
        </Route>
        <Route path="profile/:username" element={<UserProfile />} />
        <Route path="settings" element={<Settings />}>
          <Route path=":section" element={<Settings />} />
        </Route>
        <Route path="tile_images" element={<TileImages />}>
          <Route path=":letterDistribution" element={<TileImages />} />
        </Route>
        <Route path="admin" element={<Admin />} />
        <Route
          path="donate"
          element={<Navigate replace to="/settings/donate" />}
        />
        <Route path="donate_success" element={<DonateSuccess />} />
        <Route
          path="handover-signed-cookie"
          element={<HandoverSignedCookie />}
        />
      </Routes>
      <Footer />
    </div>
  );
});

export default App;
