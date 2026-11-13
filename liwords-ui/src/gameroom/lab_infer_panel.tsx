import React, { useEffect, useState } from "react";
import { InputNumber } from "antd";
import { machineLetterToRune } from "../constants/alphabets";
import { labCall } from "../wasm/lab";
import { getLoadableFiles } from "../wasm/loader";
import { wolgesLetterToLiwordsLetter } from "./analyzer";
import {
  ColumnName,
  DEFAULT_FUZZ,
  LabPosition,
  NOTHING_TO_INFER,
  PanelFrame,
  inferRequest,
} from "./analyzer_lab_panels";

// How many of the likeliest leaves are listed until the user asks for others.
const DEFAULT_LEAVES_LISTED = 10;

// What the inference finds: how many leaves the opponent may have kept, the
// likeliest of them, and by tile (0 the blank) the chance their rack holds one,
// as inferred (held) and with nothing inferred (prior).
export type Inference = {
  count: number;
  leaves: Array<{ tiles: Array<number>; weight: number }>;
  tiles: Array<{ held: number; prior: number }>;
};

type InferPanelProps = {
  lab: LabPosition;
  onClose: () => void;
};

const percent = (x: number) => `${(x * 100).toFixed(1)}%`;

// The tiles the opponent may hold, the ones the inference makes likeliest
// first: each with its chance as inferred and by chance alone. A tile none of
// which is unseen is left out.
export const inferenceRows = (tiles: Inference["tiles"]) =>
  tiles
    .map((t, tile) => ({ tile, ...t }))
    .filter((t) => t.prior > 0)
    .sort((a, b) => b.held - b.prior - (a.held - a.prior) || a.tile - b.tile);

// What the opponent likely kept, from their last move, on its own: the tiles
// the inference makes them likelier or less likely to hold, and the leaves
// they likeliest kept.
export const InferPanel = (props: InferPanelProps) => {
  const { lab, onClose } = props;
  const [fuzz, setFuzz] = useState<number | null>(DEFAULT_FUZZ);
  const [listed, setListed] = useState<number | null>(DEFAULT_LEAVES_LISTED);
  const [found, setFound] = useState<
    { key: string; inference?: Inference; error?: string } | undefined
  >();

  const request =
    fuzz === null || listed === null
      ? undefined
      : inferRequest(lab, { fuzz, list: listed });
  const key = request && `${lab.loadableKey} ${request}`;

  useEffect(() => {
    if (!request || !key) return;
    let current = true;
    (async () => {
      try {
        await labCall({
          op: "prepare",
          args: { files: getLoadableFiles(lab.loadableKey) },
        });
        const inference = await labCall<Inference>({
          op: "infer",
          args: { request },
        });
        if (current) setFound({ key, inference });
      } catch (e) {
        if (current) {
          setFound({ key, error: e instanceof Error ? e.message : String(e) });
        }
      }
    })();
    return () => {
      current = false;
    };
    // the request follows from the key
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  const shown = found && found.key === key ? found : undefined;
  const inference = shown?.inference;
  const rune = (tile: number) =>
    machineLetterToRune(
      wolgesLetterToLiwordsLetter(tile),
      lab.alphabet,
      false,
      true,
    );
  const status = !lab.lastMove
    ? `Nothing to infer: ${NOTHING_TO_INFER}.`
    : !request
      ? "Set the fuzz and the leaves listed."
      : !shown
        ? "Inferring..."
        : shown.error
          ? `Could not infer: ${shown.error}`
          : `From their last move, ${inference!.count} ${inference!.count === 1 ? "leave" : "leaves"} they may have kept${
              inference!.leaves.length
                ? `; the likeliest: ${inference!.leaves
                    .map(
                      (l) =>
                        `${l.tiles.map(rune).join("") || "(none)"} ${percent(l.weight)}`,
                    )
                    .join(", ")}`
                : ""
            }.`;

  return (
    <PanelFrame
      title="Inference"
      status={status}
      running={false}
      canResume={false}
      stop={() => {}}
      resume={() => {}}
      onClose={onClose}
    >
      <div className="lab-infer-settings">
        <label>
          Fuzz
          <InputNumber
            size="small"
            aria-label="Fuzz"
            min={0}
            value={fuzz}
            onChange={(v) => setFuzz(v)}
          />
        </label>
        <label>
          Leaves listed
          <InputNumber
            size="small"
            aria-label="Leaves listed"
            min={0}
            precision={0}
            value={listed}
            onChange={(v) => setListed(v)}
          />
        </label>
      </div>
      {inference && (
        <table className="lab-infer-tiles">
          <thead>
            <tr>
              <th>
                <ColumnName name="Tile" tip="A tile the opponent may hold" />
              </th>
              <th>
                <ColumnName
                  name="Held"
                  tip="The chance their rack holds one, as inferred from their last move"
                />
              </th>
              <th>
                <ColumnName
                  name="By chance"
                  tip="The chance their rack holds one, dealt from the tiles unseen alone"
                />
              </th>
            </tr>
          </thead>
          <tbody>
            {inferenceRows(inference.tiles).map((t) => (
              <tr key={t.tile}>
                <td>{rune(t.tile)}</td>
                <td className="lab-n">{percent(t.held)}</td>
                <td className="lab-n">{percent(t.prior)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </PanelFrame>
  );
};
