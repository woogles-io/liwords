import * as jspb from 'google-protobuf';

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

export const postBinary = async (
  service: string,
  method: string,
  msg: PBMsg,
  responseType: jspb.Message,
  successHandler?: (res: any) => void,
  errHandler?: (err: any) => void
) => {
  const url = toAPIUrl(service, method);

  try {
    const rbin = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/protobuf',
      },
      body: msg.serializeBinary(),
    });
    const ab = await rbin.arrayBuffer();
    const resp = Object.getPrototypeOf(
      responseType
    ).constructor.deserializeBinary(ab);
    if (!rbin.ok) {
      // XXX FIX ME THIS IS WEIRD
      // non-200 response
      const msg = parseWooglesError(resp);
      throw new Error(msg);
    }
    if (successHandler) {
      successHandler(msg);
    }
  } catch (e) {
    const unparsedErr = twirpErrToMsg(e);
    const msg = parseWooglesError(unparsedErr);

    if (!errHandler) {
      message.error({
        content: msg,
        duration: 8,
      });
    } else {
      errHandler(e);
    }
  }
};

export interface TwirpError {
  response: {
    data: Uint8Array;
  };
}

export const twirpErrToMsg = (err: TwirpError) => {
  // Twirp always returns JSON error messages no matter what. But since the
  // responseType is set to `arraybuffer` above it is annoying to deal with.
  // This function turns it into the JSON-encoded string that it is and
  // extracts the error message.
  if (!err.response || !err.response.data) {
    return 'non-twirp error: ' + String(err);
  }
  const errJSON = new TextDecoder().decode(err.response.data);
  return JSON.parse(errJSON).msg;
};

export const postProto: <T>(
  responseType: { deserializeBinary(x: Uint8Array): T },
  service: string,
  method: string,
  msg: { serializeBinary(): Uint8Array }
) => Promise<T> = async (responseType, service, method, msg) =>
  responseType.deserializeBinary((await postBinary(service, method, msg)).data);
