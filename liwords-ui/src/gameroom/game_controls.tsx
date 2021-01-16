import React, { useCallback, useEffect, useRef } from 'react';
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
import { ChallengeRule } from './game_info';

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
  tournamentSlug?: string;
  lexicon: string;
  challengeRule: ChallengeRule;
  setHandlePassShortcut: ((handler: (() => void) | null) => void) | null;
  setHandleChallengeShortcut: ((handler: (() => void) | null) => void) | null;
  setHandleNeitherShortcut: ((handler: (() => void) | null) => void) | null;
  tournamentPairedMode?: boolean;
};

const GameControls = React.memo((props: Props) => {
  const { useState } = useMountedState();

  // Poka-yoke against accidentally having multiple pop-ups active.
  const [actualCurrentPopUp, setCurrentPopUp] = useState<
    'NONE' | 'CHALLENGE' | 'PASS' | 'RESIGN'
  >('NONE');
  // This should match disabled= and/or hidden= props.
  const currentPopUp =
    (actualCurrentPopUp === 'CHALLENGE' &&
      (!props.myTurn || props.challengeRule === 'VOID')) ||
    (actualCurrentPopUp === 'PASS' && !props.myTurn)
      ? 'NONE'
      : actualCurrentPopUp;
  useEffect(() => {
    if (currentPopUp !== actualCurrentPopUp) {
      // Although this will still take another render, make this render correct
      // to avoid temporarily showing confirmation dialog.
      setCurrentPopUp(currentPopUp);
    }
  }, [currentPopUp, actualCurrentPopUp]);

  const passButton = useRef<HTMLElement>(null);
  const challengeButton = useRef<HTMLElement>(null);

  const history = useHistory();
  const handleExitToLobby = useCallback(() => {
    props.tournamentSlug
      ? history.replace(props.tournamentSlug)
      : history.replace('/');
  }, [history, props.tournamentSlug]);

  const {
    isExamining,
    gameEndControls,
    observer,
    onChallenge,
    onPass,
    setHandlePassShortcut,
    setHandleChallengeShortcut,
    setHandleNeitherShortcut,
  } = props;
  const hasRegularButtons = !(isExamining || gameEndControls || observer);
  const handlePassShortcut = useCallback(() => {
    if (!hasRegularButtons) return;
    if (!passButton.current) return;
    passButton.current.focus();
    setCurrentPopUp((v) => {
      if (v !== 'PASS') return 'PASS';
      if (onPass) onPass();
      return 'NONE';
    });
  }, [hasRegularButtons, onPass]);
  const handleChallengeShortcut = useCallback(() => {
    if (!hasRegularButtons) return;
    if (!challengeButton.current) return;
    challengeButton.current.focus();
    setCurrentPopUp((v) => {
      if (v !== 'CHALLENGE') return 'CHALLENGE';
      if (onChallenge) onChallenge();
      return 'NONE';
    });
  }, [hasRegularButtons, onChallenge]);
  const handleNeitherShortcut = useCallback(() => {
    if (!hasRegularButtons) return;
    setCurrentPopUp('NONE');
  }, [hasRegularButtons]);
  useEffect(() => {
    if (!setHandlePassShortcut) return;
    return () => {
      setHandlePassShortcut(null);
    };
  }, [setHandlePassShortcut]);
  useEffect(() => {
    if (!setHandleChallengeShortcut) return;
    return () => {
      setHandleChallengeShortcut(null);
    };
  }, [setHandleChallengeShortcut]);
  useEffect(() => {
    if (!setHandleNeitherShortcut) return;
    return () => {
      setHandleNeitherShortcut(null);
    };
  }, [setHandleNeitherShortcut]);
  useEffect(() => {
    if (!setHandlePassShortcut) return;
    setHandlePassShortcut(() => handlePassShortcut);
  }, [handlePassShortcut, setHandlePassShortcut]);
  useEffect(() => {
    if (!setHandleChallengeShortcut) return;
    setHandleChallengeShortcut(() => handleChallengeShortcut);
  }, [handleChallengeShortcut, setHandleChallengeShortcut]);
  useEffect(() => {
    if (!setHandleNeitherShortcut) return;
    setHandleNeitherShortcut(() => handleNeitherShortcut);
  }, [handleNeitherShortcut, setHandleNeitherShortcut]);

  if (isExamining) {
    return <ExamineGameControls lexicon={props.lexicon} />;
  }

  if (gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        onExportGCG={props.onExportGCG}
        showRematch={props.showRematch && !props.observer}
        tournamentPairedMode={props.tournamentPairedMode}
        onExit={handleExitToLobby}
      />
    );
  }

  if (observer) {
    return (
      <div className="game-controls">
        <Button onClick={props.onExamine}>Examine</Button>
        <Button onClick={handleExitToLobby}>Exit</Button>
      </div>
    );
  }

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Popconfirm
          title="Are you sure you wish to resign?"
          onCancel={() => {
            setCurrentPopUp('NONE');
          }}
          onConfirm={() => {
            props.onResign();
            setCurrentPopUp('NONE');
          }}
          onVisibleChange={(visible) => {
            setCurrentPopUp(visible ? 'RESIGN' : 'NONE');
          }}
          okText="Yes"
          cancelText="No"
          visible={currentPopUp === 'RESIGN'}
        >
          <Button
            danger
            onClick={() => {
              if (currentPopUp === 'RESIGN') {
                props.onResign();
                setCurrentPopUp('NONE');
              }
            }}
          >
            Resign
          </Button>
        </Popconfirm>

        <Popconfirm
          title="Are you sure you wish to pass?"
          onCancel={() => {
            setCurrentPopUp('NONE');
          }}
          onConfirm={() => {
            props.onPass();
            setCurrentPopUp('NONE');
          }}
          onVisibleChange={(visible) => {
            setCurrentPopUp(visible ? 'PASS' : 'NONE');
          }}
          okText="Yes"
          cancelText="No"
          visible={currentPopUp === 'PASS'}
        >
          <Button
            ref={passButton}
            onClick={() => {
              if (currentPopUp === 'PASS') {
                props.onPass();
                setCurrentPopUp('NONE');
              }
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
            setCurrentPopUp('NONE');
          }}
          onConfirm={() => {
            props.onChallenge();
            setCurrentPopUp('NONE');
          }}
          onVisibleChange={(visible) => {
            setCurrentPopUp(visible ? 'CHALLENGE' : 'NONE');
          }}
          okText="Yes"
          cancelText="No"
          visible={currentPopUp === 'CHALLENGE'}
        >
          <Button
            ref={challengeButton}
            onClick={() => {
              if (currentPopUp === 'CHALLENGE') {
                props.onChallenge();
                setCurrentPopUp('NONE');
              }
            }}
            disabled={!props.myTurn}
            hidden={props.challengeRule === 'VOID'}
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
  onExit: () => void;
  tournamentPairedMode?: boolean;
};

const EndGameControls = (props: EGCProps) => {
  const { useState } = useMountedState();

  const [rematchDisabled, setRematchDisabled] = useState(
    props.tournamentPairedMode
  );

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Button disabled>Options</Button>
        <Button onClick={props.onExamine}>Examine</Button>
      </div>
      <div className="secondary-controls">
        <Button onClick={props.onExportGCG}>Export GCG</Button>
        <Button onClick={props.onExit}>Exit</Button>
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
