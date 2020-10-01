import { useEffect, useState } from 'react';
import axios from 'axios';
import jwt from 'jsonwebtoken';
import useWebSocket from 'react-use-websocket';
import { useLocation } from 'react-router-dom';
import { useStoreContext } from '../store/store';
import { onSocketMsg } from '../store/socket_handlers';
import { decodeToMsg } from '../utils/protobuf';
import { toAPIUrl } from '../api/api';

const getSocketURI = (): string => {
  const loc = window.location;
  let protocol;
  if (loc.protocol === 'https:') {
    protocol = 'wss:';
  } else {
    protocol = 'ws:';
  }
  const host = window.RUNTIME_CONFIGURATION.socketEndpoint || loc.host;

  return `${protocol}//${host}/ws`;
};

type TokenResponse = {
  token: string;
  cid: string;
};

type DecodedToken = {
  unn: string;
  uid: string;
  a: boolean; // authed
};

export const useLiwordsSocket = () => {
  const socketUrl = getSocketURI();
  const store = useStoreContext();
  const location = useLocation();

  const [socketToken, setSocketToken] = useState('');
  const [username, setUsername] = useState('Anonymous');
  const [connID, setConnID] = useState('');
  const [userID, setUserID] = useState('');
  const [loggedIn, setLoggedIn] = useState(false);
  const [connectedToSocket, setConnectedToSocket] = useState(false);
  const [justDisconnected, setJustDisconnected] = useState(false);

  useEffect(() => {
    if (connectedToSocket) {
      // Only call this function if we are not connected to the socket.
      // If we go from unconnected to connected, there is no need to call
      // it again. If we go from connected to unconnected, then we call it
      // to fetch a new token.
      return;
    }

    axios
      .post<TokenResponse>(
        toAPIUrl('user_service.AuthenticationService', 'GetSocketToken'),
        {},
        { withCredentials: true }
      )
      .then((resp) => {
        setSocketToken(resp.data.token);
        setConnID(resp.data.cid);
        const decoded = jwt.decode(resp.data.token) as DecodedToken;
        setUsername(decoded.unn);
        setUserID(decoded.uid);
        setLoggedIn(decoded.a);
        console.log('Got token, setting state');
      })
      .catch((e) => {
        if (e.response) {
          window.console.log(e.response);
        }
      });
  }, [connectedToSocket]);

  const { sendMessage } = useWebSocket(
    `${socketUrl}?token=${socketToken}&path=${location.pathname}&cid=${connID}`,
    {
      onOpen: () => {
        console.log('connected to socket');
        setConnectedToSocket(true);
        setJustDisconnected(false);
      },
      onClose: () => {
        console.log('disconnected from socket :(');
        setConnectedToSocket(false);
        setJustDisconnected(true);
      },
      retryOnError: true,
      // Will attempt to reconnect on all close events, such as server shutting down
      shouldReconnect: (closeEvent) => true,
      onMessage: (event: MessageEvent) =>
        decodeToMsg(event.data, onSocketMsg(username, connID, store)),
    },
    socketToken !== '' &&
      connID !== '' /* only connect if the socket token is not null */
  );

  return {
    sendMessage,
    userID,
    connID,
    username,
    loggedIn,
    connectedToSocket,
    justDisconnected,
  };
};
