import { errorMap } from "../store/constants";

function sprintf(template: string, args: Array<string>): string {
  return template.replace(/\$(\d+)/g, (_, i) => args[i - 1]);
}

export function errorInfo(err: string): [number, string[]] {
  if (err.charAt(0) !== ";") {
    return [0, []];
  }
  err = err.substring(1);
  const data = err.split(";");
  const errCode = data.shift();
  if (errCode === undefined) {
    return [0, []];
  }
  return [Number(errCode), data];
}

export function parseWooglesError(err: string): string {
  const [errCode, data] = errorInfo(err);
  if (errCode === 0) {
    return err;
  }
  const template = errorMap.get(Number(errCode));
  if (template === undefined) {
    return err;
  }
  return sprintf(template, data);
}
