export const toAPIUrl = (service: string, method: string) => {
  const loc = window.location;

  return (
    `${loc.protocol}//${window.RUNTIME_CONFIGURATION.apiEndpoint}/` +
    `twirp/${service}/${method}`
  );
};
