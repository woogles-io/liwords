import axios from 'axios';
import * as jspb from 'google-protobuf';

export const toAPIUrl = (service: string, method: string) => {
  const loc = window.location;
  const apiEndpoint = window.RUNTIME_CONFIGURATION.apiEndpoint || loc.host;

  // Assuming we don't need to encodeURIComponent() here...
  return `${loc.protocol}//${apiEndpoint}/twirp/${service}/${method}`;
};

export const postBinary = (
  service: string,
  method: string,
  msg: jspb.Message
) => {
  return axios.post(toAPIUrl(service, method), msg.serializeBinary(), {
    headers: {
      'Content-Type': 'application/protobuf',
    },
    responseType: 'arraybuffer',
  });
};
