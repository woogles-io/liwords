import { errorMap } from '../store/constants';

function sprintf(template: string, args: Array<string>): string {
  let interpolated = template;
  args.forEach((arg) => {
    interpolated = interpolated.replace('$', arg);
  });
  return interpolated;
}

export function parseWooglesError(err: string): string {
  if (err.charAt(0) !== ';') {
    return err;
  }
  err = err.substring(1);
  const data = err.split(';');
  const errCode = data.shift();
  if (errCode === undefined) {
    return err;
  }
  const template = errorMap.get(Number(errCode));
  if (template === undefined) {
    return err;
  }
  return sprintf(template, data);
}
