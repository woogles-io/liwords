import { InputNumber, Modal } from "antd";
import { useEffect, useState } from "react";

// Mirrors maxTimePenalty in pkg/cwgame.
export const maxTimePenalty = 1000;

export type EndOfGamePlayer = {
  name: string;
  score: number;
};

type Props = {
  open: boolean;
  players: EndOfGamePlayer[];
  onClose: () => void;
  // Called once per player with a penalty, in player order.
  onSubmit: (playerIndex: number, points: number) => Promise<void> | void;
};

// Shown when an annotated game ends: the final scores, and an optional
// over-time penalty for either or both players.
export const EndOfGameModal = (props: Props) => {
  const [penalties, setPenalties] = useState<Array<number | null>>([]);
  const [submitting, setSubmitting] = useState(false);

  const { open, players } = props;
  useEffect(() => {
    if (open) {
      setPenalties(players.map(() => null));
    }
  }, [open, players.length]); // eslint-disable-line react-hooks/exhaustive-deps

  const hasPenalty = penalties.some((p) => !!p);

  const submit = async () => {
    setSubmitting(true);
    try {
      for (let idx = 0; idx < penalties.length; idx++) {
        const points = penalties[idx];
        if (points) {
          await props.onSubmit(idx, points);
        }
      }
      props.onClose();
    } catch {
      // The caller has already reported the error; stay open for a retry.
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="Game over"
      className="end-of-game-modal"
      open={open}
      onCancel={props.onClose}
      onOk={submit}
      okText="Apply time penalties"
      okButtonProps={{
        disabled: !hasPenalty,
        loading: submitting,
      }}
      cancelText="Close"
    >
      <table className="end-of-game-scores">
        <thead>
          <tr>
            <th>Player</th>
            <th>Score</th>
            <th>Time penalty</th>
          </tr>
        </thead>
        <tbody>
          {players.map((p, idx) => (
            <tr key={idx}>
              <td>{p.name}</td>
              <td>{p.score}</td>
              <td>
                <InputNumber
                  aria-label={`Time penalty for ${p.name}`}
                  min={1}
                  max={maxTimePenalty}
                  step={10}
                  precision={0}
                  placeholder="0"
                  value={penalties[idx] ?? null}
                  onChange={(v) =>
                    setPenalties((prev) =>
                      prev.map((old, i) => (i === idx ? v : old)),
                    )
                  }
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Modal>
  );
};
