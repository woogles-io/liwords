import { useState, useEffect } from 'react';
import axios from 'axios';
import jwt from 'jsonwebtoken';
import { SendMessage } from 'react-use-websocket';
import {
  TokenSocketLogin,
  MessageType,
} from '../gen/api/proto/realtime/realtime_pb';
import { encodeToSocketFmt } from '../utils/protobuf';

type TokenResponse = {
  token: string;
};

type DecodedToken = {
  unn: string;
  uid: string;
  a: boolean; // authed
};

export const useSocketToken = (
  sendSocketMessage: SendMessage,
  connectedToSocket: boolean
) => {
  const [username, setUsername] = useState('Anonymous');
  const [userID, setUserID] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);

  useEffect(() => {
    if (!connectedToSocket) {
      console.log('not fetching while disconnected...');
      return;
    }
    console.log('fetching socket token...');
    axios
      .post<TokenResponse>(
        '/twirp/user_service.AuthenticationService/GetSocketToken',
        {}
      )
      .then((resp) => {
        const decoded = jwt.decode(resp.data.token) as DecodedToken;
        setUsername(decoded.unn);
        setUserID(decoded.uid);
        setLoggedIn(decoded.a);
        const msg = new TokenSocketLogin();
        msg.setToken(resp.data.token);
        // Decoding the token logs us in, and we also send the token to
        // the socket server to identify ourselves there as well.
        sendSocketMessage(
          encodeToSocketFmt(
            MessageType.TOKEN_SOCKET_LOGIN,
            msg.serializeBinary()
          )
        );
      })
      .catch((e) => {
        if (e.response) {
          window.console.log(e.response);
        }
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectedToSocket]);

  return {
    username,
    userID,
    loggedIn,
  };
};
