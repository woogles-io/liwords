// A simple pubsub module
type CallbackFn = (data: string) => void;

const subscribers: { [e: string]: Array<CallbackFn> } = {};

export const publish = (event: string, data: string) => {
  if (!subscribers[event]) return;
  subscribers[event].forEach((subscriberCallback) => subscriberCallback(data));
};

export const subscribe = (event: string, callback: CallbackFn) => {
  if (!subscribers[event]) {
    subscribers[event] = [];
  }

  const index = subscribers[event].push(callback) - 1;

  return {
    unsubscribe: () => {
      subscribers[event].splice(index, 1);
    },
  };
};
