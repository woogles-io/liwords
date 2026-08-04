import { useMemo } from "react";

import { type DescService } from "@bufbuild/protobuf";
import { message } from "antd";
import { parseWooglesError } from "../parse_woogles_error";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ConnectError, Client, createClient } from "@connectrpc/connect";

const loc = window.location;
const apiEndpoint = window.RUNTIME_CONFIGURATION?.apiEndpoint || loc.host;

export const baseURL = `${loc.protocol}//${apiEndpoint}`;

// useHttpGet makes the transport issue GET instead of POST for methods marked
// `option idempotency_level = NO_SIDE_EFFECTS` in the protos. Without it the
// annotation is inert: connect-web requires both
// (see connect-transport.js -- `options.useHttpGet === true && method.idempotency === NO_SIDE_EFFECTS`).
//
// GET makes those responses cacheable by CloudFront, which cannot cache POST at
// all. Note the CDN still has to opt in per behaviour -- /api/* currently uses
// the managed CachingDisabled policy, so this change enables caching rather
// than performing it.
//
// Caveat: a GET request body is base64'd into the query string, and CloudFront
// rejects URLs over ~8192 bytes. GetBriefProfiles is the one to watch, since it
// batches every user id rendered in a tick; largest observed body is ~3.7KB
// (~4.9KB base64'd). connect-web does not fall back to POST when the URL is too
// long, so a much larger batch would 414 rather than degrade.
export const transport = createConnectTransport({
  baseUrl: `${baseURL}/api/`,
  useHttpGet: true,
  //   interceptors: [errorTranslator],
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export const binaryTransport = createConnectTransport({
  baseUrl: `${loc.protocol}//${apiEndpoint}/api/`,
  useBinaryFormat: true,
  useHttpGet: true,
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export function useClient<T extends DescService>(
  service: T,
  binary = false,
): Client<T> {
  const tf = binary ? binaryTransport : transport;
  return useMemo(() => createClient(service, tf), [service, tf]);
}

export const flashError = (e: unknown, time = 5) => {
  if (e instanceof ConnectError) {
    message.error({
      content: parseWooglesError(e.rawMessage),
      duration: time,
    });
  } else {
    message.error({
      content: "Unknown error; see console",
      duration: time,
    });
    console.error(e);
  }
};

export const connectErrorMessage = (e: unknown) => {
  return (e as ConnectError).rawMessage;
};
