import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Button, Popconfirm } from 'antd';
import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import {
  useExamineStoreContext,
  useGameContextStoreContext,
  useResetStoreContext,
} from '../store/store';

const colors = require('../base.scss');

const ExamineGameControls = React.memo((props: {}) => {
  const {
    examinedTurn,
    handleExamineEnd,
    handleExamineFirst,
    handleExaminePrev,
    handleExamineNext,
    handleExamineLast,
  } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const numberOfTurns = gameContext.turns.length;

  return (
    <div className="game-controls">
      <Button>Options</Button>
      <Button
        shape="circle"
        icon={<DoubleLeftOutlined />}
        type="primary"
        onClick={handleExamineFirst}
        disabled={examinedTurn <= 0}
      />
      <Button
        shape="circle"
        icon={<LeftOutlined />}
        type="primary"
        onClick={handleExaminePrev}
        disabled={examinedTurn <= 0}
      />
      <Button
        shape="circle"
        icon={<RightOutlined />}
        type="primary"
        onClick={handleExamineNext}
        disabled={examinedTurn >= numberOfTurns}
      />
      <Button
        shape="circle"
        icon={<DoubleRightOutlined />}
        type="primary"
        onClick={handleExamineLast}
        disabled={examinedTurn >= numberOfTurns}
      />
      <Button onClick={handleExamineEnd}>Done</Button>
    </div>
  );
});

export type Props = {
  isExamining: boolean;
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
  onExportGCG: () => void;
  onRematch: () => void;
  gameEndControls: boolean;
  showRematch: boolean;
  currentRack: string;
};

const GameControls = React.memo((props: Props) => {
  if (props.isExamining) {
    return <ExamineGameControls />;
  }

  if (props.gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        onExportGCG={props.onExportGCG}
        showRematch={props.showRematch && !props.observer}
      />
    );
  }

  if (props.observer) {
    return (
      <div className="game-controls">
        <Button onClick={props.onExamine}>Examine</Button>
      </div>
    );
  }

  // Temporary dead code.
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
  onExportGCG: () => void;
};

const EndGameControls = (props: EGCProps) => {
  const [rematchDisabled, setRematchDisabled] = useState(false);
  const { resetStore } = useResetStoreContext();
  const history = useHistory();
  const handleExitToLobby = React.useCallback(() => {
    resetStore();
    history.replace('/');
  }, [history, resetStore]);

  return (
    <div className="game-controls">
      <Button>Options</Button>
      <Button onClick={props.onExamine}>Examine</Button>
      <Button onClick={props.onExportGCG}>Export GCG</Button>
      <Button onClick={handleExitToLobby}>Exit</Button>
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
