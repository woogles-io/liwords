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
