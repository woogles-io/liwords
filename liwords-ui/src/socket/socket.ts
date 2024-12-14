import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { jwtDecode } from "jwt-decode";
import useWebSocket from "react-use-websocket";
import { useLocation } from "react-router-dom";
import { message } from "antd";
import { useLoginStateStoreContext } from "../store/store";
import {
  useOnSocketMsg,
  ReverseMessageType,
  enableShowSocket,
  parseMsgs,
} from "../store/socket_handlers";
import { decodeToMsg, encodeToSocketFmt } from "../utils/protobuf";
import { ActionType } from "../actions/actions";
import { reloadAction } from "./reload";
import { birthdateWarning } from "./birthdateWarning";
import { useClient } from "../utils/hooks/connect";
import { AuthenticationService } from "../gen/api/proto/user_service/user_service_pb";
import {
  create,
  DescMessage,
  MessageInitShape,
  toBinary,
} from "@bufbuild/protobuf";
import { MessageType } from "../gen/api/proto/ipc/ipc_pb";

const getSocketURI = (): string => {
  const loc = window.location;
  let protocol;
  if (loc.protocol === "https:") {
    protocol = "wss:";
  } else {
    protocol = "ws:";
  }
  const host = window.RUNTIME_CONFIGURATION.socketEndpoint || loc.host;

  return `${protocol}//${host}/ws`;
};

// this only depends on protocol and host, will never change as we navigate SPA.
const socketUrl = getSocketURI();

type DecodedToken = {
  unn: string;
  uid: string;
  cs: string;
  a: boolean; // authed
  perms: string;
};

// Returning undefined from useEffect is fine, but some linters dislike it.
const doNothing = () => {};

export const LiwordsSocket = (props: {
  resetSocket: () => void;
  setValues: (_: {
    sendMessage: (msg: Uint8Array) => void;
    justDisconnected: boolean;
    sendProtoSocketMsg: <T extends DescMessage>(
      schema: T,
      messageType: MessageType,
      data: MessageInitShape<T>,
    ) => void;
  }) => void;
}): null => {
  const isMountedRef = useRef(false);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const { resetSocket, setValues } = props;
  const onSocketMsg = useOnSocketMsg();

  const loginStateStore = useLoginStateStoreContext();
  const location = useLocation();
  const { pathname } = location;

  // const [socketToken, setSocketToken] = useState('');
  const [justDisconnected, setJustDisconnected] = useState(false);

  // Source-of-truth must be local, not the store.
  const [isConnectedToSocket, setIsConnectedToSocket] = useState(false);
  const { dispatchLoginState } = loginStateStore;
  const authClient = useClient(AuthenticationService);
  const getFullSocketUrlAsync = useCallback(async () => {
    console.log("About to request token");
    const failUrl = "";
    try {
      const resp = await authClient.getSocketToken({});

      // Important: resetSocket does not resetStore, be very careful to avoid
      // dispatching stuffs from a decommissioned socket after client returns.
      if (!isMountedRef.current) return failUrl;

      const { cid, frontEndVersion, token } = resp;

      const ret = `${socketUrl}?${new URLSearchParams({
        token,
        path: pathname,
        cid,
      })}`;

      const decoded = jwtDecode<DecodedToken>(token);
      dispatchLoginState({
        actionType: ActionType.SetAuthentication,
        payload: {
          username: decoded.unn,
          userID: decoded.uid,
          loggedIn: decoded.a,
          connID: cid,
          isChild: decoded.cs,
          path: pathname,
          perms: decoded.perms?.split(","),
        },
      });
      const bdateWarning = localStorage?.getItem("birthdateWarning");
      if (
        parseInt(decoded.cs) === 2 &&
        decoded.a &&
        (!bdateWarning ||
          Date.now() - parseInt(bdateWarning) > 24 * 3600 * 1000)
      ) {
        message.warning({
          content: birthdateWarning,
          className: "board-hud-message",
          key: "birthdate-warning",
          duration: 5,
        });
        // Only warn them once a day
        localStorage.setItem("birthdateWarning", Date.now().toString());
      }
      console.log("Got token, setting state, and will try to connect...");
      if (window.RUNTIME_CONFIGURATION.appVersion !== frontEndVersion) {
        console.log(
          "app version mismatch",
          "local",
          window.RUNTIME_CONFIGURATION.appVersion,
          "remote",
          frontEndVersion,
        );

        if (frontEndVersion !== "") {
          message.warning({
            content: reloadAction,
            className: "board-hud-message",
            key: "reload-warning",
            duration: 0,
          });
        }
      }
      return ret;
    } catch (e) {
      // XXX: Fix this; figure out what type of error this can be:
      if ((e as { [response: string]: string }).response) {
        window.console.log((e as { [response: string]: string }).response);
      } else {
        window.console.log("Unknown error", e);
      }
      return failUrl;
    }
  }, [dispatchLoginState, pathname, authClient]);

  useEffect(() => {
    if (isConnectedToSocket) {
      console.log("connected to socket");
      dispatchLoginState({
        actionType: ActionType.SetConnectedToSocket,
        payload: true,
      });
      message.destroy("connecting-socket");
      setJustDisconnected(false);
      return () => {
        if (isMountedRef.current) {
          console.log("disconnected from socket :(");
        } else {
          // Yes, the smiley matters!
          console.log("disconnected from socket :)");
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
        content: "Connecting to server...",
        duration: 0,
        key: "connecting-socket",
      });
    }, 2000);
    return () => {
      clearTimeout(t);
    };
  }, [isConnectedToSocket]);

  const [patienceId, setPatienceId] = useState(0);
  const resetPatience = useCallback(
    () => setPatienceId((n) => (n + 1) | 0),
    [],
  );
  useEffect(() => {
    const t = setTimeout(() => {
      console.log("reconnecting socket");
      resetSocket();
    }, 15000);
    return () => {
      clearTimeout(t);
    };
  }, [patienceId, resetSocket]);

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
      shouldReconnect: (closeEvent) => isMountedRef.current,
      onMessage: (event: MessageEvent) => {
        // Any incoming message resets the patience.
        resetPatience();
        return decodeToMsg(event.data, onSocketMsg);
      },
    },
  );

  const sendMessage = useMemo(() => {
    if (!enableShowSocket) return originalSendMessage;
    return (msg: Uint8Array) => {
      const msgs = parseMsgs(msg);
      msgs.forEach((m) => {
        const { msgType, parsedMsg } = m;
        console.log(
          "%csent",
          "background: cyan",
          ReverseMessageType[msgType] ?? msgType,
          parsedMsg,
          performance.now(),
          "bytelength:",
          msg.byteLength,
        );
      });

      return originalSendMessage(msg);
    };
  }, [originalSendMessage]);

  const sendProtoSocketMsg = useCallback(
    <T extends DescMessage>(
      schema: T,
      messageType: MessageType,
      data: MessageInitShape<T>,
    ) => {
      const message = create(schema, data); // Create the message with the provided schema and data
      const encodedMessage = encodeToSocketFmt(
        messageType,
        toBinary(schema, message),
      ); // Encode the message
      sendMessage(encodedMessage); // Send the encoded message
    },
    [sendMessage],
  );

  const ret = useMemo(
    () => ({ sendMessage, justDisconnected, sendProtoSocketMsg }),
    [sendMessage, justDisconnected, sendProtoSocketMsg],
  );
  useEffect(() => {
    setValues(ret);
  }, [setValues, ret]);

  return null;
};
