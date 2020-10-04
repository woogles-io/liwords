import { useCallback, useEffect, useState } from 'react';
import axios from 'axios';
import jwt from 'jsonwebtoken';
import useWebSocket from 'react-use-websocket';
import { useLocation } from 'react-router-dom';
import { useStoreContext } from '../store/store';
import { onSocketMsg } from '../store/socket_handlers';
import { decodeToMsg } from '../utils/protobuf';
import { toAPIUrl } from '../api/api';
import { ActionType } from '../actions/actions';

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

export const useLiwordsSocket = (disconnect = false) => {
  const socketUrl = getSocketURI();
  const store = useStoreContext();
  const location = useLocation();

  // const [socketToken, setSocketToken] = useState('');
  const [fullSocketUrl, setFullSocketUrl] = useState('');
  const [justDisconnected, setJustDisconnected] = useState(false);

  useEffect(() => {
    if (store.loginState.connectedToSocket) {
      // Only call this function if we are not connected to the socket.
      // If we go from unconnected to connected, there is no need to call
      // it again. If we go from connected to unconnected, then we call it
      // to fetch a new token.
      console.log('already connected');
      return;
    }
    console.log('About to request token');

    axios
      .post<TokenResponse>(
        toAPIUrl('user_service.AuthenticationService', 'GetSocketToken'),
        {},
        { withCredentials: true }
      )
      .then((resp) => {
        const socketToken = resp.data.token;
        const { cid } = resp.data;

        setFullSocketUrl(
          `${socketUrl}?token=${socketToken}&path=${location.pathname}&cid=${cid}`
        );

        const decoded = jwt.decode(socketToken) as DecodedToken;
        store.dispatchLoginState({
          actionType: ActionType.SetAuthentication,
          payload: {
            username: decoded.unn,
            userID: decoded.uid,
            loggedIn: decoded.a,
            connID: cid,
          },
        });
        console.log('Got token, setting state, and will try to connect...');
      })
      .catch((e) => {
        if (e.response) {
          window.console.log(e.response);
        }
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [store.loginState.connectedToSocket]);

  const { sendMessage } = useWebSocket(
    useCallback(() => fullSocketUrl, [fullSocketUrl]),
    {
      onOpen: () => {
        console.log('connected to socket');
        store.dispatchLoginState({
          actionType: ActionType.SetConnectedToSocket,
          payload: true,
        });
        setJustDisconnected(false);
      },
      onClose: () => {
        console.log('disconnected from socket :(');
        store.dispatchLoginState({
          actionType: ActionType.SetConnectedToSocket,
          payload: false,
        });
        setJustDisconnected(true);
      },
      retryOnError: true,
      shouldReconnect: (closeEvent) => true,
      onMessage: (event: MessageEvent) =>
        decodeToMsg(event.data, onSocketMsg(store)),
    },
    !disconnect &&
      fullSocketUrl !== '' /* only connect if the socket token is not null */
  );

  return {
    sendMessage,
    justDisconnected,
  };
};
