export const getSocketURI = (username: string): string => {
  const loc = window.location;
  let socketURI;
  if (loc.protocol === 'https:') {
    socketURI = 'wss:';
  } else {
    socketURI = 'ws:';
  }
  if (loc.host.includes('localhost') || loc.host.includes('127.0.0.1')) {
    // Use the local Go server
    socketURI += `//localhost:8087/ws?user=${username}`;
  } else {
    // We are in prod; use same domain (use proxy on prod).
    socketURI += `//${loc.host}/ws?user=${username}`;
  }
  return socketURI;
};
