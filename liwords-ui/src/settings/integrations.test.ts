import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { OrganizationTitleSchema } from "../gen/api/proto/user_service/user_service_pb";
import { offersConnection } from "./integrations";

const title = (organizationCode: string, verified: boolean) =>
  create(OrganizationTitleSchema, { organizationCode, verified });

describe("offersConnection", () => {
  it("offers the Singapore Scrabble Association only to a Singapore flag", () => {
    expect(offersConnection("sg", "sg", [])).toBe(true);
    expect(offersConnection("sg", "SG", [])).toBe(true);
    expect(offersConnection("sg", "my", [])).toBe(false);
    expect(offersConnection("sg", "", [])).toBe(false);
    expect(offersConnection("sg", undefined, [])).toBe(false);
  });

  it("offers nothing the player is connected to or still waiting on", () => {
    expect(offersConnection("sg", "sg", [title("sg", false)])).toBe(false);
    expect(offersConnection("sg", "sg", [title("sg", true)])).toBe(false);
    expect(offersConnection("wespa", "", [title("wespa", false)])).toBe(false);
    expect(offersConnection("wespa", "", [title("sg", true)])).toBe(true);
  });

  it("leaves the other organizations as they were", () => {
    expect(offersConnection("wespa", "my", [])).toBe(true);
    expect(offersConnection("naspa", undefined, [])).toBe(true);
    expect(offersConnection("absp", "sg", [])).toBe(true);
    expect(offersConnection("absp", undefined, [])).toBe(true);
    expect(offersConnection("absp", "gb", [title("absp", true)])).toBe(false);
  });
});
