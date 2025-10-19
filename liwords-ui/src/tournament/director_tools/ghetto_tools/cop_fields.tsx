// COP (Castellano O'Connor Pairings) configuration fields

import { Button, Form, Input, InputNumber } from "antd";
import React, { useState } from "react";
import { Modal } from "../../../utils/focus_modal";
import {
  PairingMethod,
  RoundControl,
} from "../../../gen/api/proto/ipc/tournament_pb";
import { HelptipLabel } from "../helptip_label";

export type SingleRdCtrlFieldsProps = {
  setting: RoundControl;
  onChange: (
    fieldName: keyof RoundControl,
    value: string | number | boolean | PairingMethod | number[] | string[],
  ) => void;
  // Optional props for COP validation (second half only)
  beginRound?: number;
  endRound?: number;
  totalRounds?: number;
};

// Get default COP values
export const getCOPDefaults = (totalRounds: number) => {
  return {
    gibsonSpreads: [250, 200],
    hopefulnessThresholds: [0.1, 0.1],
    placePrizes: 4,
    controlLossActivationRound: Math.max(totalRounds - 3, 1) - 1, // 0-indexed for backend (for 16-rd tournament: 16-3 = 13 (display), -1 = 12 (backend))
    divisionSims: 100000,
    controlLossSims: 10000,
    controlLossThreshold: 0.25,
  };
};

export const COPRoundControlFields = (props: SingleRdCtrlFieldsProps) => {
  const { setting, totalRounds } = props;
  const [showCOPDetails, setShowCOPDetails] = useState(false);
  const [touchedFields, setTouchedFields] = useState<Set<string>>(new Set());

  // Get defaults
  const defaults = React.useMemo(() => {
    return totalRounds !== undefined ? getCOPDefaults(totalRounds) : null;
  }, [totalRounds]);

  // Wrapper to mark fields as touched and call parent onChange
  const handleFieldChange = (
    fieldName: keyof RoundControl,
    value: string | number | boolean | PairingMethod | number[] | string[],
  ) => {
    setTouchedFields((prev) => new Set(prev).add(fieldName));
    props.onChange(fieldName, value);
  };

  // If field is touched, show actual value (even if 0). Otherwise show default.
  const displayPlacePrizes = touchedFields.has("placePrizes")
    ? setting.placePrizes
    : setting.placePrizes || defaults?.placePrizes || 4;
  const displayControlLossActivationRound = touchedFields.has(
    "controlLossActivationRound",
  )
    ? setting.controlLossActivationRound
    : setting.controlLossActivationRound ||
      defaults?.controlLossActivationRound ||
      0;
  const displayDivisionSims = touchedFields.has("divisionSims")
    ? setting.divisionSims
    : setting.divisionSims || defaults?.divisionSims || 100000;
  const displayControlLossSims = touchedFields.has("controlLossSims")
    ? setting.controlLossSims
    : setting.controlLossSims || defaults?.controlLossSims || 10000;
  const displayControlLossThreshold = touchedFields.has("controlLossThreshold")
    ? setting.controlLossThreshold
    : setting.controlLossThreshold || defaults?.controlLossThreshold || 0.25;

  const displayGibsonSpreads =
    setting.gibsonSpreads && setting.gibsonSpreads.length > 0
      ? setting.gibsonSpreads.join(", ")
      : defaults?.gibsonSpreads.join(", ") || "250, 200";

  const displayHopefulnessThresholds =
    setting.hopefulnessThresholds && setting.hopefulnessThresholds.length > 0
      ? setting.hopefulnessThresholds.join(", ")
      : defaults?.hopefulnessThresholds.join(", ") || "0.1, 0.1";

  return (
    <div
      className="cop-config-panel"
      style={{
        padding: "16px",
        borderRadius: "4px",
        border: "1px solid var(--color-border)",
      }}
    >
      <h4>COP Configuration</h4>
      <p style={{ fontSize: "12px", marginBottom: "16px" }}>
        COP (Castellano O'Connor Pairings) is an advanced pairing algorithm
        designed for the second half of tournaments.
        <a
          href="#"
          onClick={(e) => {
            e.preventDefault();
            setShowCOPDetails(true);
          }}
          style={{ textDecoration: "underline" }}
        >
          Read more
        </a>
      </p>

      <Modal
        title="About COP (Castellano O'Connor Pairings)"
        open={showCOPDetails}
        onCancel={() => setShowCOPDetails(false)}
        footer={[
          <Button
            key="close"
            type="primary"
            onClick={() => setShowCOPDetails(false)}
          >
            Close
          </Button>,
        ]}
        width={700}
      >
        <div style={{ fontSize: "14px", lineHeight: "1.6" }}>
          <p style={{ marginBottom: "16px" }}>
            Castellano O'Connor Pairings (COP) is an automated tournament
            Scrabble pairing system that replaces slow, manual late-round
            decisions with data-driven, minimum-weight matching. It simulates
            many possible futures of the event—without using player ratings—to
            estimate who still has a realistic shot at prizes ("contenders") and
            to detect when a player risks losing control of their destiny.
            Simulations start with "factor" pairings (e.g., with 3 rounds left:
            1v4, 2v5, 3v6, then 1v3/2v4, then KOTH), then tighten those bounds
            based on who actually reaches first in the trials. COP can also run
            "control loss" simulations to ensure pivotal challengers meet the
            right opponents.
          </p>
          <p style={{ marginBottom: "16px" }}>
            COP's policies turn those insights into constraints and weights.
            Constraints enforce things like: preserving any pre-set pairings;
            KOTH among contenders in the final round (with a top noncontender
            added if needed); special handling for class prizes; control-loss
            matchups late in events; Gibson group separation and Gibson byes;
            and optional top-down bye assignment. Weights act as penalties the
            matcher tries to minimize: major (never pair contenders with
            noncontenders or with a Gibsonized player; avoid repeat byes), minor
            (avoid back-to-back repeats for noncontenders), and normal (prefer
            small rank gaps—especially contender vs contender where the
            lower-ranked player can still catch up—and penalize repeat
            pairings). The result is fast, equitable, low-drama pairings for the
            rest of the tournament.
          </p>
          <p style={{ marginBottom: "16px" }}>
            The default values are generally good for most tournaments, but you
            should carefully review the <strong>Place Prizes</strong> and{" "}
            <strong>Control Loss Activation Round</strong> settings to ensure
            they match your tournament structure.
          </p>
          <p>
            To read more, take a look at{" "}
            <a
              href="https://docs.google.com/document/d/1JNCbaesdBMGYtka3ZajfGf62zIzjnKEG-QBQ6_sEuBo/"
              target="_blank"
              rel="noopener noreferrer"
              style={{ textDecoration: "underline" }}
            >
              this document
            </a>
            .
          </p>
        </div>
      </Modal>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Place Prizes"
            help="Number of places that receive prizes. Used to determine who is in contention."
          />
        }
      >
        <InputNumber
          inputMode="numeric"
          min={1}
          value={displayPlacePrizes}
          onChange={(v) => handleFieldChange("placePrizes", v as number)}
        />
      </Form.Item>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Control Loss Activation Round"
            help="Round at which control loss simulation activates (displayed as 1-indexed, sent as 0-indexed to backend). Typically set to total_rounds - 3 for the last 4 rounds (e.g., round 13 for a 16-round tournament)."
          />
        }
      >
        <InputNumber
          inputMode="numeric"
          min={1}
          value={displayControlLossActivationRound + 1}
          onChange={(v) =>
            handleFieldChange("controlLossActivationRound", (v as number) - 1)
          }
        />
      </Form.Item>

      <hr
        style={{
          margin: "24px 0",
          border: "none",
          borderTop: "1px solid var(--color-border)",
        }}
      />
      <h5
        style={{
          marginBottom: "16px",
          fontSize: "13px",
          fontWeight: 600,
          color: "var(--color-text-secondary)",
        }}
      >
        Advanced Settings (rarely need adjustment)
      </h5>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Gibson Spreads"
            help="Comma-separated values (ordered from last round to first). If fewer values are provided than rounds, the last value will be repeated. Example: 250,200 means the last round has a Gibson threshold of 250 points (spread between players), then every round from the penultimate and back has a threshold of an additional 200 points."
          />
        }
      >
        <Input
          defaultValue={displayGibsonSpreads}
          onChange={(e) => {
            const values = e.target.value
              .split(",")
              .map((v) => parseInt(v.trim(), 10))
              .filter((v) => !isNaN(v));
            if (values.length > 0) {
              handleFieldChange("gibsonSpreads", values);
            }
          }}
          placeholder="250, 200"
        />
      </Form.Item>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Hopefulness Thresholds"
            help="Comma-separated decimal values (ordered from last round to first). If fewer values are provided than rounds, the last value will be repeated. Example: 0.1, 0.1"
          />
        }
      >
        <Input
          defaultValue={displayHopefulnessThresholds}
          onChange={(e) => {
            const values = e.target.value
              .split(",")
              .map((v) => parseFloat(v.trim()))
              .filter((v) => !isNaN(v));
            if (values.length > 0) {
              handleFieldChange("hopefulnessThresholds", values);
            }
          }}
          placeholder="0.1, 0.1"
        />
      </Form.Item>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Control Loss Threshold"
            help="Probability threshold for control loss scenarios. Typical value: 0.25. Must be greater than 0."
          />
        }
      >
        <InputNumber
          inputMode="numeric"
          step={0.01}
          min={0.01}
          max={1}
          value={displayControlLossThreshold}
          onChange={(v) =>
            handleFieldChange("controlLossThreshold", v as number)
          }
        />
      </Form.Item>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Division Simulations"
            help="Number of Monte Carlo simulations for division outcomes. Typical value: 100000"
          />
        }
      >
        <InputNumber
          inputMode="numeric"
          min={1000}
          step={1000}
          value={displayDivisionSims}
          onChange={(v) => handleFieldChange("divisionSims", v as number)}
        />
      </Form.Item>

      <Form.Item
        label={
          <HelptipLabel
            labelText="Control Loss Simulations"
            help="Number of simulations for control loss scenarios. Typical value: 10000"
          />
        }
      >
        <InputNumber
          inputMode="numeric"
          min={1000}
          step={1000}
          value={displayControlLossSims}
          onChange={(v) => handleFieldChange("controlLossSims", v as number)}
        />
      </Form.Item>
    </div>
  );
};
