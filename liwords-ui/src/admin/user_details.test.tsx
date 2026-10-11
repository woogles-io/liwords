import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import { UserModActionSchema } from "../gen/api/proto/config_service/config_service_pb";
import { suspensionText, withoutSuspended } from "./user_details";

describe("suspensionText", () => {
  it("is blank for an account that isn't suspended", () => {
    expect(suspensionText(undefined)).toBe("");
  });

  it("says permanently for a suspension without an end", () => {
    expect(
      suspensionText(create(UserModActionSchema, { type: "SUSPEND_ACCOUNT" })),
    ).toBe("permanently");
  });

  it("says until when for a suspension that ends", () => {
    const end = new Date(2026, 9, 20, 13, 45);
    expect(
      suspensionText(
        create(UserModActionSchema, {
          type: "SUSPEND_ACCOUNT",
          end: timestampFromDate(end),
        }),
      ),
    ).toBe("until 2026-10-20 13:45");
  });
});

describe("withoutSuspended", () => {
  const rows = [
    { key: "clean" },
    { key: "suspended", suspension: create(UserModActionSchema, {}) },
  ];

  it("keeps every row unless asked to hide", () => {
    expect(withoutSuspended(rows, false)).toEqual(rows);
  });

  it("drops the suspended rows when asked to hide", () => {
    expect(withoutSuspended(rows, true)?.map((r) => r.key)).toEqual(["clean"]);
  });

  it("passes no rows through", () => {
    expect(withoutSuspended(undefined, true)).toBeUndefined();
  });
});
