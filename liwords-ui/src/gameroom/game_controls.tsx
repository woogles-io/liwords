import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { useHistory } from 'react-router-dom';
import { Button, Dropdown, Menu, Modal, message, Popconfirm } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';

import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  DownOutlined,
  ExclamationCircleOutlined,
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
  const initiallyFocusHere = useCallback((elt) => {
    elt?.focus();
  }, []);
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
      <Button onClick={handleExamineEnd} ref={initiallyFocusHere}>
        Done
      </Button>
    </div>
  );
});

type OptionsMenuProps = {
  handleOptionsClick: (e: MenuInfo) => void;
  hideMe: (e: React.MouseEvent<HTMLElement>) => void;
  showAbort: boolean;
  showNudge: boolean;
  darkMode: boolean;
};

const OptionsGameMenu = (props: OptionsMenuProps) => (
  <Menu
    onClick={props.handleOptionsClick}
    onMouseLeave={props.hideMe}
    theme={props.darkMode ? 'dark' : 'light'}
  >
    <Menu.Item key="resign">Resign</Menu.Item>
    {props.showAbort && <Menu.Item key="abort">Cancel game</Menu.Item>}
    {props.showNudge && <Menu.Item key="nudge">Nudge</Menu.Item>}
  </Menu>
);

export type Props = {
  isExamining: boolean;
  exchangeAllowed?: boolean;
  finalPassOrChallenge?: boolean;
  myTurn?: boolean;
  observer?: boolean;
  showExchangeModal: () => void;
  onPass: () => void;
  onResign: () => void;
  onRequestAbort: () => void;
  onNudge: () => void;
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
  showNudge: boolean;
  showAbort: boolean;
};

const GameControls = React.memo((props: Props) => {
  const { useState } = useMountedState();

  // Poka-yoke against accidentally having multiple pop-ups active.
  const [actualCurrentPopUp, setCurrentPopUp] = useState<
    'NONE' | 'CHALLENGE' | 'PASS'
  >('NONE');
  const [optionsMenuVisible, setOptionsMenuVisible] = useState(false);
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

  const darkMode = useMemo(
    () => localStorage?.getItem('darkMode') === 'true',
    []
  );

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

  const [optionsMenuId, setOptionsMenuId] = useState(0);
  useEffect(() => {
    if (!optionsMenuVisible) {
      // when the menu is hidden, yeet it and replace with a new instance altogether.
      // this works around old items being selected when reopening the menu.
      setOptionsMenuId((n) => (n + 1) | 0);
    }
  }, [optionsMenuVisible]);

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

  const optionsMenu = (
    <OptionsGameMenu
      key={optionsMenuId}
      showAbort={props.showAbort}
      showNudge={props.showNudge}
      hideMe={(e) => {
        setOptionsMenuVisible(false);
      }}
      handleOptionsClick={(e) => {
        setOptionsMenuVisible(false);
        switch (e.key) {
          case 'resign':
            Modal.confirm({
              title: <p>Are you sure you wish to resign?</p>,
              icon: <ExclamationCircleOutlined />,
              // XXX: what if it's unrated?
              content: <p>Your rating will be maximally affected.</p>,
              onOk() {
                props.onResign();
              },
            });
            break;
          case 'abort':
            Modal.confirm({
              title: <p>Request an abort</p>,
              icon: <ExclamationCircleOutlined />,
              content: <p>This will request an abort from your opponent.</p>,
              onOk() {
                props.onRequestAbort();
              },
            });
            break;
          case 'nudge':
            Modal.confirm({
              title: <p>Nudge your opponent</p>,
              icon: <ExclamationCircleOutlined />,
              content: (
                <p>
                  Clicking OK will send a nudge to your opponent. If they do not
                  respond, the game will be adjudicated in your favor.
                </p>
              ),
              onOk() {
                props.onNudge();
              },
            });
            break;
        }
      }}
      darkMode={darkMode}
    />
  );

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Dropdown
          overlay={optionsMenu}
          trigger={['click']}
          visible={optionsMenuVisible}
        >
          <Button onClick={() => setOptionsMenuVisible((v) => !v)}>
            Options <DownOutlined />
          </Button>
        </Dropdown>

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
  const [rematchDisabled, setRematchDisabled] = useState(false);

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
      {props.showRematch && !props.tournamentPairedMode && !rematchDisabled && (
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
