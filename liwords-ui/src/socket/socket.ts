import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { jwtDecode } from "jwt-decode";
import useWebSocket from "react-use-websocket";
import { useLocation } from "react-router";
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

// Global counters to track socket issues over time
const socketDebugCounters = {
  failUrlReturns: 0,
  patienceTimeouts: 0,
  wsErrors: 0,
  reconnectAttempts: 0,
  successfulConnections: 0,
  tokenFetchAttempts: 0,
  tokenFetchSuccesses: 0,
  componentMounts: 0,
  componentUnmounts: 0,
};

// Track active WebSocket test to diagnose connection limit issues
const testWebSocketConnectivity = async (url: string): Promise<void> => {
  return new Promise((resolve) => {
    console.log(
      "🧪 Testing WebSocket connectivity to:",
      url.substring(0, 50) + "...",
    );
    const testWs = new WebSocket(url);
    const timeout = setTimeout(() => {
      console.warn(
        "🧪 TEST: WebSocket connection timed out after 5s - possible connection limit!",
      );
      testWs.close();
      resolve();
    }, 5000);

    testWs.onopen = () => {
      clearTimeout(timeout);
      console.log("🧪 TEST: WebSocket connected successfully!");
      testWs.close();
      resolve();
    };
    testWs.onerror = () => {
      clearTimeout(timeout);
      console.error(
        "🧪 TEST: WebSocket error - may be hitting browser connection limit!",
      );
      resolve();
    };
    testWs.onclose = (e) => {
      clearTimeout(timeout);
      console.log("🧪 TEST: WebSocket closed", {
        code: e.code,
        reason: e.reason,
        wasClean: e.wasClean,
      });
      resolve();
    };
  });
};

// Expose counters and test function to window for easy console access
declare global {
  interface Window {
    getSocketDebugCounters?: () => typeof socketDebugCounters;
    testSocketConnection?: () => Promise<void>;
  }
}

if (typeof window !== "undefined") {
  window.getSocketDebugCounters = () => {
    console.table(socketDebugCounters);
    return socketDebugCounters;
  };
  window.testSocketConnection = async () => {
    const testUrl = `${socketUrl}?token=test&path=/test&cid=test-${Date.now()}`;
    await testWebSocketConnectivity(testUrl);
  };
  console.log(
    "💡 Debug tip: Type getSocketDebugCounters() to see stats, testSocketConnection() to test WS connectivity",
  );
}

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
    socketDebugCounters.componentMounts++;
    console.log("🟢 LiwordsSocket MOUNTED", {
      mountCount: socketDebugCounters.componentMounts,
      unmountCount: socketDebugCounters.componentUnmounts,
      timestamp: new Date().toISOString(),
    });
    isMountedRef.current = true;

    // Add beforeunload listener to help Chrome clean up WebSocket connections
    const handleBeforeUnload = () => {
      console.log("🚪 Page unloading, ensuring WebSocket cleanup...");
    };
    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      socketDebugCounters.componentUnmounts++;
      console.log("🔵 LiwordsSocket UNMOUNTING", {
        mountCount: socketDebugCounters.componentMounts,
        unmountCount: socketDebugCounters.componentUnmounts,
        timestamp: new Date().toISOString(),
      });
      isMountedRef.current = false;
      window.removeEventListener("beforeunload", handleBeforeUnload);
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
    socketDebugCounters.tokenFetchAttempts++;
    console.log("About to request token", {
      isMounted: isMountedRef.current,
      pathname,
      timestamp: new Date().toISOString(),
      tokenFetchAttempt: socketDebugCounters.tokenFetchAttempts,
    });
    // Empty string will cause issues but react-use-websocket types don't support null for async functions
    const failUrl = "";
    try {
      const resp = await authClient.getSocketToken({});

      // Important: resetSocket does not resetStore, be very careful to avoid
      // dispatching stuffs from a decommissioned socket after client returns.
      if (!isMountedRef.current) {
        socketDebugCounters.failUrlReturns++;
        console.error(
          "🚨 SOCKET BUG: Component unmounted during token fetch, returning null (no connection)!",
          {
            pathname,
            totalFailUrlReturns: socketDebugCounters.failUrlReturns,
            allCounters: socketDebugCounters,
          },
        );
        return failUrl;
      }

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
      socketDebugCounters.tokenFetchSuccesses++;
      console.log("🎯 Got token, returning WebSocket URL to library...", {
        socketUrl: ret.substring(0, 80) + "...",
        baseSocketUrl: socketUrl,
        protocol: ret.split("://")[0],
        host: ret.split("//")[1]?.split("/")[0],
        fullUrlLength: ret.length,
        timestamp: new Date().toISOString(),
        tokenFetchSuccess: socketDebugCounters.tokenFetchSuccesses,
      });
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
      socketDebugCounters.failUrlReturns++;
      console.error(
        "🚨 SOCKET BUG: Error fetching socket token, returning null (no connection)!",
        {
          pathname,
          error: e,
          totalFailUrlReturns: socketDebugCounters.failUrlReturns,
          allCounters: socketDebugCounters,
        },
      );
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
      socketDebugCounters.patienceTimeouts++;
      console.warn(
        "⏱️ PATIENCE TIMEOUT: No connection after 15s, forcing socket reset",
        {
          patienceId,
          isConnectedToSocket,
          totalTimeouts: socketDebugCounters.patienceTimeouts,
          allCounters: socketDebugCounters,
        },
      );
      resetSocket();
    }, 15000);
    return () => {
      clearTimeout(t);
    };
  }, [patienceId, resetSocket, isConnectedToSocket]);

  const { sendMessage: originalSendMessage } = useWebSocket(
    getFullSocketUrlAsync,
    {
      onOpen: () => {
        socketDebugCounters.successfulConnections++;
        console.log("✅ WebSocket opened successfully", {
          totalSuccessfulConnections: socketDebugCounters.successfulConnections,
          allCounters: socketDebugCounters,
        });
        resetPatience();
        setIsConnectedToSocket(true);
      },
      onClose: (event) => {
        console.log("🔴 WebSocket closed", {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean,
          allCounters: socketDebugCounters,
        });
        resetPatience();
        setIsConnectedToSocket(false);
      },
      onError: (event) => {
        socketDebugCounters.wsErrors++;
        console.error("❌ WebSocket error occurred", {
          event,
          totalWsErrors: socketDebugCounters.wsErrors,
          allCounters: socketDebugCounters,
        });
      },
      reconnectAttempts: Infinity,
      reconnectInterval: 1000,
      retryOnError: true,
      shouldReconnect: (closeEvent) => {
        const shouldReconn = isMountedRef.current;
        if (shouldReconn) {
          socketDebugCounters.reconnectAttempts++;
        }
        console.log("🔄 shouldReconnect?", {
          isMounted: isMountedRef.current,
          decision: shouldReconn,
          closeCode: closeEvent?.code,
          totalReconnectAttempts: socketDebugCounters.reconnectAttempts,
          allCounters: socketDebugCounters,
        });
        return shouldReconn;
      },
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
