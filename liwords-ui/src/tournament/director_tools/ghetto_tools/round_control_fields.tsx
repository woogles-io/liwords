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
  PairingMethodField,
  RoundSetting,
} from "../pairing_methods";
import {
  COPRoundControlFields,
  SingleRdCtrlFieldsProps,
  getCOPDefaults,
} from "./cop_fields";

export type RdCtrlFieldsProps = {
  setting: RoundSetting;
  onChange: (
    fieldName: string,
    value: string | number | boolean | number[] | string[],
  ) => void;
  onRemove: () => void;
  totalRounds?: number;
};

export const SingleRoundControlFields = (props: SingleRdCtrlFieldsProps) => {
  const { setting, beginRound, endRound, totalRounds } = props;
  const addlFields = fieldsForMethod(setting.pairingMethod);

  // Determine if COP should be disabled (only allowed for second half)
  const isCOPDisabled = React.useMemo(() => {
    if (beginRound === undefined || totalRounds === undefined) {
      // If we don't have round information, allow COP (e.g., in single round controls)
      return false;
    }
    // COP is only allowed if the BEGIN round is in the second half or later
    const halfwayPoint = Math.ceil(totalRounds / 2);
    return beginRound <= halfwayPoint;
  }, [beginRound, totalRounds]);

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
          players who are performing similarly.
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
          - <strong>King of the hill:</strong> These pairings pair 1v2, 3v4,
          5v6, and so forth. It is a good format for the very last round of a
          tournament.
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
          <Select.Option value={PairingMethod.SWISS}>Swiss</Select.Option>
          <Select.Option
            value={PairingMethod.PAIRING_METHOD_COP}
            disabled={isCOPDisabled}
          >
            COP (Castellano O'Connor Pairings)
            {isCOPDisabled && " - Only for 2nd half"}
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
          <Select.Option value={PairingMethod.INTERLEAVED_ROUND_ROBIN}>
            Shirts and Skins
          </Select.Option>
          <Select.Option value={PairingMethod.FACTOR}>Factor</Select.Option>
          <Select.Option value={PairingMethod.MANUAL}>Manual</Select.Option>
          <Select.Option value={PairingMethod.TEAM_ROUND_ROBIN}>
            Team Round Robin
          </Select.Option>
        </Select>
      </Form.Item>
      {isCOPDisabled &&
        setting.pairingMethod === PairingMethod.PAIRING_METHOD_COP &&
        beginRound !== undefined &&
        totalRounds !== undefined && (
          <p
            style={{
              fontSize: "12px",
              color: "#ff4d4f",
              marginTop: "-8px",
              marginBottom: "8px",
            }}
          >
            COP can only be used for the second half of the tournament (round{" "}
            {Math.ceil(totalRounds / 2) + 1} and later).
          </p>
        )}
      <p></p>
      {/* potential additional fields */}
      {addlFields.map((v: PairingMethodField, idx) => {
        const key = `ni-${idx}`;
        const [fieldType, fieldName, displayName, help] = v;
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
                  min={0}
                  value={setting[fieldName] as number}
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

export const rdCtrlFromSetting = (
  rdSetting: RoundControl,
  totalRounds?: number,
): RoundControl => {
  const rdCtrl = create(RoundControlSchema, {
    firstMethod: FirstMethod.AUTOMATIC_FIRST,
    gamesPerRound: 1,
    pairingMethod: rdSetting.pairingMethod,
  });

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
      rdCtrl.gamesPerRound = rdSetting.gamesPerRound || 1;
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
        0.25;
      break;
  }
  // Other cases don't matter, we've already set the pairing method.
  return rdCtrl;
};
