// CloudFront viewer-request function for page loads (/game/*, /profile/*,
// /anno/*).
//
// - Crawlers that ignore robots.txt get a 403.
// - Cookieless requests on www. are sent straight to the naked domain
//   (they have no session to hand over, so the in-app handover would only
//   load the whole app twice).
// - Desktop browser page loads need a valid wg_ok cookie. Without one they
//   get a small page with a Cloudflare Turnstile check; passing it calls
//   /api/human-check, which sets the cookie, and the page reloads. Phones,
//   tablets and self-identified crawlers are not gated.
//
// wg_ok is "<unix expiry>.<hex HMAC-SHA256(expiry)>", signed by the API with
// the same key as below (HUMAN_CHECK_COOKIE_KEY).

import crypto from "crypto";

var COOKIE_KEY = "__HUMAN_CHECK_COOKIE_KEY__";
var COOKIE_NAME = "wg_ok";
var SITE_KEY = "0x4AAAAAAFgCaIVA5vNeRk19";

var BLOCKED_UA = /Sogou/i;
var DESKTOP_UA = /\((Windows NT|Macintosh|X11|CrOS)/;
var UNGATED_UA = /Mobile|Android|iPhone|iPad|bot|crawl|spider|slurp|facebookexternalhit|meta-external|preview/i;

var GATE_PAGE =
  '<!doctype html><html lang="en"><head><meta charset="utf-8">' +
  '<meta name="viewport" content="width=device-width,initial-scale=1">' +
  '<meta name="robots" content="noindex,nofollow">' +
  "<title>Woogles.io</title>" +
  '<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>' +
  "<style>" +
  "body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;" +
  "font-family:system-ui,sans-serif;background:#f4f4f4;color:#222}" +
  "main{text-align:center;padding:24px}" +
  "@media (prefers-color-scheme:dark){body{background:#1d1d1d;color:#eee}}" +
  "</style></head><body><main>" +
  "<h1>Woogles.io</h1><p id=\"msg\">Checking your browser before continuing.</p>" +
  '<div class="cf-turnstile" data-sitekey="' + SITE_KEY + '" data-callback="wgPassed" data-error-callback="wgError"></div>' +
  "<noscript><p>Please enable JavaScript.</p></noscript>" +
  "</main><script>" +
  "function wgFail(){" +
  'document.getElementById("msg").textContent="Verification failed. Please reload the page.";}' +
  "function wgError(){wgFail();}" +
  "function wgPassed(token){" +
  'fetch("/api/human-check",{method:"POST",headers:{"Content-Type":"application/json"},' +
  "body:JSON.stringify({token:token})}).then(function(r){" +
  "if(r.ok){location.reload();}else{wgFail();}" +
  "},wgFail);}" +
  "</script></body></html>";

function queryString(qs) {
  var parts = [];
  for (var key in qs) {
    var entry = qs[key];
    var values = entry.multiValue ? entry.multiValue : [entry];
    for (var i = 0; i < values.length; i++) {
      parts.push(key + "=" + values[i].value);
    }
  }
  return parts.length ? "?" + parts.join("&") : "";
}

function hasValidPass(cookies) {
  var cookie = cookies[COOKIE_NAME];
  if (!cookie) {
    return false;
  }
  var parts = cookie.value.split(".");
  if (parts.length !== 2) {
    return false;
  }
  var expiry = parseInt(parts[0], 10);
  if (!(expiry > Date.now() / 1000)) {
    return false;
  }
  var expected = crypto.createHmac("sha256", COOKIE_KEY).update(parts[0]).digest("hex");
  return expected === parts[1];
}

function handler(event) {
  var request = event.request;
  var headers = request.headers;
  var ua = headers["user-agent"] ? headers["user-agent"].value : "";

  if (BLOCKED_UA.test(ua)) {
    return { statusCode: 403, statusDescription: "Forbidden" };
  }

  var host = headers.host ? headers.host.value : "";
  if (host.indexOf("www.") === 0 && Object.keys(request.cookies).length === 0) {
    return {
      statusCode: 302,
      statusDescription: "Found",
      headers: {
        location: {
          value:
            "https://" + host.slice(4) + request.uri + queryString(request.querystring),
        },
        "cache-control": { value: "no-store" },
      },
    };
  }

  if (DESKTOP_UA.test(ua) && !UNGATED_UA.test(ua) && !hasValidPass(request.cookies)) {
    return {
      statusCode: 200,
      statusDescription: "OK",
      headers: {
        "content-type": { value: "text/html; charset=utf-8" },
        "cache-control": { value: "no-store" },
      },
      body: GATE_PAGE,
    };
  }

  return request;
}
