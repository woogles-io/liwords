export const getSocketURI = (jwt: string): string => {
  const loc = window.location;
  let socketURI;
  if (loc.protocol === 'https:') {
    socketURI = 'wss:';
  } else {
    socketURI = 'ws:';
  }
  socketURI += `//${loc.host}/ws?user=${jwt}`;

  return socketURI;
};
