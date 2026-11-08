import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  PairingMethod,
  RoundControlSchema,
  TournamentPersonSchema,
} from "../../../gen/api/proto/ipc/tournament_pb";
import {
  assignedTeamsProblem,
  fieldsForMethod,
  isPrePaired,
  seedOrderWhenPicked,
  settingsEqual,
  teamNames,
} from "../pairing_methods";
import {
  pairingLagAppliesTo,
  pairingLagDisabledReason,
  rdCtrlFromSetting,
  settingFromRdCtrl,
} from "./round_control_fields";

const setting = (fields: {
  pairingMethod?: PairingMethod;
  pairingLag?: number;
  resetRound?: number;
  playWithinTeam?: boolean;
  gamesPerRound?: number;
  seedOrder?: boolean;
}) =>
  create(RoundControlSchema, {
    pairingMethod: PairingMethod.KING_OF_THE_HILL,
    ...fields,
  });

describe("rounds per opponent", () => {
  it("is offered and kept for the round robins that repeat opponents", () => {
    for (const pairingMethod of [
      PairingMethod.ROUND_ROBIN,
      PairingMethod.TEAM_ROUND_ROBIN,
      PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN,
    ]) {
      expect(fieldsForMethod(pairingMethod).map((f) => f[1])).toContain(
        "gamesPerRound",
      );
      expect(
        rdCtrlFromSetting(setting({ pairingMethod, gamesPerRound: 2 }), 10, 1)
          .gamesPerRound,
      ).toBe(2);
      expect(
        rdCtrlFromSetting(setting({ pairingMethod }), 10, 1).gamesPerRound,
      ).toBe(1);
    }
  });

  it("is 1 for the other methods", () => {
    expect(
      rdCtrlFromSetting(
        setting({
          pairingMethod: PairingMethod.KING_OF_THE_HILL,
          gamesPerRound: 2,
        }),
        10,
        1,
      ).gamesPerRound,
    ).toBe(1);
  });
});

describe("seed order", () => {
  it("is offered and kept for the round robin only", () => {
    expect(fieldsForMethod(PairingMethod.ROUND_ROBIN).map((f) => f[1])).toEqual(
      ["gamesPerRound", "seedOrder"],
    );
    expect(
      rdCtrlFromSetting(
        setting({ pairingMethod: PairingMethod.ROUND_ROBIN, seedOrder: true }),
        10,
        1,
      ).seedOrder,
    ).toBe(true);
    for (const pairingMethod of [
      PairingMethod.TEAM_ROUND_ROBIN,
      PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN,
      PairingMethod.KING_OF_THE_HILL,
    ]) {
      expect(fieldsForMethod(pairingMethod).map((f) => f[1])).not.toContain(
        "seedOrder",
      );
      expect(
        rdCtrlFromSetting(setting({ pairingMethod, seedOrder: true }), 10, 1)
          .seedOrder,
      ).toBe(false);
    }
  });

  it("is shown as saved, and keeps round ranges apart", () => {
    const saved = rdCtrlFromSetting(
      setting({ pairingMethod: PairingMethod.ROUND_ROBIN, seedOrder: true }),
      10,
      1,
    );
    expect(settingFromRdCtrl(saved).seedOrder).toBe(true);
    expect(
      settingsEqual(
        settingFromRdCtrl(saved),
        settingFromRdCtrl(
          rdCtrlFromSetting(
            setting({ pairingMethod: PairingMethod.ROUND_ROBIN }),
            10,
            1,
          ),
        ),
      ),
    ).toBe(false);
  });

  it("turns on when Round Robin is picked, and stays as saved otherwise", () => {
    const manual = setting({ pairingMethod: PairingMethod.MANUAL });
    const picked = setting({ pairingMethod: PairingMethod.ROUND_ROBIN });
    expect(seedOrderWhenPicked(manual, picked).seedOrder).toBe(true);
    // A round robin saved without seed order keeps it off when edited.
    const saved = setting({ pairingMethod: PairingMethod.ROUND_ROBIN });
    const edited = setting({
      pairingMethod: PairingMethod.ROUND_ROBIN,
      gamesPerRound: 2,
    });
    expect(seedOrderWhenPicked(saved, edited).seedOrder).toBe(false);
    // The director can still turn it off.
    const on = setting({
      pairingMethod: PairingMethod.ROUND_ROBIN,
      seedOrder: true,
    });
    const off = setting({ pairingMethod: PairingMethod.ROUND_ROBIN });
    expect(seedOrderWhenPicked(on, off).seedOrder).toBe(false);
  });
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

  it("is shown as saved when the round controls are edited again", () => {
    const saved = rdCtrlFromSetting(
      setting({
        pairingMethod: PairingMethod.SNAKED_ROUND_ROBIN,
        playWithinTeam: true,
      }),
      7,
      1,
    );
    expect(settingFromRdCtrl(saved).playWithinTeam).toBe(true);
    expect(
      rdCtrlFromSetting(settingFromRdCtrl(saved), 7, 1).playWithinTeam,
    ).toBe(true);
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

  it("is left off the first round", () => {
    expect(rdCtrlFromSetting(setting({ pairingLag: 1 }), 7, 1).pairingLag).toBe(
      0,
    );
  });

  it("is left off an elimination round", () => {
    expect(
      rdCtrlFromSetting(
        setting({ pairingMethod: PairingMethod.ELIMINATION, pairingLag: 1 }),
        7,
        5,
      ).pairingLag,
    ).toBe(0);
  });

  it("is left off COP's last two rounds", () => {
    const cop = setting({
      pairingMethod: PairingMethod.PAIRING_METHOD_COP,
      pairingLag: 1,
    });
    expect(rdCtrlFromSetting(cop, 7, 6).pairingLag).toBe(0);
    expect(rdCtrlFromSetting(cop, 7, 7).pairingLag).toBe(0);
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

  it("is allowed on a range that starts with the first round", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.SWISS, 1, 3, 7),
    ).toBeUndefined();
    expect(pairingLagAppliesTo(PairingMethod.INITIAL_FONTES, 1, 6, 16)).toBe(
      "This applies to rounds 2–6: round 1 has no round before it.",
    );
  });

  it("explains a range that is only the first round", () => {
    expect(pairingLagDisabledReason(PairingMethod.SWISS, 1, 1, 7)).toMatch(
      /first round cannot be lagged/,
    );
    expect(
      pairingLagDisabledReason(PairingMethod.INITIAL_FONTES, 1, 1, 7),
    ).toMatch(/first round can't start early/);
  });

  it("says which rounds of a COP range it applies to", () => {
    expect(
      pairingLagAppliesTo(PairingMethod.PAIRING_METHOD_COP, 4, 16, 16),
    ).toBe(
      "This applies to rounds 4–14: COP's last two rounds use the previous round's results.",
    );
    expect(
      pairingLagAppliesTo(PairingMethod.PAIRING_METHOD_COP, 4, 12, 16),
    ).toBeUndefined();
  });

  it("explains a COP range that is only the last two rounds", () => {
    expect(
      pairingLagDisabledReason(PairingMethod.PAIRING_METHOD_COP, 6, 7, 7),
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

describe("assigned teams", () => {
  const players = (...teams: string[]) =>
    teams.map((team, i) =>
      create(TournamentPersonSchema, { id: `u${i}:p${i}`, team }),
    );

  it("are fine on exactly two teams", () => {
    expect(
      assignedTeamsProblem(players("Portland", "Seattle", "Seattle")),
    ).toBeUndefined();
  });

  it("list every team when there are more than two, to show typos", () => {
    expect(
      assignedTeamsProblem(
        players("Portland", "Portland", "Seattle", "Seattle", "Seatte"),
      ),
    ).toBe(
      "The Team Round Robin needs every player on one of exactly two teams. " +
        "This division has Portland (2 players), Seattle (2 players), Seatte (1 player). " +
        "Fix teams in Edit player.",
    );
  });

  it("count players without a team", () => {
    expect(assignedTeamsProblem(players("Portland", "Seattle", ""))).toBe(
      "The Team Round Robin needs every player on one of exactly two teams. " +
        "This division has Portland (1 player), Seattle (1 player), 1 player without a team. " +
        "Fix teams in Edit player.",
    );
  });

  it("are named most players first", () => {
    expect(teamNames(players("B", "A", "A", "", "C", "B", "A"))).toEqual([
      "A",
      "B",
      "C",
    ]);
  });
});
