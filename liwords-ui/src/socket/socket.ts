import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from 'react';
import axios from 'axios';
import jwt from 'jsonwebtoken';
import useWebSocket from 'react-use-websocket';
import { useLocation } from 'react-router-dom';
import { message } from 'antd';
import { useMountedState } from '../utils/mounted';
import { useLoginStateStoreContext } from '../store/store';
import {
  ReverseMessageType,
  enableShowSocket,
  parseMsgs,
} from '../store/socket_handlers';
import { decodeToMsg } from '../utils/protobuf';
import { toAPIUrl } from '../api/api';
import { ActionType } from '../actions/actions';
import { reloadAction } from './reload';
import { birthdateWarning } from './birthdateWarning';

// Store-specific code.

const defaultFunction = () => {};

export type LiwordsSocketValues = {
  sendMessage: (msg: Uint8Array) => void;
  justDisconnected: boolean;
};

export type OnSocketMsgType = (reader: FileReader) => void;

type LiwordsSocketStoreData = {
  liwordsSocketValues: LiwordsSocketValues;
  onSocketMsg: OnSocketMsgType;
  resetLiwordsSocketStore: () => void;
  setLiwordsSocketValues: React.Dispatch<
    React.SetStateAction<LiwordsSocketValues>
  >;
  setOnSocketMsg: React.Dispatch<React.SetStateAction<OnSocketMsgType>>;
};

export const LiwordsSocketContext = createContext<LiwordsSocketStoreData>({
  liwordsSocketValues: {
    sendMessage: defaultFunction,
    justDisconnected: false,
  },
  onSocketMsg: defaultFunction,
  resetLiwordsSocketStore: defaultFunction,
  setLiwordsSocketValues: defaultFunction,
  setOnSocketMsg: defaultFunction,
});

export const useLiwordsSocketContext = () => useContext(LiwordsSocketContext);

// Non-Store code follows.

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

// this only depends on protocol and host, will never change as we navigate SPA.
const socketUrl = getSocketURI();

type TokenResponse = {
  token: string;
  cid: string;
  front_end_version: string;
};

type DecodedToken = {
  unn: string;
  uid: string;
  cs: string;
  a: boolean; // authed
  perms: string;
};

// Returning undefined from useEffect is fine, but some linters dislike it.
const doNothing = () => {};

export const LiwordsSocket = (props: {}): null => {
  const isMountedRef = useRef(true);
  useEffect(() => () => void (isMountedRef.current = false), []);
  const { useState } = useMountedState();

  const {
    onSocketMsg,
    resetLiwordsSocketStore,
    setLiwordsSocketValues,
  } = useLiwordsSocketContext();

  const loginStateStore = useLoginStateStoreContext();
  const location = useLocation();
  const pathname = useMemo(() => {
    const originalPathname = location.pathname;
    // XXX: The socket requires path to know which realms it has to connect to.
    // See liwords-socket pkg/hub/hub.go RegisterRealm.
    // That calls back into liwords pkg/bus/bus.go handleNatsRequest.

    // It seems only a few paths matter.
    if (
      originalPathname.startsWith('/game/') ||
      originalPathname.startsWith('/tournament/') ||
      originalPathname.startsWith('/club/')
    )
      return originalPathname;

    // For everything else, there's MasterCard.
    return '/';
  }, [location.pathname]);

  // const [socketToken, setSocketToken] = useState('');
  const [justDisconnected, setJustDisconnected] = useState(false);

  // Source-of-truth must be local, not the store.
  const [isConnectedToSocket, setIsConnectedToSocket] = useState(false);
  const { dispatchLoginState } = loginStateStore;
  const getFullSocketUrlAsync = useCallback(async () => {
    console.log('About to request token');
    // Unfortunately this function must return a valid url.
    const failUrl = `${socketUrl}?${new URLSearchParams({
      path: pathname,
    })}`;

    try {
      const resp = await axios.post<TokenResponse>(
        toAPIUrl('user_service.AuthenticationService', 'GetSocketToken'),
        {},
        { withCredentials: true }
      );
      // Important: resetSocket does not resetStore, be very careful to avoid
      // dispatching stuffs from a decommissioned socket after axios returns.
      if (!isMountedRef.current) return failUrl;

      const socketToken = resp.data.token;
      const { cid, front_end_version } = resp.data;

      const ret = `${socketUrl}?${new URLSearchParams({
        token: socketToken,
        path: pathname,
        cid,
      })}`;

      const decoded = jwt.decode(socketToken) as DecodedToken;
      dispatchLoginState({
        actionType: ActionType.SetAuthentication,
        payload: {
          username: decoded.unn,
          userID: decoded.uid,
          loggedIn: decoded.a,
          connID: cid,
          perms: decoded.perms?.split(','),
        },
      });
      const bdateWarning = localStorage?.getItem('birthdateWarning');
      if (
        parseInt(decoded.cs) === 2 &&
        decoded.a &&
        (!bdateWarning ||
          Date.now() - parseInt(bdateWarning) > 24 * 3600 * 1000)
      ) {
        message.warning({
          content: birthdateWarning,
          className: 'board-hud-message',
          key: 'birthdate-warning',
          duration: 5,
        });
        // Only warn them once a day
        localStorage.setItem('birthdateWarning', Date.now().toString());
      }
      if (!isMountedRef.current) return failUrl;
      console.log('Got token, setting state, and will try to connect...');
      if (window.RUNTIME_CONFIGURATION.appVersion !== front_end_version) {
        console.log(
          'app version mismatch',
          'local',
          window.RUNTIME_CONFIGURATION.appVersion,
          'remote',
          front_end_version
        );

        if (front_end_version !== '') {
          message.warning({
            content: reloadAction,
            className: 'board-hud-message',
            key: 'reload-warning',
            duration: 0,
          });
        }
      }

      return ret;
    } catch (e) {
      if (e.response) {
        window.console.log(e.response);
      }
      return failUrl;
    }
  }, [dispatchLoginState, pathname]);

  useEffect(() => {
    if (isConnectedToSocket) {
      console.log('connected to socket');
      dispatchLoginState({
        actionType: ActionType.SetConnectedToSocket,
        payload: true,
      });
      message.destroy('connecting-socket');
      setJustDisconnected(false);
      return () => {
        if (isMountedRef.current) {
          console.log('disconnected from socket :(');
        } else {
          // Yes, the smiley matters!
          console.log('disconnected from socket :)');
        }
        // Special case: useEffect cleanups seem to be run in forward order,
        // but resetSocket does not imply resetStore, and it is important that
        // we inform loginStateStore of the unmount.
        dispatchLoginState({
          actionType: ActionType.SetConnectedToSocket,
          payload: false,
        });
        setJustDisconnected(true);
      };
    }
    return doNothing;
  }, [dispatchLoginState, isConnectedToSocket]);

  useEffect(() => {
    if (isConnectedToSocket) {
      return doNothing;
    }
    const t = setTimeout(() => {
      message.warning({
        content: 'Connecting to server...',
        duration: 0,
        key: 'connecting-socket',
      });
    }, 2000);
    return () => {
      clearTimeout(t);
    };
  }, [isConnectedToSocket]);

  const [patienceId, setPatienceId] = useState(0);
  const resetPatience = useCallback(
    () => setPatienceId((n) => (n + 1) | 0),
    []
  );
  useEffect(() => {
    const t = setTimeout(() => {
      console.log('reconnecting socket');
      resetLiwordsSocketStore();
    }, 15000);
    return () => {
      clearTimeout(t);
    };
  }, [patienceId, resetLiwordsSocketStore]);

  // Force reconnection when pathname materially changes.
  const knownPathname = useRef(pathname); // Remember the pathname on first render.
  const isCurrentPathname = knownPathname.current === pathname;
  useEffect(() => {
    if (!isCurrentPathname) {
      resetLiwordsSocketStore();
    }
  }, [isCurrentPathname, resetLiwordsSocketStore]);

  const { sendMessage: originalSendMessage } = useWebSocket(
    getFullSocketUrlAsync,
    {
      onOpen: () => {
        resetPatience();
        setIsConnectedToSocket(true);
      },
      onClose: () => {
        resetPatience();
        setIsConnectedToSocket(false);
      },
      reconnectAttempts: Infinity,
      reconnectInterval: 1000,
      retryOnError: true,
      shouldReconnect: (closeEvent) => true,
      onMessage: (event: MessageEvent) => {
        // Any incoming message resets the patience.
        resetPatience();
        return decodeToMsg(event.data, onSocketMsg);
      },
    }
  );

  const sendMessage = useMemo(() => {
    if (!enableShowSocket) return originalSendMessage;

    return (msg: Uint8Array) => {
      const msgs = parseMsgs(msg);

      msgs.forEach((m) => {
        const { msgType, parsedMsg } = m;
        console.log(
          '%csent',
          'background: cyan',
          ReverseMessageType[msgType] ?? msgType,
          parsedMsg.toObject(),
          performance.now(),
          'bytelength:',
          msg.byteLength
        );
      });

      return originalSendMessage(msg);
    };
  }, [originalSendMessage]);

  const ret = useMemo(() => ({ sendMessage, justDisconnected }), [
    sendMessage,
    justDisconnected,
  ]);
  useEffect(() => {
    setLiwordsSocketValues(ret);
  }, [setLiwordsSocketValues, ret]);

  return null;
};
