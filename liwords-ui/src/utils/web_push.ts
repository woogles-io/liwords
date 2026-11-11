// Web push: this device's subscription to correspondence turns, which the
// service worker (public/sw.js) shows with Woogles closed.

export type PushClient = {
  subscribe: (req: {
    endpoint: string;
    p256dh: string;
    auth: string;
  }) => Promise<unknown>;
  unsubscribe: (req: { endpoint: string }) => Promise<unknown>;
};

export const webPushSupported = (): boolean =>
  typeof navigator !== "undefined" &&
  "serviceWorker" in navigator &&
  typeof window !== "undefined" &&
  "PushManager" in window &&
  "Notification" in window;

// The VAPID public key arrives in base64url; PushManager wants its bytes.
export const keyBytes = (key: string): Uint8Array<ArrayBuffer> => {
  const base64 = key.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const raw = atob(padded);
  const bytes = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) {
    bytes[i] = raw.charCodeAt(i);
  }
  return bytes;
};

const sameKey = (a: ArrayBuffer | null | undefined, b: Uint8Array) => {
  if (!a || a.byteLength !== b.length) {
    return false;
  }
  const av = new Uint8Array(a);
  return av.every((x, i) => x === b[i]);
};

// The page's service worker registration, or undefined if there is none
// (it is registered on load, in index.tsx).
const registration = () => navigator.serviceWorker.getRegistration();

export const currentSubscription =
  async (): Promise<PushSubscription | null> => {
    const reg = await registration();
    return reg ? reg.pushManager.getSubscription() : null;
  };

export type SubscribeResult =
  | "subscribed"
  | "denied"
  | "dismissed"
  | "no-service-worker";

// Asks for notification permission, subscribes this browser with the
// server's key, and sends the subscription to the server.
export const subscribeThisDevice = async (
  client: PushClient,
  publicKey: string,
): Promise<SubscribeResult> => {
  const permission =
    Notification.permission === "granted"
      ? "granted"
      : await Notification.requestPermission();
  if (permission === "denied") {
    return "denied";
  }
  if (permission !== "granted") {
    return "dismissed";
  }
  const reg = await registration();
  if (!reg) {
    return "no-service-worker";
  }
  const key = keyBytes(publicKey);
  let sub = await reg.pushManager.getSubscription();
  // A subscription made with another key is useless to this server.
  if (sub && !sameKey(sub.options.applicationServerKey, key)) {
    await sub.unsubscribe();
    sub = null;
  }
  if (!sub) {
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: key,
    });
  }
  await sendSubscription(client, sub);
  return "subscribed";
};

const sendSubscription = async (client: PushClient, sub: PushSubscription) => {
  const json = sub.toJSON();
  await client.subscribe({
    endpoint: sub.endpoint,
    p256dh: json.keys?.p256dh ?? "",
    auth: json.keys?.auth ?? "",
  });
};

// Closes the turn notifications this browser shows for a game, once the
// player has the game open.
export const closeGameNotifications = async (gameId: string): Promise<void> => {
  if (!webPushSupported()) {
    return;
  }
  const reg = await registration();
  if (!reg) {
    return;
  }
  const shown = await reg.getNotifications({ tag: `woogles-turn-${gameId}` });
  for (const notification of shown) {
    notification.close();
  }
};

// Tells the server to forget this browser, then drops the browser's
// subscription even if the server could not be told: a push to a dropped
// subscription is refused, and the server then forgets it anyway.
export const unsubscribeThisDevice = async (
  client: PushClient,
): Promise<void> => {
  const sub = await currentSubscription();
  if (!sub) {
    return;
  }
  try {
    await client.unsubscribe({ endpoint: sub.endpoint });
  } finally {
    await sub.unsubscribe();
  }
};

// Logs out, first forgetting this browser's subscription while the session
// can still say whose it is, so the next person here gets no turns of the
// last one. A failure to forget never stops the logout.
export const logOutForgettingDevice = async (
  client: PushClient,
  logout: () => Promise<unknown>,
): Promise<void> => {
  if (webPushSupported()) {
    try {
      await unsubscribeThisDevice(client);
    } catch (e) {
      console.log("Forgetting this device's web push failed:", e);
    }
  }
  await logout();
};
