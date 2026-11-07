import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { OrganizationTitleSchema } from "../gen/api/proto/user_service/user_service_pb";
import { titleLines } from "./display_title";

const title = (
  organizationCode: string,
  rawTitle: string,
  titleFullName: string,
  verified = true,
) =>
  create(OrganizationTitleSchema, {
    organizationCode,
    rawTitle,
    titleFullName,
    verified,
  });

describe("titleLines", () => {
  const titles = [
    title("wespa", "M", "Master"),
    title("naspa", "GM", "Grandmaster"),
    title("sg", "SM", "Senior Master"),
  ];

  it("lists the title the circle shows first, then the rest in order", () => {
    expect(titleLines(titles, "naspa")).toEqual([
      "Grandmaster (NASPA)",
      "Master (WESPA)",
      "Senior Master (Singapore Scrabble Association)",
    ]);
  });

  it("leaves out a link without a title", () => {
    expect(titleLines([title("absp", "", ""), ...titles], "wespa")).toEqual([
      "Master (WESPA)",
      "Grandmaster (NASPA)",
      "Senior Master (Singapore Scrabble Association)",
    ]);
  });

  it("leaves out an unverified title, as an admin is sent", () => {
    expect(
      titleLines(
        [title("absp", "GM", "Grandmaster", false), ...titles],
        "wespa",
      ),
    ).toEqual([
      "Master (WESPA)",
      "Grandmaster (NASPA)",
      "Senior Master (Singapore Scrabble Association)",
    ]);
  });

  it("names a title without a full name by its letters", () => {
    expect(titleLines([title("wespa", "XM", "")], "wespa")).toEqual([
      "XM (WESPA)",
    ]);
  });
});
