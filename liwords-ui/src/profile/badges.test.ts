import { readdirSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { BADGE_DESCRIPTIONS } from "../gen/badges";

// import.meta.url is not a file: URL under the jsdom environment, so resolve
// from the package root instead — vitest runs with cwd set to liwords-ui/.
const badgeDir = resolve(process.cwd(), "src/assets/badges");

// Mirrors the filename derivation in getImage() in badge.tsx. If that changes,
// change it here too.
const fileFor = (badgeCode: string) =>
  `${badgeCode.toLowerCase().replaceAll(" ", "_")}.png`;

describe("badge metadata", () => {
  const codes = Object.keys(BADGE_DESCRIPTIONS);
  const images = new Set(readdirSync(badgeDir));

  it("is non-empty", () => {
    expect(codes.length).toBeGreaterThan(0);
  });

  it("has an image for every badge code", () => {
    const missing = codes.filter((code) => !images.has(fileFor(code)));
    expect(missing).toEqual([]);
  });

  it("has no orphaned badge images", () => {
    // An orphan usually means a PNG was committed but cmd/gen-badges was not
    // re-run after calling ConfigService.AddBadge.
    const expected = new Set(codes.map(fileFor));
    const orphans = [...images].filter((f) => !expected.has(f));
    expect(orphans).toEqual([]);
  });

  it("has a non-empty description for every badge code", () => {
    const blank = codes.filter((code) => !BADGE_DESCRIPTIONS[code]?.trim());
    expect(blank).toEqual([]);
  });
});
