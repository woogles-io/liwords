// Service Worker for Woogles PWA
// Handles notification clicks to focus/open the app

self.addEventListener("install", (event) => {
  console.log("Service Worker installing...");
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  console.log("Service Worker activating...");
  event.waitUntil(clients.claim());
});

// Handle notification clicks
self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  event.waitUntil(
    clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clientList) => {
        // Try to focus an existing window
        for (const client of clientList) {
          if ("focus" in client) {
            return client.focus();
          }
        }
        // If no window exists, open a new one
        if (clients.openWindow) {
          return clients.openWindow("/");
        }
      }),
  );
});

// Minimal fetch handler - just let all requests pass through normally
// We don't need to intercept requests since we're not doing offline caching
self.addEventListener("fetch", (event) => {
  // Do nothing - let the browser handle all fetches normally
  // This is valid for PWA, we just need the event listener to exist
});
