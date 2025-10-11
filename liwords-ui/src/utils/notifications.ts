// Notification utilities for PWA turn notifications

const NOTIFICATION_PERMISSION_KEY = "turnNotificationsEnabled";

export const isPWA = (): boolean => {
  // Check if the app is running as a PWA (installed)
  return (
    window.matchMedia("(display-mode: standalone)").matches ||
    // @ts-expect-error - iOS Safari
    window.navigator.standalone === true
  );
};

export const getTurnNotificationPreference = (): boolean => {
  const stored = localStorage.getItem(NOTIFICATION_PERMISSION_KEY);
  return stored === "true";
};

export const setTurnNotificationPreference = (enabled: boolean): void => {
  localStorage.setItem(NOTIFICATION_PERMISSION_KEY, enabled.toString());
};

export const getNotificationPermissionState = ():
  | "granted"
  | "denied"
  | "default"
  | "unsupported" => {
  if (!("Notification" in window)) {
    return "unsupported";
  }
  return Notification.permission;
};

export const requestNotificationPermission = async (): Promise<boolean> => {
  if (!("Notification" in window)) {
    console.warn("This browser does not support notifications");
    return false;
  }

  if (Notification.permission === "granted") {
    return true;
  }

  if (Notification.permission === "denied") {
    return false;
  }

  // Request permission
  const permission = await Notification.requestPermission();
  return permission === "granted";
};

export const canShowNotifications = (): boolean => {
  return (
    "Notification" in window &&
    Notification.permission === "granted" &&
    getTurnNotificationPreference()
  );
};

export interface TurnNotificationOptions {
  opponentName: string;
  timeControl?: string; // e.g., "Rapid", "5 days/turn"
  gameId?: string;
}

export const showTurnNotification = (
  options: TurnNotificationOptions,
): Notification | null => {
  if (!canShowNotifications()) {
    return null;
  }

  // Don't show if the window/tab is currently focused
  if (document.hasFocus()) {
    return null;
  }

  const { opponentName, timeControl } = options;
  const title = `Your turn vs ${opponentName}`;
  let body = "It's your turn to play";

  if (timeControl) {
    body += ` - ${timeControl}`;
  }

  try {
    const notification = new Notification(title, {
      body,
      icon: "/logo192.png",
      badge: "/favicon.ico",
      tag: "woogles-turn", // Replaces previous notification
      requireInteraction: false, // Auto-dismiss after a few seconds
      silent: false,
    });

    // Focus the window when notification is clicked
    notification.onclick = () => {
      window.focus();
      notification.close();
    };

    return notification;
  } catch (error) {
    console.error("[Notifications] Error creating notification:", error);
    return null;
  }
};
