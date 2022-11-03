import { useMemo } from 'react';
import {
  createConnectTransport,
  createPromiseClient,
  PromiseClient,
} from '@bufbuild/connect-web';
import { ServiceType } from '@bufbuild/protobuf';
const loc = window.location;
const apiEndpoint = window.RUNTIME_CONFIGURATION.apiEndpoint || loc.host;

const transport = createConnectTransport({
  baseUrl: `${loc.protocol}//${apiEndpoint}/twirp/`,
});

export function useClient<T extends ServiceType>(service: T): PromiseClient<T> {
  return useMemo(() => createPromiseClient(service, transport), [service]);
}
