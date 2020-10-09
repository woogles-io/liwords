import React, { useState } from 'react';
import { Button, Popconfirm } from 'antd';

export type Props = {
  exchangeAllowed?: boolean;
  finalPassOrChallenge?: boolean;
  myTurn?: boolean;
  observer?: boolean;
  showExchangeModal: () => void;
  onPass: () => void;
  onResign: () => void;
  onRecall: () => void;
  onChallenge: () => void;
  onCommit: () => void;
  onExamine: () => void;
  onRematch: () => void;
  gameEndControls: boolean;
  showRematch: boolean;
  currentRack: string;
};

const GameControls = React.memo((props: Props) => {
  if (props.gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        showRematch={props.showRematch && !props.observer}
      />
    );
  }

  if (props.observer) {
    return null;
  }

  return (
    <div className="game-controls">
      <Popconfirm
        title="Are you sure you wish to resign?"
        onConfirm={props.onResign}
        okText="Yes"
        cancelText="No"
      >
        <Button danger>Ragequit</Button>
      </Popconfirm>

      <Button
        onClick={props.onPass}
        danger
        disabled={!props.myTurn}
        type={
          props.finalPassOrChallenge && props.myTurn ? 'primary' : 'default'
        }
      >
        Pass
        <span className="key-command">2</span>
      </Button>

      <Button onClick={props.onChallenge} disabled={!props.myTurn}>
        Challenge
        <span className="key-command">3</span>
      </Button>

      <Button
        onClick={props.showExchangeModal}
        disabled={!(props.myTurn && props.exchangeAllowed)}
      >
        Exchange
        <span className="key-command">4</span>
      </Button>

      <Button
        type="primary"
        onClick={props.onCommit}
        disabled={!props.myTurn || props.finalPassOrChallenge}
      >
        Play
      </Button>
    </div>
  );
});

type EGCProps = {
  onRematch: () => void;
  showRematch: boolean;
  onExamine: () => void;
};

const EndGameControls = (props: EGCProps) => {
  const [rematchDisabled, setRematchDisabled] = useState(false);
  return (
    <div className="game-controls">
      <Button>Options</Button>
      <Button onClick={props.onExamine}>Export GCG</Button>
      <Button onClick={() => window.location.replace('/')}>Exit</Button>
      {props.showRematch && !rematchDisabled && (
        <Button
          type="primary"
          data-testid="rematch-button"
          onClick={() => {
            setRematchDisabled(true);
            if (!rematchDisabled) {
              props.onRematch();
            }
          }}
        >
          Rematch
        </Button>
      )}
    </div>
  );
};

export default GameControls;
