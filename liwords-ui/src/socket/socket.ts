export const getSocketURI = (): string => {
  const loc = window.location;
  let socketURI;
  if (loc.protocol === 'https:') {
    socketURI = 'wss:';
  } else {
    socketURI = 'ws:';
  }

  socketURI += `//${loc.host}/ws`;

  return socketURI;
};
