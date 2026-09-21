import { Button, Form, InputNumber, Select } from "antd";
import { useState } from "react";

// Mirrors maxTimePenalty in pkg/cwgame.
export const maxTimePenalty = 1000;

type Props = {
  playerNames: string[];
  onSubmit: (playerIndex: number, points: number) => Promise<void> | void;
};

// Enter an over-time penalty for a finished annotated game.
export const TimePenaltyControl = (props: Props) => {
  const [playerIndex, setPlayerIndex] = useState(0);
  const [points, setPoints] = useState<number | null>(10);
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!points) {
      return;
    }
    setSubmitting(true);
    try {
      await props.onSubmit(playerIndex, points);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Form layout="inline" className="time-penalty-control">
      <Form.Item label="Player">
        <Select
          aria-label="Time penalty player"
          value={playerIndex}
          onChange={setPlayerIndex}
          options={props.playerNames.map((name, idx) => ({
            value: idx,
            label: name,
          }))}
          style={{ minWidth: 120 }}
        />
      </Form.Item>
      <Form.Item label="Points">
        <InputNumber
          aria-label="Time penalty points"
          min={1}
          max={maxTimePenalty}
          step={10}
          precision={0}
          value={points}
          onChange={setPoints}
        />
      </Form.Item>
      <Form.Item>
        <Button
          onClick={submit}
          disabled={!points}
          loading={submitting}
          data-testid="time-penalty-submit"
        >
          Add time penalty
        </Button>
      </Form.Item>
    </Form>
  );
};
