import { useEffect, useRef } from "react";
import { useLocation } from "react-router";
import { create, toBinary } from "@bufbuild/protobuf";
import {
  SubscribeRequestSchema,
  UnsubscribeRequestSchema,
  MessageType,
} from "../gen/api/proto/ipc/ipc_pb";
import { encodeToSocketFmt } from "../utils/protobuf";

/**
 * Hook for managing dynamic subscriptions in protocol v2.
 * Sends Subscribe/Unsubscribe messages when navigating between pages.
 *
 * @param sendMessage Function to send messages to the socket
 * @param isConnected Whether the socket is connected
 * @param handshakeComplete Whether the v2 handshake has completed
 */
export const useSubscription = (
  sendMessage: (msg: Uint8Array) => void,
  isConnected: boolean,
  handshakeComplete: boolean
) => {
  const location = useLocation();
  const currentPath = useRef<string | null>(null);
  const subscribedThisConnection = useRef(false);

  // Reset subscription tracking on disconnect
  useEffect(() => {
    if (!isConnected) {
      subscribedThisConnection.current = false;
      // Keep currentPath so we know what to re-subscribe to on reconnect
    }
  }, [isConnected]);

  useEffect(() => {
    if (!isConnected || !handshakeComplete) return;

    const newPath = location.pathname;
    const needsResubscribe = !subscribedThisConnection.current;
    const pathChanged = currentPath.current !== newPath;

    if (needsResubscribe || pathChanged) {
      // Unsubscribe from old path (only on navigation, not on reconnect)
      if (!needsResubscribe && currentPath.current) {
        const unsub = create(UnsubscribeRequestSchema, {
          path: currentPath.current,
        });
        sendMessage(
          encodeToSocketFmt(
            MessageType.UNSUBSCRIBE_REQUEST,
            toBinary(UnsubscribeRequestSchema, unsub)
          )
        );
        console.log("Unsubscribed from:", currentPath.current);
      }

      // Subscribe to current path
      const sub = create(SubscribeRequestSchema, { path: newPath });
      sendMessage(
        encodeToSocketFmt(
          MessageType.SUBSCRIBE_REQUEST,
          toBinary(SubscribeRequestSchema, sub)
        )
      );
      console.log("Subscribed to:", newPath);

      currentPath.current = newPath;
      subscribedThisConnection.current = true;
    }
  }, [location.pathname, sendMessage, isConnected, handshakeComplete]);
};
