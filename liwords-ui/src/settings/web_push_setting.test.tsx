import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { syncWebPush, useWebPushSync } from "./web_push_setting";
import { keyBytes, logOutForgettingDevice } from "../utils/web_push";

// A public key in base64url, as the server hands it out.
const PUBLIC_KEY =
  "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM";

const pushClient = {
  getPushConfig: vi.fn(),
  subscribe: vi.fn(),
  unsubscribe: vi.fn(),
};

vi.mock("../utils/hooks/connect", async () => {
  const mod = await vi.importActual("../utils/hooks/connect");
  return { ...mod, useClient: () => pushClient, flashError: vi.fn() };
});

type FakeSubscription = {
  endpoint: string;
  options: { applicationServerKey: ArrayBuffer | null };
  toJSON: () => { keys: { p256dh: string; auth: string } };
  unsubscribe: ReturnType<typeof vi.fn>;
};

let current: FakeSubscription | null;
let permission: NotificationPermission;
let requestedPermission: NotificationPermission;
let pushManager: {
  getSubscription: ReturnType<typeof vi.fn>;
  subscribe: ReturnType<typeof vi.fn>;
};

const fakeSubscription = (
  endpoint: string,
  key: Uint8Array | null,
): FakeSubscription => {
  const sub: FakeSubscription = {
    endpoint,
    options: {
      applicationServerKey: key ? (key.slice().buffer as ArrayBuffer) : null,
    },
    toJSON: () => ({ keys: { p256dh: "the-p256dh", auth: "the-auth" } }),
    unsubscribe: vi.fn(async () => {
      if (current === sub) current = null;
      return true;
    }),
  };
  return sub;
};

const stubBrowser = (opts: { push?: boolean; registered?: boolean } = {}) => {
  const { push = true, registered = true } = opts;
  pushManager = {
    getSubscription: vi.fn(async () => current),
    subscribe: vi.fn(
      async (o: { applicationServerKey: Uint8Array }) =>
        (current = fakeSubscription(
          "https://push.example.com/new",
          o.applicationServerKey,
        )),
    ),
  };
  vi.stubGlobal("Notification", {
    get permission() {
      return permission;
    },
    requestPermission: vi.fn(async () => (permission = requestedPermission)),
  });
  if (push) {
    vi.stubGlobal("PushManager", function PushManager() {});
  }
  Object.defineProperty(navigator, "serviceWorker", {
    configurable: true,
    value: {
      getRegistration: vi.fn(async () =>
        registered ? { pushManager } : undefined,
      ),
    },
  });
};

beforeEach(() => {
  current = null;
  permission = "default";
  requestedPermission = "granted";
  pushClient.getPushConfig.mockResolvedValue({
    enabled: true,
    publicKey: PUBLIC_KEY,
  });
  pushClient.subscribe.mockResolvedValue({});
  pushClient.unsubscribe.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.clearAllMocks();
  Reflect.deleteProperty(navigator, "serviceWorker");
});

describe("syncWebPush", () => {
  it("with the switch on and permission granted, subscribes and tells the server, asking nothing", async () => {
    stubBrowser();
    permission = "granted";
    await syncWebPush(pushClient, true);
    expect(Notification.requestPermission).not.toHaveBeenCalled();
    expect(pushManager.subscribe).toHaveBeenCalledWith({
      userVisibleOnly: true,
      applicationServerKey: keyBytes(PUBLIC_KEY),
    });
    expect(pushClient.subscribe).toHaveBeenCalledWith({
      endpoint: "https://push.example.com/new",
      p256dh: "the-p256dh",
      auth: "the-auth",
    });
  });

  it("sends an existing subscription again, so it moves to this user", async () => {
    stubBrowser();
    permission = "granted";
    current = fakeSubscription(
      "https://push.example.com/left",
      keyBytes(PUBLIC_KEY),
    );
    await syncWebPush(pushClient, true);
    expect(pushManager.subscribe).not.toHaveBeenCalled();
    expect(pushClient.subscribe).toHaveBeenCalledWith(
      expect.objectContaining({ endpoint: "https://push.example.com/left" }),
    );
  });

  it("replaces a subscription made with another key", async () => {
    stubBrowser();
    permission = "granted";
    current = fakeSubscription(
      "https://push.example.com/stale",
      new Uint8Array([1, 2, 3]),
    );
    const stale = current;
    await syncWebPush(pushClient, true);
    expect(stale.unsubscribe).toHaveBeenCalledTimes(1);
    expect(pushClient.subscribe).toHaveBeenLastCalledWith(
      expect.objectContaining({ endpoint: "https://push.example.com/new" }),
    );
  });

  it("with the switch off, tells the server and unsubscribes the browser", async () => {
    stubBrowser();
    permission = "granted";
    current = fakeSubscription(
      "https://push.example.com/old",
      keyBytes(PUBLIC_KEY),
    );
    const old = current;
    await syncWebPush(pushClient, false);
    expect(pushClient.unsubscribe).toHaveBeenCalledWith({
      endpoint: "https://push.example.com/old",
    });
    expect(old.unsubscribe).toHaveBeenCalledTimes(1);
    expect(pushClient.subscribe).not.toHaveBeenCalled();
  });

  it("without permission, subscribes nothing and asks nothing", async () => {
    stubBrowser();
    permission = "default";
    await syncWebPush(pushClient, true);
    expect(Notification.requestPermission).not.toHaveBeenCalled();
    expect(pushManager.subscribe).not.toHaveBeenCalled();
    expect(pushClient.subscribe).not.toHaveBeenCalled();
  });

  it("does nothing more when the server has no push", async () => {
    stubBrowser();
    permission = "granted";
    pushClient.getPushConfig.mockResolvedValue({
      enabled: false,
      publicKey: "",
    });
    await syncWebPush(pushClient, true);
    expect(pushManager.subscribe).not.toHaveBeenCalled();
    expect(pushClient.subscribe).not.toHaveBeenCalled();
  });

  it("does nothing when the browser lacks push", async () => {
    stubBrowser({ push: false });
    permission = "granted";
    await syncWebPush(pushClient, true);
    expect(pushClient.getPushConfig).not.toHaveBeenCalled();
  });

  it("logging out forgets this device first, then logs out", async () => {
    stubBrowser();
    current = fakeSubscription("https://push.example.com/old", null);
    const old = current;
    const order: string[] = [];
    pushClient.unsubscribe.mockImplementation(async () => order.push("server"));
    old.unsubscribe.mockImplementation(async () => order.push("browser"));
    await logOutForgettingDevice(pushClient, async () => order.push("logout"));
    expect(order).toEqual(["server", "browser", "logout"]);
    expect(pushClient.unsubscribe).toHaveBeenCalledWith({
      endpoint: "https://push.example.com/old",
    });
  });

  it("logging out still logs out if forgetting the device fails", async () => {
    stubBrowser();
    current = fakeSubscription("https://push.example.com/old", null);
    const old = current;
    pushClient.unsubscribe.mockRejectedValue(new Error("down"));
    const logout = vi.fn(async () => ({}));
    await logOutForgettingDevice(pushClient, logout);
    expect(old.unsubscribe).toHaveBeenCalledTimes(1);
    expect(logout).toHaveBeenCalledTimes(1);
  });

  it("logging out without push support just logs out", async () => {
    stubBrowser({ push: false });
    const logout = vi.fn(async () => ({}));
    await logOutForgettingDevice(pushClient, logout);
    expect(logout).toHaveBeenCalledTimes(1);
    expect(pushClient.unsubscribe).not.toHaveBeenCalled();
  });
});

describe("useWebPushSync", () => {
  const Sync = (props: { userID?: string }) => {
    useWebPushSync(props.userID);
    return null;
  };

  it("subscribes a signed-in player whose switch was already on", async () => {
    stubBrowser();
    permission = "granted";
    localStorage.setItem("turnNotificationsEnabled", "true");
    render(<Sync userID="userA" />);
    await waitFor(() => expect(pushClient.subscribe).toHaveBeenCalledTimes(1));
    expect(Notification.requestPermission).not.toHaveBeenCalled();
    localStorage.clear();
  });

  it("moves the subscription when another player signs in", async () => {
    stubBrowser();
    permission = "granted";
    localStorage.setItem("turnNotificationsEnabled", "true");
    const { rerender } = render(<Sync userID="userA" />);
    await waitFor(() => expect(pushClient.subscribe).toHaveBeenCalledTimes(1));
    rerender(<Sync userID="userB" />);
    await waitFor(() => expect(pushClient.subscribe).toHaveBeenCalledTimes(2));
    localStorage.clear();
  });

  it("does nothing when signed out", async () => {
    stubBrowser();
    permission = "granted";
    localStorage.setItem("turnNotificationsEnabled", "true");
    render(<Sync />);
    await new Promise((r) => setTimeout(r, 20));
    expect(pushClient.getPushConfig).not.toHaveBeenCalled();
    localStorage.clear();
  });
});
