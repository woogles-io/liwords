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

export const postBinary = async (
  service: string,
  method: string,
  msg: PBMsg
) => {
  return axios.post(toAPIUrl(service, method), msg.serializeBinary(), {
    headers: {
      'Content-Type': 'application/protobuf',
    },
    responseType: 'arraybuffer',
  });
};
