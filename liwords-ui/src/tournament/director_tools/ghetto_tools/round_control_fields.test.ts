import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PairingMethod,
  RoundControlSchema,
} from "../../../gen/api/proto/ipc/tournament_pb";
import { settingsEqual } from "../pairing_methods";
import {
  pairingLagDisabledReason,
  rdCtrlFromSetting,
} from "./round_control_fields";

const setting = (fields: {
  pairingMethod?: PairingMethod;
  pairingLag?: number;
}) =>
  create(RoundControlSchema, {
    pairingMethod: PairingMethod.KING_OF_THE_HILL,
    ...fields,
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
