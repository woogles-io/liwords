import { Code, ConnectError } from "@connectrpc/connect";

// Errors that asking again won't change. A query that fails with one of these
// shows its error at once instead of after three retries' worth of backoff.
const permanentCodes = new Set<Code>([
  Code.NotFound,
  Code.PermissionDenied,
  Code.Unauthenticated,
  Code.InvalidArgument,
]);

const maxRetries = 3; // react-query's default

export function shouldRetryQuery(failureCount: number, error: unknown) {
  if (error instanceof ConnectError && permanentCodes.has(error.code)) {
    return false;
  }
  return failureCount < maxRetries;
}
