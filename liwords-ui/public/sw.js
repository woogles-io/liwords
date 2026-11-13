// Service Worker for Woogles PWA
// Handles notification clicks to focus/open the app
// Shows the correspondence turns the server pushes (pkg/push)

self.addEventListener("install", (event) => {
  console.log("Service Worker installing...");
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  console.log("Service Worker activating...");
  event.waitUntil(clients.claim());
});

// The deadline (unix ms) as the device's own date and time, e.g.
// "Nov 14, 2:30 PM PST", so it is still right whenever it is read; the zone
// tells apart the two runs of an hour repeated when daylight saving ends.
const moveBy = (deadline) =>
  new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(new Date(deadline));

// A pushed turn: { gameId, title, body, deadline }. It is always shown, even
// while that game is open: a push that shows nothing can make the browser
// show a notice of its own.
self.addEventListener("push", (event) => {
  let turn = null;
  try {
    turn = event.data ? event.data.json() : null;
  } catch (e) {
    console.log("A push without a readable turn:", e);
  }
  if (!turn || !turn.gameId) {
    return;
  }
  const body = turn.deadline
    ? `${turn.body} - move by ${moveBy(turn.deadline)}`
    : turn.body;
  event.waitUntil(
    self.registration.showNotification(turn.title, {
      body,
      icon: "/logo192.png",
      badge: "/favicon.ico",
      tag: `woogles-turn-${turn.gameId}`,
      data: { gameId: turn.gameId },
    }),
  );
});

// Handle notification clicks
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const gameId = event.notification.data && event.notification.data.gameId;

  event.waitUntil(
    clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clientList) => {
        // A pushed turn opens its game, in the window already showing it if
        // there is one.
        if (gameId) {
          const path = `/game/${encodeURIComponent(gameId)}`;
          for (const client of clientList) {
            if (new URL(client.url).pathname === path && "focus" in client) {
              return client.focus();
            }
          }
          if (clients.openWindow) {
            return clients.openWindow(path);
          }
          return undefined;
        }
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

// Sound effects live under /static/media/ with content-hashed names, so a
// stored copy never goes stale. <audio> asks for them with Range requests,
// which WebKit does not answer from its HTTP cache, so every page load
// downloaded every sound again (#2022). Those requests are answered from a
// copy in Cache Storage instead; the first one downloads the whole file
// once. If anything goes wrong, the request goes to the network as before.
const MEDIA_CACHE = "woogles-media-v1";
const MEDIA_PATH = "/static/media/";

// Downloads under way, by URL, so the requests a page makes for one sound
// at the same time share a single download.
const downloads = new Map();

const storedMedia = async (cache, url) => {
  const stored = await cache.match(url);
  if (stored) {
    return stored;
  }
  let download = downloads.get(url);
  if (!download) {
    download = (async () => {
      const response = await fetch(url);
      if (response.status !== 200) {
        throw new Error(`${url}: status ${response.status}`);
      }
      await cache.put(url, response);
    })().finally(() => downloads.delete(url));
    downloads.set(url, download);
  }
  await download;
  return cache.match(url);
};

// The part of a full response that a Range header asks for, or null if the
// header is not a single byte range.
const partialResponse = async (full, range) => {
  const match = /^bytes=(\d*)-(\d*)$/.exec(range.trim());
  if (!match || (match[1] === "" && match[2] === "")) {
    return null;
  }
  const body = await full.arrayBuffer();
  const size = body.byteLength;
  let start;
  let end;
  if (match[1] === "") {
    // "bytes=-n" asks for the last n bytes.
    start = Math.max(size - Number(match[2]), 0);
    end = size - 1;
  } else {
    start = Number(match[1]);
    end = match[2] === "" ? size - 1 : Math.min(Number(match[2]), size - 1);
  }
  const headers = new Headers({ "Accept-Ranges": "bytes" });
  const contentType = full.headers.get("Content-Type");
  if (contentType) {
    headers.set("Content-Type", contentType);
  }
  if (start > end) {
    headers.set("Content-Range", `bytes */${size}`);
    return new Response(null, {
      status: 416,
      statusText: "Range Not Satisfiable",
      headers,
    });
  }
  headers.set("Content-Range", `bytes ${start}-${end}/${size}`);
  headers.set("Content-Length", String(end - start + 1));
  return new Response(body.slice(start, end + 1), {
    status: 206,
    statusText: "Partial Content",
    headers,
  });
};

const serveMedia = async (request) => {
  try {
    const cache = await caches.open(MEDIA_CACHE);
    const full = await storedMedia(cache, request.url);
    if (full) {
      const range = request.headers.get("Range");
      if (!range) {
        return full;
      }
      const partial = await partialResponse(full, range);
      if (partial) {
        return partial;
      }
    }
  } catch (e) {
    console.log("Serving a sound from Cache Storage failed:", e);
  }
  return fetch(request);
};

// Every other request is left alone, so the browser handles it normally.
// (The PWA needs a fetch listener to exist either way.)
self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }
  const url = new URL(request.url);
  if (
    url.origin !== self.location.origin ||
    !url.pathname.startsWith(MEDIA_PATH)
  ) {
    return;
  }
  event.respondWith(serveMedia(request));
});
