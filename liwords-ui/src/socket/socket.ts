export const getSocketURI = (): string => {
  const loc = window.location;
  let socketURI;
  if (loc.protocol === 'https:') {
    socketURI = 'wss:';
  } else {
    socketURI = 'ws:';
  }
  // We are running local
  if (loc.host.includes('127.0.0.1') || loc.host.includes('localhost')) {
    socketURI += 'liwords.localhost/ws';
  } else {
    socketURI += `//${loc.host}/ws`;
  }

  return socketURI;
};
