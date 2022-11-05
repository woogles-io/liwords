import { useMemo } from 'react';
import {
  ConnectError,
  createConnectTransport,
  createPromiseClient,
  PromiseClient,
  Interceptor,
} from '@bufbuild/connect-web';

import { ServiceType } from '@bufbuild/protobuf';
import { message } from 'antd';
import { parseWooglesError } from '../parse_woogles_error';

const loc = window.location;
const apiEndpoint = window.RUNTIME_CONFIGURATION?.apiEndpoint || loc.host;

// const errorTranslator: Interceptor = (next) => async (req) => {
//   try {
//     const res = await next(req);
//     console.log('in interceptor', res);
//     return res;
//   } catch (e) {
//     console.log('in interceptor, caught', e);
//     throw e;
//   }
// };

const transport = createConnectTransport({
  baseUrl: `${loc.protocol}//${apiEndpoint}/twirp/`,
  //   interceptors: [errorTranslator],
});

export function useClient<T extends ServiceType>(service: T): PromiseClient<T> {
  return useMemo(() => createPromiseClient(service, transport), [service]);
}

export type TwirpError = {
  response: { data: { msg: string } };
};

export const flashError = (e: unknown, time = 5) => {
  if (e instanceof ConnectError) {
    message.error({
      content: parseWooglesError(e.message),
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
