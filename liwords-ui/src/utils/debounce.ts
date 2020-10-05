// https://www.matthewgerstman.com/tech/throttle-and-debounce/

export function debounce(func: Function, timeout: number) {
  let timer: NodeJS.Timeout;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (...args: any) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      func(...args);
    }, timeout);
  };
}
