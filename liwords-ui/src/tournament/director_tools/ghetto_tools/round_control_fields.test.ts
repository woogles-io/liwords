import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PairingMethod,
  RoundControlSchema,
} from "../../../gen/api/proto/ipc/tournament_pb";
import {
  fieldsForMethod,
  isPrePaired,
  settingsEqual,
} from "../pairing_methods";
import {
  pairingLagDisabledReason,
  rdCtrlFromSetting,
} from "./round_control_fields";

const setting = (fields: {
  pairingMethod?: PairingMethod;
  pairingLag?: number;
  resetRound?: number;
  playWithinTeam?: boolean;
}) =>
  create(RoundControlSchema, {
    pairingMethod: PairingMethod.KING_OF_THE_HILL,
    ...fields,
  });

describe("play within team", () => {
  it("is kept for the team-splitting round robins", () => {
    for (const pairingMethod of [
      PairingMethod.INTERLEAVED_ROUND_ROBIN,
      PairingMethod.SNAKED_ROUND_ROBIN,
    ]) {
      expect(
        rdCtrlFromSetting(
          setting({ pairingMethod, playWithinTeam: true }),
          7,
          1,
        ).playWithinTeam,
      ).toBe(true);
      expect(
        rdCtrlFromSetting(setting({ pairingMethod }), 7, 1).playWithinTeam,
      ).toBe(false);
    }
  });
});

describe("the pairing lag of a round control", () => {
  it("is copied for every pairing method", () => {
    expect(rdCtrlFromSetting(setting({ pairingLag: 1 }), 7, 5).pairingLag).toBe(
      1,
    );
    expect(rdCtrlFromSetting(setting({}), 7, 5).pairingLag).toBe(0);
  });

  it("must be 0 or 1", () => {
    expect(() => rdCtrlFromSetting(setting({ pairingLag: 2 }), 7, 5)).toThrow(
      "Round 5 has an invalid pairing lag of 2. The lag must be 0 or 1.",
    );
  });

  it("is not allowed on the first round", () => {
    expect(() => rdCtrlFromSetting(setting({ pairingLag: 1 }), 7, 1)).toThrow(
      /first round cannot be lagged/,
    );
  });

  it("is not allowed on an elimination round", () => {
    expect(() =>
      rdCtrlFromSetting(
        setting({ pairingMethod: PairingMethod.ELIMINATION, pairingLag: 1 }),
        7,
        5,
      ),
    ).toThrow(/Elimination rounds cannot be lagged/);
  });

  it("is not allowed on a COP round in the last two rounds", () => {
    const cop = setting({
      pairingMethod: PairingMethod.PAIRING_METHOD_COP,
      pairingLag: 1,
    });
    expect(() => rdCtrlFromSetting(cop, 7, 6)).toThrow(/COP rounds/);
    expect(() => rdCtrlFromSetting(cop, 7, 7)).toThrow(/COP rounds/);
    expect(rdCtrlFromSetting(cop, 7, 5).pairingLag).toBe(1);
  });

  it("is allowed on a round that is unaffected by any rule", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.SWISS, 2, 7, 7),
    ).toBeUndefined();
  });

  it("is allowed and kept on an Australian Draw round", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.AUSTRALIAN_DRAW, 2, 7, 7),
    ).toBeUndefined();
    const audraw = setting({
      pairingMethod: PairingMethod.AUSTRALIAN_DRAW,
      pairingLag: 1,
    });
    expect(rdCtrlFromSetting(audraw, 7, 5).pairingLag).toBe(1);
  });

  it("explains a range that starts with the first round", () => {
    expect(pairingLagDisabledReason(PairingMethod.SWISS, 1, 3, 7)).toMatch(
      /first round/,
    );
  });

  it("explains a COP range that reaches the last two rounds", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.PAIRING_METHOD_COP, 5, 6, 7),
    ).toMatch(/COP rounds/);
    expect(
      pairingLagDisabledReason(PairingMethod.PAIRING_METHOD_COP, 4, 5, 7),
    ).toBeUndefined();
  });

  it("keeps rounds that differ only by lag in separate ranges", () => {
    expect(settingsEqual(setting({}), setting({ pairingLag: 1 }))).toBe(false);
    expect(
      settingsEqual(setting({ pairingLag: 1 }), setting({ pairingLag: 1 })),
    ).toBe(true);
  });
});

describe("the reset round of an Australian Draw round", () => {
  it("is round 1 when the director leaves the field alone", () => {
    // A new round's setting holds the proto3 default of 0, while the field
    // shows 1.
    const audraw = setting({ pairingMethod: PairingMethod.AUSTRALIAN_DRAW });
    expect(rdCtrlFromSetting(audraw, 7, 3).resetRound).toBe(0);
  });

  it("is stored 0-based", () => {
    const audraw = setting({
      pairingMethod: PairingMethod.AUSTRALIAN_DRAW,
      resetRound: 9,
    });
    expect(rdCtrlFromSetting(audraw, 16, 12).resetRound).toBe(8);
  });
});

describe("a Win Groups round", () => {
  it("offers the Reset Round field", () => {
    expect(fieldsForMethod(PairingMethod.WIN_GROUPS).map((f) => f[1])).toEqual([
      "resetRound",
    ]);
  });

  it("has reset round 1 when the director leaves the field alone", () => {
    const wg = setting({ pairingMethod: PairingMethod.WIN_GROUPS });
    expect(rdCtrlFromSetting(wg, 7, 3).resetRound).toBe(0);
  });

  it("stores the reset round 0-based", () => {
    const wg = setting({
      pairingMethod: PairingMethod.WIN_GROUPS,
      resetRound: 9,
    });
    expect(rdCtrlFromSetting(wg, 16, 12).resetRound).toBe(8);
  });

  it("can be lagged", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.WIN_GROUPS, 2, 7, 7),
    ).toBeUndefined();
    const wg = setting({
      pairingMethod: PairingMethod.WIN_GROUPS,
      pairingLag: 1,
    });
    expect(rdCtrlFromSetting(wg, 7, 5).pairingLag).toBe(1);
  });
});

describe("pre-paired methods", () => {
  it("are the ones that don't pair from results", () => {
    for (const m of [
      PairingMethod.INITIAL_FONTES,
      PairingMethod.ROUND_ROBIN,
      PairingMethod.SNAKED_ROUND_ROBIN,
      PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN,
      PairingMethod.MANUAL,
    ]) {
      expect(isPrePaired(m)).toBe(true);
    }
    for (const m of [
      PairingMethod.PAIRING_METHOD_COP,
      PairingMethod.SWISS,
      PairingMethod.KING_OF_THE_HILL,
      PairingMethod.WIN_GROUPS,
      PairingMethod.AUSTRALIAN_DRAW,
    ]) {
      expect(isPrePaired(m)).toBe(false);
    }
  });
});
