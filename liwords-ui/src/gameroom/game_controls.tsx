import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Affix, Button, Dropdown, Menu, Modal, Popconfirm } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';

import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  ExclamationCircleOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { EphemeralTile } from '../utils/cwgame/common';
import { ChallengeRule } from '../gen/api/proto/macondo/macondo_pb';

const downloadGameImg = (downloadFilename: string) => {
  const link = document.createElement('a');
  link.href = new URL(
    `/gameimg/${encodeURIComponent(downloadFilename)}`,
    window.location.href
  ).href;
  link.setAttribute('download', downloadFilename);
  document.body.appendChild(link);
  link.onclick = () => {
    link.remove();
  };
  link.click();
};

const ExamineGameControls = React.memo(
  (props: {
    lexicon: string;
    darkMode: boolean;
    onExportGCG: () => void;
    gameDone: boolean;
    puzzleMode: boolean;
    exitable: boolean;
    editMode: boolean;
  }) => {
    const { useState } = useMountedState();
    const { gameContext: examinableGameContext } =
      useExaminableGameContextStoreContext();
    const {
      examinedTurn,
      handleExamineEnd,
      handleExamineFirst,
      handleExaminePrev,
      handleExamineNext,
      handleExamineLast,
      doneButtonRef,
    } = useExamineStoreContext();
    const { gameContext } = useGameContextStoreContext();
    const { setPlacedTiles, setPlacedTilesTempScore } =
      useTentativeTileContext();
    useEffect(() => {
      setPlacedTilesTempScore(undefined);
      setPlacedTiles(new Set<EphemeralTile>());
    }, [examinedTurn, setPlacedTiles, setPlacedTilesTempScore]);
    useEffect(() => {
      doneButtonRef.current?.focus();
    }, [doneButtonRef]);
    const numberOfTurns = gameContext.turns.length;
    const gameHasNotStarted = gameContext.players.length === 0; // :shrug:
    const gameDone = props.gameDone;
    const isAtLastTurn =
      gameDone &&
      examinableGameContext.turns.length === gameContext.turns.length;

    const [exportMenuVisible, setExportMenuVisible] = useState(false);
    const [exportMenuId, setExportMenuId] = useState(0);
    useEffect(() => {
      if (!exportMenuVisible) {
        // when the menu is hidden, yeet it and replace with a new instance altogether.
        // this works around old items being selected when reopening the menu.
        setExportMenuId((n) => (n + 1) | 0);
      }
    }, [exportMenuVisible]);
    const exportMenu = (
      <Menu
        key={exportMenuId}
        onClick={(e) => {
          setExportMenuVisible(false);
          // When at the last move, examineStoreContext.examinedTurn === Infinity.
          // To also detect new moves, we use examinableGameContext.turns.length.
          switch (e.key) {
            case 'download-png':
              downloadGameImg(
                `${gameContext.gameID}${gameDone ? '-v2' : ''}.png`
              );
              break;
            case 'download-png-turn':
              downloadGameImg(
                `${gameContext.gameID}${gameDone ? '-v2' : ''}-${
                  examinableGameContext.turns.length + 1
                }.png`
              );
              break;
            case 'download-animated-gif-turn':
              downloadGameImg(
                `${gameContext.gameID}${gameDone ? '-v2-b' : '-a'}-${
                  examinableGameContext.turns.length + 1
                }.gif`
              );
              break;
            case 'download-animated-gif':
              downloadGameImg(
                `${gameContext.gameID}${gameDone ? '-v2-b' : '-a'}.gif`
              );
              break;
          }
        }}
        onMouseLeave={(e) => {
          setExportMenuVisible(false);
        }}
        theme={props.darkMode ? 'dark' : 'light'}
      >
        {isAtLastTurn && (
          <Menu.Item key="download-png" disabled={gameHasNotStarted}>
            PNG
          </Menu.Item>
        )}
        {!isAtLastTurn && (
          <Menu.Item key="download-png-turn" disabled={gameHasNotStarted}>
            PNG
          </Menu.Item>
        )}
        {!isAtLastTurn && (
          <Menu.Item
            key="download-animated-gif-turn"
            disabled={gameHasNotStarted}
          >
            Animated GIF to this position
          </Menu.Item>
        )}
        {gameDone && (
          <Menu.Item key="download-animated-gif" disabled={gameHasNotStarted}>
            Animated GIF of complete game
          </Menu.Item>
        )}
        {(gameDone || props.editMode) && (
          <Menu.Item
            key="download-gcg"
            disabled={gameHasNotStarted}
            onClick={props.onExportGCG}
          >
            GCG
          </Menu.Item>
        )}
      </Menu>
    );

    return (
      <Affix offsetTop={210} className="examiner-controls">
        <div className="game-controls">
          <Dropdown
            overlay={exportMenu}
            trigger={['click']}
            visible={exportMenuVisible}
            placement="topLeft"
            disabled={props.puzzleMode}
          >
            <Button onClick={() => setExportMenuVisible((v) => !v)}>
              Export
            </Button>
          </Dropdown>

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
          <Button
            onClick={handleExamineEnd}
            ref={doneButtonRef}
            hidden={!props.exitable}
          >
            Done
          </Button>
        </div>
      </Affix>
    );
  }
);

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
    <Menu.Item key="download-png-turn">PNG</Menu.Item>
    <Menu.Item key="download-animated-gif-turn">
      Animated GIF to this position
    </Menu.Item>
  </Menu>
);

export type Props = {
  isExamining: boolean;
  exchangeAllowed?: boolean;
  finalPassOrChallenge?: boolean;
  myTurn?: boolean;
  observer?: boolean;
  allowAnalysis: boolean;
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
  puzzleMode?: boolean;
  setHandlePassShortcut: ((handler: (() => void) | null) => void) | null;
  setHandleChallengeShortcut: ((handler: (() => void) | null) => void) | null;
  setHandleNeitherShortcut: ((handler: (() => void) | null) => void) | null;
  tournamentPairedMode?: boolean;
  showNudge: boolean;
  showAbort: boolean;
  exitableExaminer?: boolean;
  boardEditingMode?: boolean;
};

const GameControls = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const { gameContext } = useGameContextStoreContext();
  const gameHasNotStarted = gameContext.players.length === 0; // :shrug:

  // Poka-yoke against accidentally having multiple pop-ups active.
  const [actualCurrentPopUp, setCurrentPopUp] = useState<
    'NONE' | 'CHALLENGE' | 'PASS'
  >('NONE');
  const [optionsMenuVisible, setOptionsMenuVisible] = useState(false);
  // This should match disabled= and/or hidden= props.
  const currentPopUp =
    (actualCurrentPopUp === 'CHALLENGE' &&
      (!props.myTurn || props.challengeRule === ChallengeRule.VOID)) ||
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

  const navigate = useNavigate();
  const handleExitToLobby = useCallback(() => {
    props.tournamentSlug ? navigate(props.tournamentSlug) : navigate('/');
  }, [navigate, props.tournamentSlug]);

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

  // this gameDone is slightly different from the one in table.tsx,
  // but it's good enough, otherwise we need to prop-drill further.
  const gameDone = !!gameEndControls;

  if (isExamining && !props.boardEditingMode) {
    return (
      <ExamineGameControls
        lexicon={props.lexicon}
        darkMode={darkMode}
        onExportGCG={props.onExportGCG}
        gameDone={gameDone}
        puzzleMode={!!props.puzzleMode}
        exitable={props.exitableExaminer ?? true}
        editMode={false}
      />
    );
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
        darkMode={darkMode}
        puzzleMode={!!props.puzzleMode}
      />
    );
  }

  if (observer) {
    return (
      <div className="game-controls">
        <Button
          onClick={props.onExamine}
          disabled={gameHasNotStarted || !props.allowAnalysis}
        >
          Examine
        </Button>
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
              title: (
                <p className="readable-text-color">
                  Are you sure you wish to resign?
                </p>
              ),
              icon: <ExclamationCircleOutlined />,
              // XXX: what if it's unrated?
              content: (
                <p className="readable-text-color">
                  Your rating may be affected.
                </p>
              ),
              onOk() {
                props.onResign();
              },
            });
            break;
          case 'abort':
            props.onRequestAbort();
            break;
          case 'nudge':
            props.onNudge();
            break;
          case 'download-png-turn':
            downloadGameImg(
              `${gameContext.gameID}${gameDone ? '-v2' : ''}-${
                gameContext.turns.length + 1
              }.png`
            );
            break;
          case 'download-animated-gif-turn':
            downloadGameImg(
              `${gameContext.gameID}${gameDone ? '-v2-b' : '-a'}-${
                gameContext.turns.length + 1
              }.gif`
            );
            break;
        }
      }}
      darkMode={darkMode}
    />
  );

  return (
    <div className={props.boardEditingMode ? 'board-editor-controls' : ''}>
      <div className="game-controls">
        <div className="secondary-controls">
          {!props.puzzleMode && !props.boardEditingMode && (
            <Dropdown
              overlay={optionsMenu}
              trigger={['click']}
              visible={optionsMenuVisible}
              disabled={gameHasNotStarted}
              placement="topLeft"
            >
              <Button onClick={() => setOptionsMenuVisible((v) => !v)}>
                Options
              </Button>
            </Dropdown>
          )}
          {!props.puzzleMode && (
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
                  props.finalPassOrChallenge && props.myTurn
                    ? 'primary'
                    : 'default'
                }
              >
                Pass
                <span className="key-command">2</span>
              </Button>
            </Popconfirm>
          )}
        </div>
        <div className="secondary-controls">
          {!props.puzzleMode && (
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
                hidden={props.challengeRule === ChallengeRule.VOID}
              >
                Challenge
                <span className="key-command">3</span>
              </Button>
            </Popconfirm>
          )}
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
          {props.puzzleMode ? 'Solve' : 'Play'}
        </Button>
      </div>
      {props.boardEditingMode && (
        <div className="secondary-controls">
          <ExamineGameControls
            lexicon={props.lexicon}
            darkMode={darkMode}
            onExportGCG={props.onExportGCG}
            gameDone={gameDone}
            puzzleMode={!!props.puzzleMode}
            exitable={props.exitableExaminer ?? true}
            editMode={!!props.boardEditingMode}
          />
        </div>
      )}
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
  darkMode: boolean;
  puzzleMode: boolean;
};

const EndGameControls = (props: EGCProps) => {
  const { useState } = useMountedState();
  const [rematchDisabled, setRematchDisabled] = useState(false);
  const { gameContext } = useGameContextStoreContext();
  const gameHasNotStarted = gameContext.players.length === 0; // :shrug:
  const gameDone = true; // it is endgame controls after all

  const [exportMenuVisible, setExportMenuVisible] = useState(false);
  const [exportMenuId, setExportMenuId] = useState(0);
  useEffect(() => {
    if (!exportMenuVisible) {
      // when the menu is hidden, yeet it and replace with a new instance altogether.
      // this works around old items being selected when reopening the menu.
      setExportMenuId((n) => (n + 1) | 0);
    }
  }, [exportMenuVisible]);
  const exportMenu = (
    <Menu
      key={exportMenuId}
      onClick={(e) => {
        setExportMenuVisible(false);
        // When at the last move, examineStoreContext.examinedTurn === Infinity.
        // To also detect new moves, we use examinableGameContext.turns.length.
        switch (e.key) {
          case 'download-png':
            downloadGameImg(
              `${gameContext.gameID}${gameDone ? '-v2' : ''}.png`
            );
            break;
          case 'download-animated-gif':
            downloadGameImg(
              `${gameContext.gameID}${gameDone ? '-v2-b' : '-a'}.gif`
            );
            break;
        }
      }}
      onMouseLeave={(e) => {
        setExportMenuVisible(false);
      }}
      theme={props.darkMode ? 'dark' : 'light'}
    >
      <Menu.Item key="download-png" disabled={gameHasNotStarted}>
        PNG
      </Menu.Item>
      <Menu.Item key="download-animated-gif" disabled={gameHasNotStarted}>
        Animated GIF of complete game
      </Menu.Item>
      <Menu.Item
        key="download-gcg"
        disabled={gameHasNotStarted}
        onClick={props.onExportGCG}
      >
        GCG
      </Menu.Item>
    </Menu>
  );

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        {!props.puzzleMode && (
          <Dropdown
            overlay={exportMenu}
            trigger={['click']}
            visible={exportMenuVisible}
            placement="topLeft"
          >
            <Button onClick={() => setExportMenuVisible((v) => !v)}>
              Export
            </Button>
          </Dropdown>
        )}
        <Button onClick={props.onExamine} disabled={gameHasNotStarted}>
          Examine
        </Button>
      </div>
      <div className="secondary-controls">
        {!props.puzzleMode && <Button onClick={props.onExit}>Exit</Button>}
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
