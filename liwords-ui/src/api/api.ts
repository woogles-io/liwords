import { message } from 'antd';
import { parseWooglesError } from '../utils/parse_woogles_error';

export const toAPIUrl = (service: string, method: string) => {
  const loc = window.location;
  const apiEndpoint = window.RUNTIME_CONFIGURATION.apiEndpoint || loc.host;

  // Assuming we don't need to encodeURIComponent() here...
  return `${loc.protocol}//${apiEndpoint}/twirp/${service}/${method}`;
};

interface PBMsg {
  toBinary(): Uint8Array;
}

export type LiwordsAPIError = {
  message: string;
};

export const postJsonObj = async (
  service: string,
  method: string,
  msg: unknown,
  successHandler?: (res: unknown) => void,
  errHandler?: (err: LiwordsAPIError) => void
) => {
  const url = toAPIUrl(service, method);

  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(msg),
    });
    const json = await response.json();
    if (!response.ok) {
      // non-200 response
      const msg = parseWooglesError(json.msg);
      throw new Error(msg);
    }
    if (successHandler) {
      successHandler(json);
    }
  } catch (e) {
    if (!errHandler) {
      message.error({
        content: (e as LiwordsAPIError).message,
        duration: 8,
      });
    } else {
      errHandler(e as LiwordsAPIError);
    }
  }
};

const postBinary = async (service: string, method: string, msg: PBMsg) => {
  const url = toAPIUrl(service, method);

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/protobuf',
    },
    body: msg.toBinary(),
  });

  if (!response.ok) {
    // non-200 response.
    // because Twirp always returns errors as JSON, no matter what the
    // content-type, we must parse this differently.
    const errJSON = await response.json();
    const msg = parseWooglesError(errJSON.msg);
    throw new Error(msg);
  } else {
    const ab = await response.arrayBuffer();
    return ab;
  }
};

export const postProto: <T>(
  responseType: { fromBinary(x: Uint8Array): T },
  service: string,
  method: string,
  msg: { toBinary(): Uint8Array }
) => Promise<T> = async (responseType, service, method, msg) =>
  responseType.fromBinary(
    new Uint8Array(await postBinary(service, method, msg))
  );
