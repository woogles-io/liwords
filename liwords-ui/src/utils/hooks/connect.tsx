import { useMemo } from 'react';

import { ServiceType } from '@bufbuild/protobuf';
import { message } from 'antd';
import { parseWooglesError } from '../parse_woogles_error';
import { createConnectTransport } from '@connectrpc/connect-web';
import {
  ConnectError,
  PromiseClient,
  createPromiseClient,
} from '@connectrpc/connect';

const loc = window.location;
const apiEndpoint = window.RUNTIME_CONFIGURATION?.apiEndpoint || loc.host;

export const baseURL = `${loc.protocol}//${apiEndpoint}`;

export const transport = createConnectTransport({
  baseUrl: `${baseURL}/api/`,
  //   interceptors: [errorTranslator],
});

export const binaryTransport = createConnectTransport({
  baseUrl: `${loc.protocol}//${apiEndpoint}/api/`,
  useBinaryFormat: true,
});

export function useClient<T extends ServiceType>(
  service: T,
  binary = false
): PromiseClient<T> {
  const tf = binary ? binaryTransport : transport;
  return useMemo(() => createPromiseClient(service, tf), [service, tf]);
}

export const flashError = (e: unknown, time = 5) => {
  if (e instanceof ConnectError) {
    message.error({
      content: parseWooglesError(e.rawMessage),
      duration: time,
    });
  } else {
    message.error({
      content: 'Unknown error; see console',
      duration: time,
    });
    console.error(e);
  }
};

export const connectErrorMessage = (e: unknown) => {
  return (e as ConnectError).rawMessage;
};
