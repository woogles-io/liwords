export const getSocketURI = (username: string): string => {
  const loc = window.location;
  let socketURI;
  if (loc.protocol === 'https:') {
    socketURI = 'wss:';
  } else {
    socketURI = 'ws:';
  }
  socketURI += `//${loc.host}${loc.pathname}ws?user=${username}`;
  return socketURI;
};

export function websocket(
  uri: string,
  onSocket: (sock: WebSocket) => void,
  onEvent: (evt: MessageEvent) => void
) {
  const socket = new WebSocket(uri);
  socket.addEventListener('open', (event) => {
    console.log('connected');
    onSocket(socket);
  });
  socket.addEventListener('message', (event) => {
    onEvent(event);
  });
}
