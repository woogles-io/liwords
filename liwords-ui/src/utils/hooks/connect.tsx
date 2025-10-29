import { useMemo } from "react";

import { type DescService } from "@bufbuild/protobuf";
import { message } from "antd";
import { parseWooglesError } from "../parse_woogles_error";
import { createConnectTransport } from "@connectrpc/connect-web";
import { ConnectError, Client, createClient } from "@connectrpc/connect";

const loc = window.location;
const apiEndpoint = window.RUNTIME_CONFIGURATION?.apiEndpoint || loc.host;

export const baseURL = `${loc.protocol}//${apiEndpoint}`;

export const transport = createConnectTransport({
  baseUrl: `${baseURL}/api/`,
  //   interceptors: [errorTranslator],
  fetch: (input, init) => fetch(input, { ...init, credentials: "include" }),
});

export const binaryTransport = createConnectTransport({
  baseUrl: `${loc.protocol}//${apiEndpoint}/api/`,
  useBinaryFormat: true,
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
