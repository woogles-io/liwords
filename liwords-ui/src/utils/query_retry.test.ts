import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { shouldRetryQuery } from "./query_retry";

describe("shouldRetryQuery", () => {
  it("does not retry errors that won't change", () => {
    expect(
      shouldRetryQuery(0, new ConnectError("no such broadcast", Code.NotFound)),
    ).toBe(false);
    expect(
      shouldRetryQuery(0, new ConnectError("nope", Code.PermissionDenied)),
    ).toBe(false);
  });

  it("retries other errors up to three times", () => {
    const err = new ConnectError("try again", Code.Unavailable);
    expect(shouldRetryQuery(0, err)).toBe(true);
    expect(shouldRetryQuery(2, err)).toBe(true);
    expect(shouldRetryQuery(3, err)).toBe(false);
    expect(shouldRetryQuery(0, new Error("network"))).toBe(true);
  });
});
