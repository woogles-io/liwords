import React, { useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import { Button, Popconfirm } from 'antd';
import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';
import {
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { EphemeralTile } from '../utils/cwgame/common';

const ExamineGameControls = React.memo((props: { lexicon: string }) => {
  const {
    examinedTurn,
    handleExamineEnd,
    handleExamineFirst,
    handleExaminePrev,
    handleExamineNext,
    handleExamineLast,
  } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const { setPlacedTiles, setPlacedTilesTempScore } = useTentativeTileContext();
  useEffect(() => {
    setPlacedTilesTempScore(undefined);
    setPlacedTiles(new Set<EphemeralTile>());
  }, [examinedTurn, setPlacedTiles, setPlacedTilesTempScore]);
  const numberOfTurns = gameContext.turns.length;
  return (
    <div className="game-controls">
      <Button disabled>Options</Button>
      <Button
        shape="circle"
        icon={<DoubleLeftOutlined />}
        type="primary"
        onClick={handleExamineFirst}
        disabled={examinedTurn <= 0 || numberOfTurns <= 0}
      />
      <Button
        shape="circle"
        icon={<LeftOutlined />}
        type="primary"
        onClick={handleExaminePrev}
        disabled={examinedTurn <= 0 || numberOfTurns <= 0}
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
  tournamentID?: string;
  lexicon: string;
};

const GameControls = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [passVisible, setPassVisible] = useState(false);
  const [challengeVisible, setChallengeVisible] = useState(false);
  const [resignVisible, setResignVisible] = useState(false);

  if (props.isExamining) {
    return <ExamineGameControls lexicon={props.lexicon} />;
  }

  if (props.gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        onExportGCG={props.onExportGCG}
        showRematch={props.showRematch && !props.observer}
        tournamentID={props.tournamentID}
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
      <div className="secondary-controls">
        <Popconfirm
          title="Are you sure you wish to resign?"
          onCancel={() => {
            setResignVisible(false);
          }}
          onConfirm={() => {
            props.onResign();
            setResignVisible(false);
          }}
          onVisibleChange={(visible) => {
            setResignVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={resignVisible}
        >
          <Button
            danger
            onDoubleClick={() => {
              props.onResign();
              setResignVisible(false);
            }}
          >
            Ragequit
          </Button>
        </Popconfirm>

        <Popconfirm
          title="Are you sure you wish to pass?"
          onCancel={() => {
            setPassVisible(false);
          }}
          onConfirm={() => {
            props.onPass();
            setPassVisible(false);
          }}
          onVisibleChange={(visible) => {
            setPassVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={passVisible}
        >
          <Button
            onDoubleClick={() => {
              props.onPass();
              setPassVisible(false);
            }}
            danger
            disabled={!props.myTurn}
            type={
              props.finalPassOrChallenge && props.myTurn ? 'primary' : 'default'
            }
          >
            Pass
            <span className="key-command">2</span>
          </Button>
        </Popconfirm>
      </div>
      <div className="secondary-controls">
        <Popconfirm
          title="Are you sure you wish to challenge?"
          onCancel={() => {
            setChallengeVisible(false);
          }}
          onConfirm={() => {
            props.onChallenge();
            setChallengeVisible(false);
          }}
          onVisibleChange={(visible) => {
            setChallengeVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={challengeVisible}
        >
          <Button
            onDoubleClick={() => {
              props.onChallenge();
              setChallengeVisible(false);
            }}
            disabled={!props.myTurn}
          >
            Challenge
            <span className="key-command">3</span>
          </Button>
        </Popconfirm>
        <Button
          onClick={props.showExchangeModal}
          disabled={!(props.myTurn && props.exchangeAllowed)}
        >
          Exchange
          <span className="key-command">4</span>
        </Button>
      </div>
      <Button
        type="primary"
        className="play"
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
  tournamentID?: string;
};

const EndGameControls = (props: EGCProps) => {
  const { useState } = useMountedState();

  const [rematchDisabled, setRematchDisabled] = useState(false);
  const history = useHistory();
  const handleExitToLobby = React.useCallback(() => {
    props.tournamentID
      ? history.replace(`/tournament/${props.tournamentID}`)
      : history.replace('/');
  }, [history, props.tournamentID]);

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Button disabled>Options</Button>
        <Button onClick={props.onExamine}>Examine</Button>
      </div>
      <div className="secondary-controls">
        <Button onClick={props.onExportGCG}>Export GCG</Button>
        <Button onClick={handleExitToLobby}>Exit</Button>
      </div>
      {props.showRematch && !rematchDisabled && (
        <Button
          type="primary"
          data-testid="rematch-button"
          className="play"
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
