import axios from 'axios';

export const toAPIUrl = (service: string, method: string) => {
  const loc = window.location;
  const apiEndpoint = window.RUNTIME_CONFIGURATION.apiEndpoint || loc.host;

  // Assuming we don't need to encodeURIComponent() here...
  return `${loc.protocol}//${apiEndpoint}/twirp/${service}/${method}`;
};

interface PBMsg {
  serializeBinary(): Uint8Array;
}

// Warning -- trying to install @types/google-protobuf (so that we can use
// the built-in jspb.Message interface) resulted in completely
// unrelated build errors for me. I have no idea why. This can be revisited later
// I hope.

export const postBinary = (service: string, method: string, msg: PBMsg) => {
  return axios.post(toAPIUrl(service, method), msg.serializeBinary(), {
    headers: {
      'Content-Type': 'application/protobuf',
    },
    responseType: 'arraybuffer',
  });
};

export const postProto: <T>(
  responseType: { deserializeBinary(x: Uint8Array): T },
  service: string,
  method: string,
  msg: { serializeBinary(): Uint8Array }
) => Promise<T> = async (responseType, service, method, msg) =>
  responseType.deserializeBinary((await postBinary(service, method, msg)).data);
