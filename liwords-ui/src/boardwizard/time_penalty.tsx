import { Button } from "antd";
import { useEffect, useState } from "react";
import { Modal } from "../utils/focus_modal";

// Mirrors maxTimePenalty in pkg/cwgame.
export const maxTimePenalty = 1000;
const penaltyIncrement = 10;

export type EndOfGamePlayer = {
  name: string;
  // Current score, after any time penalty already applied.
  score: number;
  // Time penalty already applied.
  penalty: number;
};

type Props = {
  open: boolean;
  // In turn order: the player who went first is shown on the left.
  players: EndOfGamePlayer[];
  onClose: () => void;
  // Sets a player's total time penalty (zero removes it). Called once per
  // player whose penalty changed, in player order.
  onSubmit: (playerIndex: number, points: number) => Promise<void> | void;
};

// Shown when an annotated game ends: the final scores, the winner, and an
// over-time penalty for either or both players.
export const EndOfGameModal = (props: Props) => {
  const [penalties, setPenalties] = useState<number[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const { open, players, onClose } = props;
  useEffect(() => {
    if (open) {
      setPenalties(players.map((p) => p.penalty));
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  const penaltyFor = (idx: number) => penalties[idx] ?? players[idx].penalty;
  const finalScores = players.map(
    (p, idx) => p.score + p.penalty - penaltyFor(idx),
  );
  const best = Math.max(...finalScores);
  const tied = finalScores.every((s) => s === best);
  const changed = players.some((p, idx) => penaltyFor(idx) !== p.penalty);

  const addPenalty = (idx: number) =>
    setPenalties((prev) =>
      prev.map((p, i) =>
        i === idx ? Math.min(p + penaltyIncrement, maxTimePenalty) : p,
      ),
    );

  const clearPenalty = (idx: number) =>
    setPenalties((prev) => prev.map((p, i) => (i === idx ? 0 : p)));

  const submit = async () => {
    setSubmitting(true);
    try {
      for (let idx = 0; idx < players.length; idx++) {
        if (penaltyFor(idx) !== players[idx].penalty) {
          await props.onSubmit(idx, penaltyFor(idx));
        }
      }
      onClose();
    } catch {
      // The caller has already reported the error; stay open for a retry.
    } finally {
      setSubmitting(false);
    }
  };

  const column = (idx: number) => (idx === 0 ? 1 : 3);

  return (
    <Modal
      className="end-of-game"
      title="Game over"
      open={open}
      onCancel={onClose}
      width={320}
      styles={{ mask: { backgroundColor: "rgba(0, 0, 0, 0.15)" } }}
      footer={
        <div style={{ display: "flex", justifyContent: "flex-end" }}>
          <Button onClick={onClose}>Close</Button>
          <Button
            type="primary"
            onClick={submit}
            disabled={!changed}
            loading={submitting}
            style={{ marginLeft: "8px" }}
          >
            Apply
          </Button>
        </div>
      }
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr auto 1fr",
          alignItems: "center",
          justifyItems: "center",
          rowGap: "2px",
          columnGap: "12px",
        }}
      >
        {players.map((p, idx) => (
          <div
            key={`name-${idx}`}
            style={{
              gridColumn: column(idx),
              gridRow: 1,
              fontSize: "14px",
              color: "var(--text-secondary)",
            }}
          >
            {p.name}
          </div>
        ))}
        {players.map((_, idx) => (
          <div
            key={`score-${idx}`}
            style={{
              gridColumn: column(idx),
              gridRow: 2,
              fontSize: "24px",
              fontWeight: "bold",
            }}
          >
            {finalScores[idx]}
          </div>
        ))}
        {players.map((_, idx) => (
          <div
            key={`result-${idx}`}
            style={{
              gridColumn: column(idx),
              gridRow: 3,
              minHeight: "20px",
              marginTop: "-3px",
              marginBottom: "3px",
              fontSize: "12px",
              fontWeight: "bold",
              color: "var(--success-color)",
            }}
          >
            {tied ? "TIE" : finalScores[idx] === best ? "WINNER!" : ""}
          </div>
        ))}
        <div
          style={{
            gridColumn: 2,
            gridRow: 4,
            fontSize: "12px",
            color: "var(--text-secondary)",
          }}
        >
          Add Time Penalty
        </div>
        {players.map((p, idx) => (
          <div
            key={`penalty-${idx}`}
            style={{ gridColumn: column(idx), gridRow: 4 }}
          >
            <Button
              size="small"
              aria-label={`Time penalty for ${p.name}`}
              onClick={() => addPenalty(idx)}
              disabled={penaltyFor(idx) >= maxTimePenalty}
            >
              -{penaltyFor(idx) + penaltyIncrement}
            </Button>
          </div>
        ))}
        {players.map(
          (p, idx) =>
            penaltyFor(idx) > 0 && (
              <Button
                key={`clear-${idx}`}
                type="link"
                size="small"
                aria-label={`Clear time penalty for ${p.name}`}
                onClick={() => clearPenalty(idx)}
                style={{ gridColumn: column(idx), gridRow: 5, height: "auto" }}
              >
                clear
              </Button>
            ),
        )}
      </div>
    </Modal>
  );
};
