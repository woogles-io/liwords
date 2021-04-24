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

interface TwirpError {
  response: {
    data: Uint8Array;
  };
}

export const twirpErrToMsg = (err: TwirpError) => {
  // Twirp always returns JSON error messages no matter what. But since the
  // responseType is set to `arraybuffer` above it is annoying to deal with.
  // This function turns it into the JSON-encoded string that it is and
  // extracts the error message.
  const errJSON = String.fromCharCode.apply(
    null,
    Array.from(new Uint8Array(err.response.data))
  );
  return JSON.parse(errJSON).msg;
};
