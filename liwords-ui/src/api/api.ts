import { message } from 'antd';
import { parseWooglesError } from '../utils/parse_woogles_error';

export const toAPIUrl = (service: string, method: string) => {
  const loc = window.location;
  const apiEndpoint = window.RUNTIME_CONFIGURATION.apiEndpoint || loc.host;

  // Assuming we don't need to encodeURIComponent() here...
  return `${loc.protocol}//${apiEndpoint}/twirp/${service}/${method}`;
};

interface PBMsg {
  serializeBinary(): Uint8Array;
}

export const postJsonObj = async (
  service: string,
  method: string,
  msg: any,
  successHandler?: (res: any) => void,
  errHandler?: (err: any) => void
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
        content: e?.message,
        duration: 8,
      });
    } else {
      errHandler(e);
    }
  }
};

interface JsonError {
  code: string;
  msg: string;
}

const postBinary = async (service: string, method: string, msg: PBMsg) => {
  const url = toAPIUrl(service, method);

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/protobuf',
    },
    body: msg.serializeBinary(),
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
  responseType: { deserializeBinary(x: Uint8Array): T },
  service: string,
  method: string,
  msg: { serializeBinary(): Uint8Array }
) => Promise<T> = async (responseType, service, method, msg) =>
  responseType.deserializeBinary(
    new Uint8Array(await postBinary(service, method, msg))
  );
