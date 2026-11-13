import { useEffect } from "react";
import { PushService } from "../gen/api/proto/push_service/push_service_pb";
import { useClient } from "../utils/hooks/connect";
import { getTurnNotificationPreference } from "../utils/notifications";
import {
  PushClient,
  subscribeThisDevice,
  unsubscribeThisDevice,
  webPushSupported,
} from "../utils/web_push";

type PushSyncClient = PushClient & {
  getPushConfig: (req: object) => Promise<{
    enabled: boolean;
    publicKey: string;
  }>;
};

// The notifications switch in preferences is the one place players allow
// notifications on a device. While it is on and the browser has granted
// permission, the browser is also subscribed to web push when the server has
// it, so turns arrive with Woogles closed; sending the subscription moves it
// to whoever is signed in. While it is off, the browser is unsubscribed.
// Nothing here asks for permission: only the switch does.
export const syncWebPush = async (
  client: PushSyncClient,
  on: boolean,
): Promise<void> => {
  if (!webPushSupported()) {
    return;
  }
  if (!on || Notification.permission !== "granted") {
    await unsubscribeThisDevice(client);
    return;
  }
  const config = await client.getPushConfig({});
  if (!config.enabled || !config.publicKey) {
    return;
  }
  await subscribeThisDevice(client, config.publicKey);
};

// Keeps this browser's web push in line with the notifications switch on
// every page, for whoever is signed in: a browser shared without a logout
// then gets only the current player's turns, and a player who turned the
// switch on before web push existed gets it without being asked again.
export const useWebPushSync = (userID: string | undefined) => {
  const client = useClient(PushService);
  useEffect(() => {
    if (!userID) {
      return;
    }
    syncWebPush(client, getTurnNotificationPreference()).catch((e) =>
      console.log("Updating this device's web push failed:", e),
    );
  }, [client, userID]);
};
