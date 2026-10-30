// Round control field components

import { Button, Divider, Form, InputNumber, Select, Switch } from "antd";
import React from "react";
import { create } from "@bufbuild/protobuf";
import {
  FirstMethod,
  PairingMethod,
  RoundControl,
  RoundControlSchema,
} from "../../../gen/api/proto/ipc/tournament_pb";
import { HelptipLabel } from "../helptip_label";
import {
  fieldsForMethod,
  isPrePaired,
  PairingMethodField,
  RoundSetting,
} from "../pairing_methods";
import {
  COPRoundControlFields,
  SingleRdCtrlFieldsProps,
  getCOPDefaults,
} from "./cop_fields";

// The rounds from beginRound to endRound (1-based) that can have a pairing
// lag with the given pairing method, as [first, last], or undefined if none
// can. The first round has nothing before it, Elimination needs the previous
// round's results, and so do COP's last two rounds. The server enforces the
// same rules for each round.
export const laggableRounds = (
  pairingMethod: PairingMethod,
  beginRound?: number,
  endRound?: number,
  totalRounds?: number,
): [number, number] | undefined => {
  if (pairingMethod === PairingMethod.ELIMINATION) {
    return undefined;
  }
  const first = Math.max(beginRound ?? 2, 2);
  let last = endRound ?? first;
  if (
    pairingMethod === PairingMethod.PAIRING_METHOD_COP &&
    totalRounds !== undefined
  ) {
    last = Math.min(last, totalRounds - 2);
  }
  return first <= last ? [first, last] : undefined;
};

// Why none of rounds beginRound..endRound (1-based) can have a pairing lag
// with the given pairing method, or undefined if some can.
export const pairingLagDisabledReason = (
  pairingMethod: PairingMethod,
  beginRound?: number,
  endRound?: number,
  totalRounds?: number,
): string | undefined => {
  if (laggableRounds(pairingMethod, beginRound, endRound, totalRounds)) {
    return undefined;
  }
  if (pairingMethod === PairingMethod.ELIMINATION) {
    return "Elimination rounds cannot be lagged, because they need the previous round's results.";
  }
  if ((endRound ?? beginRound) === 1) {
    return isPrePaired(pairingMethod)
      ? "The first round can't start early, because there is no round before it."
      : "The first round cannot be lagged, because there are no earlier results to pair from.";
  }
  return "COP rounds in the last two rounds of a division cannot be lagged, because they must use the previous round's results.";
};

// Which rounds of a range a pairing lag applies to, when it can't apply to
// all of them, or undefined if it applies to every round (or none).
export const pairingLagAppliesTo = (
  pairingMethod: PairingMethod,
  beginRound?: number,
  endRound?: number,
  totalRounds?: number,
): string | undefined => {
  const rounds = laggableRounds(
    pairingMethod,
    beginRound,
    endRound,
    totalRounds,
  );
  if (
    !rounds ||
    beginRound === undefined ||
    endRound === undefined ||
    (rounds[0] === beginRound && rounds[1] === endRound)
  ) {
    return undefined;
  }
  const which =
    rounds[0] === rounds[1]
      ? `round ${rounds[0]}`
      : `rounds ${rounds[0]}–${rounds[1]}`;
  const skipped = [];
  if (beginRound <= 1) {
    skipped.push("round 1 has no round before it");
  }
  if (rounds[1] < endRound) {
    skipped.push("COP's last two rounds use the previous round's results");
  }
  return `This applies to ${which}: ${skipped.join(", and ")}.`;
};

export type RdCtrlFieldsProps = {
  setting: RoundSetting;
  onChange: (
    fieldName: string,
    value: string | number | boolean | number[] | string[],
  ) => void;
  onRemove: () => void;
  totalRounds?: number;
};

type SingleRoundControlFieldsProps = SingleRdCtrlFieldsProps & {
  // The 1-based round being edited, when editing a single round rather than a
  // range. Only used to decide whether the round can be lagged.
  round?: number;
};

export const SingleRoundControlFields = (
  props: SingleRoundControlFieldsProps,
) => {
  const { setting, beginRound, endRound, totalRounds, round } = props;
  const addlFields = fieldsForMethod(setting.pairingMethod);

  // Pre-paired rounds don't use results, so for them the pairing lag only
  // decides whether the round can start before the previous one ends.
  const prePaired = isPrePaired(setting.pairingMethod);
  const lagDisabledReason = pairingLagDisabledReason(
    setting.pairingMethod,
    beginRound ?? round,
    endRound ?? round,
    totalRounds,
  );
  // A range can include rounds the lag can't apply to, such as round 1;
  // those rounds are saved without it.
  const lagAppliesTo = setting.pairingLag
    ? pairingLagAppliesTo(
        setting.pairingMethod,
        beginRound,
        endRound,
        totalRounds,
      )
    : undefined;
  // With no results yet, COP pairs by seed: 1v2, 3v4, and so on.
  const copStartsFirstRound =
    setting.pairingMethod === PairingMethod.PAIRING_METHOD_COP &&
    (beginRound ?? round) === 1;

  const formItemLayout = {
    labelCol: {
      span: 6,
    },
    wrapperCol: {
      span: 8,
    },
  };

  const pairingTypesHelptip = (
    <>
      <ul>
        <li>
          - <strong>Random:</strong> Pairings are random. This is only
          recommended for the very first round.
        </li>
        <li>
          - <strong>Swiss:</strong> Swiss pairings by default try to match
          players who are performing similarly. Not recommended anymore: use COP
          or Win Groups instead.
        </li>
        <li>
          - <strong>Win Groups:</strong> Players on the same number of wins form
          a group, and each group&apos;s top half plays its bottom half (1v5,
          2v6, and so on) while avoiding earlier repeat meetings. Every pairing
          can be checked by hand.
        </li>
        <li>
          - <strong>Round Robin:</strong> These pairings match everyone in the
          division against each other. If there are fewer rounds than players,
          it will do a partial round robin.
        </li>
        <li>
          - <strong>Initial Fontes:</strong> These pairings split up the field
          into groups of size N+1, and pair everyone in the group against each
          other. The number you provide (N) must be an odd number. This should
          be used at the beginning of a tournament.
        </li>
        <li>
          - <strong>Shirts and Skins:</strong> These pairings split the division
          into two halves, by alternating seeds. Each player in one half plays a
          round robin against every player in the other half.
        </li>
        <li>
          - <strong>Snaked:</strong> These pairings split the division into two
          halves the way a snake draft picks teams (seeds 1, 4, 5, 8, 9, ...
          against 2, 3, 6, 7, 10, ...), so the halves are evenly matched. Each
          player in one half plays a round robin against every player in the
          other half.
        </li>
        <li>
          - <strong>Team Round Robin:</strong> Each player on one team plays a
          round robin against every player on the other team, using the teams
          you give players (for example, Portland vs Seattle). Teammates never
          play each other. If one team is bigger, its players sit out the
          difference in byes. Once players have teams, every round must use this
          or Manual pairings.
        </li>
        <li>
          - <strong>King of the hill:</strong> These pairings pair 1v2, 3v4,
          5v6, and so forth. It is a good format for the very last round of a
          tournament.
        </li>
        <li>
          - <strong>Australian Draw:</strong> The first round pairs the top half
          against the bottom half by seed; later rounds pair adjacent players by
          standing (1v2, 3v4, and so on) while avoiding earlier repeat meetings.
          Use the Reset Round field to choose from which round repeats start
          being avoided.
        </li>
        <li>
          - <strong>Factor:</strong> Factor 1 pairs 1 vs 2 and the rest Swiss.
          Factor 2 pairs 1v3, 2v4, and the rest Swiss. Factor 3 pairs 1v4, 2v5,
          3v6 and the rest Swiss, and so on.
        </li>
        <li>
          - <strong>Manual:</strong> Manual pairings must be provided manually
          by the director every round. This is not recommended except for the
          smallest tournaments.
        </li>
        <li>
          - <strong>Team Round Robin:</strong> Set up a round robin where each
          "team" member plays each other team member some set number of times in
          a row. You must divide teams into top and bottom halves, by "rating",
          and you must have an even number of players.
        </li>
      </ul>
    </>
  );

  const pairingLagHelptip = (
    <>
      <p>
        Which results this round's pairings are based on. Rounds are normally
        paired from the results through the previous round, so they cannot be
        paired until every game of the previous round is over.
      </p>
      <p>
        A round paired from the results through two rounds back is paired as
        soon as the round before the previous one is complete. It can then be
        opened while the previous round is still being played, so players who
        finish early do not wait for the slowest game. Each player still has to
        finish their previous game before starting this one.
      </p>
      <p>
        This cannot be used for the first round, for Elimination rounds, or for
        COP rounds that are one of the last two rounds.
      </p>
    </>
  );

  const startEarlyHelptip = (
    <>
      <p>
        These rounds are paired in advance, so they don't wait for results. With
        this on, the round can be started once the round before the previous one
        is complete, while some games of the previous round are still being
        played. Players who finish early don't wait for the slowest game. Each
        player still has to finish their previous game before starting this one.
      </p>
      <p>You still start the round yourself.</p>
    </>
  );

  return (
    <>
      <Form.Item
        {...formItemLayout}
        label={
          <HelptipLabel labelText="Pairing type" help={pairingTypesHelptip} />
        }
      >
        <Select
          value={setting.pairingMethod}
          onChange={(e) => {
            props.onChange("pairingMethod", e);
          }}
        >
          <Select.Option value={PairingMethod.RANDOM}>Random</Select.Option>
          <Select.Option value={PairingMethod.WIN_GROUPS}>
            Win Groups
          </Select.Option>
          <Select.Option value={PairingMethod.PAIRING_METHOD_COP}>
            COP (Castellano O'Connor Pairings)
          </Select.Option>
          <Select.Option value={PairingMethod.ROUND_ROBIN}>
            Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.INITIAL_FONTES}>
            Initial Fontes
          </Select.Option>
          <Select.Option value={PairingMethod.KING_OF_THE_HILL}>
            King of the Hill
          </Select.Option>
          <Select.Option value={PairingMethod.AUSTRALIAN_DRAW}>
            Australian Draw
          </Select.Option>
          <Select.Option value={PairingMethod.INTERLEAVED_ROUND_ROBIN}>
            Shirts and Skins Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.SNAKED_ROUND_ROBIN}>
            Snaked (Boustrophedonic) Round Robin
          </Select.Option>
          <Select.Option value={PairingMethod.FACTOR}>Factor</Select.Option>
          <Select.Option value={PairingMethod.MANUAL}>Manual</Select.Option>
          <Select.Option value={PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN}>
            Team Round Robin
          </Select.Option>
          {/* Retired in favor of the Team Round Robin between assigned
              teams. Still listed, so divisions that used it show it, but it
              can't be picked. */}
          <Select.Option value={PairingMethod.TEAM_ROUND_ROBIN} disabled>
            Top vs Bottom Round Robin (retired)
          </Select.Option>
          {/* Not recommended anymore, so it's last and grayed out, but
              still selectable. */}
          <Select.Option value={PairingMethod.SWISS}>
            <span style={{ color: "#999" }}>Swiss (not recommended)</span>
          </Select.Option>
        </Select>
      </Form.Item>
      {setting.pairingMethod === PairingMethod.SWISS && (
        <p
          style={{
            fontSize: "12px",
            color: "#faad14",
            marginTop: "-8px",
            marginBottom: "8px",
          }}
        >
          Swiss isn't recommended anymore. We strongly recommend COP, or Win
          Groups, which pairs players with the same number of wins against each
          other.
        </p>
      )}
      {copStartsFirstRound && (
        <p
          style={{
            fontSize: "12px",
            color: "#faad14",
            marginTop: "-8px",
            marginBottom: "8px",
          }}
        >
          A pre-paired scheme, such as Initial Fontes or Round Robin, is
          recommended for the first few rounds of a tournament. With no results
          yet, COP pairs players by seed (1 vs 2, 3 vs 4, and so on), so the top
          players meet right away.
        </p>
      )}
      {prePaired ? (
        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Can start early"
              help={startEarlyHelptip}
            />
          }
        >
          <Switch
            checked={!!setting.pairingLag}
            // Leave it enabled while it is on, so that an invalid lag can be
            // turned off.
            disabled={!!lagDisabledReason && !setting.pairingLag}
            onChange={(checked) => {
              props.onChange("pairingLag", checked ? 1 : 0);
            }}
          />
        </Form.Item>
      ) : (
        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel labelText="Paired from" help={pairingLagHelptip} />
          }
        >
          <Select
            value={setting.pairingLag || 0}
            // Leave it enabled while it is set, so that an invalid lag can be
            // turned off.
            disabled={!!lagDisabledReason && !setting.pairingLag}
            onChange={(e) => {
              props.onChange("pairingLag", e);
            }}
          >
            <Select.Option value={0}>Previous round</Select.Option>
            <Select.Option value={1}>Two rounds back</Select.Option>
          </Select>
        </Form.Item>
      )}
      {lagDisabledReason && (
        <p
          style={{
            fontSize: "12px",
            color: setting.pairingLag ? "#ff4d4f" : "#666",
            marginTop: "-8px",
            marginBottom: "8px",
          }}
        >
          {lagDisabledReason}
        </p>
      )}
      {lagAppliesTo && (
        <p
          style={{
            fontSize: "12px",
            color: "#666",
            marginTop: "-8px",
            marginBottom: "8px",
          }}
        >
          {lagAppliesTo}
        </p>
      )}
      <p></p>
      {/* potential additional fields */}
      {addlFields.map((v: PairingMethodField, idx) => {
        const key = `ni-${idx}`;
        const [fieldType, fieldName, displayName, help, fieldMin] = v;
        const minValue = fieldMin ?? 0;
        switch (fieldType) {
          case "number":
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 16, offset: 1 }}
                label={<HelptipLabel labelText={displayName} help={help} />}
                key={`${idx}-${fieldName}`}
              >
                <InputNumber
                  inputMode="numeric"
                  key={key}
                  min={minValue}
                  value={(setting[fieldName] as number) || minValue}
                  onChange={(e) => {
                    props.onChange(fieldName, e as number);
                  }}
                />
              </Form.Item>
            );

          case "boolean":
            return (
              <Form.Item
                {...formItemLayout}
                labelCol={{ span: 12, offset: 1 }}
                label={<HelptipLabel labelText={displayName} help={help} />}
                key={`${idx}-${fieldName}`}
              >
                <Switch
                  key={key}
                  checked={setting[fieldName] as boolean}
                  onChange={(e) => props.onChange(fieldName, e)}
                />
              </Form.Item>
            );
        }
        return null;
      })}
      {/* Conditional rendering for COP configuration */}
      {setting.pairingMethod === PairingMethod.PAIRING_METHOD_COP && (
        <COPRoundControlFields
          setting={setting}
          onChange={props.onChange}
          beginRound={beginRound}
          endRound={endRound}
          totalRounds={totalRounds}
        />
      )}
    </>
  );
};

export const RoundControlFields = (props: RdCtrlFieldsProps) => {
  const { setting, totalRounds } = props;
  return (
    <>
      <h5 className="readable-text-color">
        {setting.beginRound === setting.endRound
          ? `Round ${setting.beginRound}`
          : `Rounds ${setting.beginRound} to ${setting.endRound}`}
        {setting.setting.pairingLag > 0 && " (lagged)"}
      </h5>
      <Form size="small">
        <Form.Item label="First round">
          <InputNumber
            inputMode="numeric"
            min={1}
            value={setting.beginRound}
            onChange={(e) => props.onChange("beginRound", e as number)}
          />
        </Form.Item>
        <Form.Item label="Last round">
          <InputNumber
            inputMode="numeric"
            min={1}
            value={setting.endRound}
            onChange={(e) => props.onChange("endRound", e as number)}
          />
        </Form.Item>
      </Form>
      <Form size="small" style={{ marginTop: 8 }}>
        <SingleRoundControlFields
          setting={setting.setting}
          onChange={props.onChange}
          beginRound={setting.beginRound}
          endRound={setting.endRound}
          totalRounds={totalRounds}
        />
      </Form>
      <Button onClick={props.onRemove}>- Remove</Button>
      <Divider />
    </>
  );
};

// round is the 1-based round this control is for, when known. It is only used
// to validate the pairing lag.
export const rdCtrlFromSetting = (
  rdSetting: RoundControl,
  totalRounds?: number,
  round?: number,
): RoundControl => {
  const rdCtrl = create(RoundControlSchema, {
    firstMethod: FirstMethod.AUTOMATIC_FIRST,
    gamesPerRound: 1,
    pairingMethod: rdSetting.pairingMethod,
  });

  // The pairing lag applies to every pairing method.
  rdCtrl.pairingLag = rdSetting.pairingLag || 0;
  if (rdCtrl.pairingLag > 1) {
    throw new Error(
      `${round !== undefined ? `Round ${round} has` : "This round has"} an invalid pairing lag of ${rdCtrl.pairingLag}. The lag must be 0 or 1.`,
    );
  }
  // A range's lag skips the rounds that can't have one, such as round 1.
  if (
    rdCtrl.pairingLag > 0 &&
    !laggableRounds(rdCtrl.pairingMethod, round, round, totalRounds)
  ) {
    rdCtrl.pairingLag = 0;
  }

  switch (rdSetting.pairingMethod) {
    case PairingMethod.SWISS:
    case PairingMethod.FACTOR:
      rdCtrl.maxRepeats = rdSetting.maxRepeats || 0;
      rdCtrl.allowOverMaxRepeats = true;
      rdCtrl.repeatRelativeWeight = rdSetting.repeatRelativeWeight || 0;
      rdCtrl.winDifferenceRelativeWeight =
        rdSetting.winDifferenceRelativeWeight || 0;
      // This should be auto-calculated, and only for factor
      if (
        rdSetting.pairingMethod === PairingMethod.FACTOR &&
        rdSetting.factor <= 0
      ) {
        throw new Error(
          "Factor 0 is equivalent to just Swiss for every player. Use Swiss pairings instead, or use a Factor greater than 0.",
        );
      }
      rdCtrl.factor = rdSetting.factor || 0;

      if (rdCtrl.maxRepeats <= 0) {
        throw new Error(
          "Max Pairings Between Any Two Players should be at least 1. Setting it to 0 will allow no pairings to occur.",
        );
      }

      if (rdCtrl.repeatRelativeWeight <= 0) {
        throw new Error(
          "Repeat relative weight should be at least 1. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.winDifferenceRelativeWeight <= 0) {
        throw new Error(
          "Win difference relative weight should be at least 1. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.repeatRelativeWeight > 100) {
        throw new Error(
          "Repeat relative weight should be at most 100. Please hover on the question mark to see more info about what this means.",
        );
      }

      if (rdCtrl.winDifferenceRelativeWeight > 100) {
        throw new Error(
          "Win difference relative weight should be at most 100. Please hover on the question mark to see more info about what this means.",
        );
      }

      break;

    case PairingMethod.TEAM_ROUND_ROBIN:
    case PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN:
      rdCtrl.gamesPerRound = rdSetting.gamesPerRound || 1;
      break;

    case PairingMethod.INTERLEAVED_ROUND_ROBIN:
    case PairingMethod.SNAKED_ROUND_ROBIN:
      rdCtrl.playWithinTeam = rdSetting.playWithinTeam;
      break;

    case PairingMethod.WIN_GROUPS:
    case PairingMethod.AUSTRALIAN_DRAW:
      // The Reset Round field is 1-based (a round number; the default is 1 =
      // avoid all repeats), but the proto/engine store reset_round 0-based like
      // every other round. Convert on the way in: input N -> N-1. Default input
      // 1 -> 0, which is also the proto3 zero-default ("avoid all"). A new
      // round's setting holds the proto3 default of 0 rather than undefined,
      // while the field shows 1, so 0 is read as 1 too.
      rdCtrl.resetRound = (rdSetting.resetRound || 1) - 1;
      break;

    case PairingMethod.PAIRING_METHOD_COP:
      // Apply defaults if values are not set
      const copDefaults = totalRounds ? getCOPDefaults(totalRounds) : null;

      rdCtrl.gibsonSpreads =
        rdSetting.gibsonSpreads && rdSetting.gibsonSpreads.length > 0
          ? rdSetting.gibsonSpreads
          : copDefaults?.gibsonSpreads || [250, 200];

      rdCtrl.hopefulnessThresholds =
        rdSetting.hopefulnessThresholds &&
        rdSetting.hopefulnessThresholds.length > 0
          ? rdSetting.hopefulnessThresholds
          : copDefaults?.hopefulnessThresholds || [0.1, 0.1];

      rdCtrl.placePrizes =
        rdSetting.placePrizes || copDefaults?.placePrizes || 4;
      rdCtrl.controlLossActivationRound =
        rdSetting.controlLossActivationRound ||
        copDefaults?.controlLossActivationRound ||
        0;
      rdCtrl.divisionSims =
        rdSetting.divisionSims || copDefaults?.divisionSims || 100000;
      rdCtrl.controlLossSims =
        rdSetting.controlLossSims || copDefaults?.controlLossSims || 10000;
      rdCtrl.controlLossThreshold =
        rdSetting.controlLossThreshold ||
        copDefaults?.controlLossThreshold ||
        0.4;
      break;
  }
  // Other cases don't matter, we've already set the pairing method.
  return rdCtrl;
};
