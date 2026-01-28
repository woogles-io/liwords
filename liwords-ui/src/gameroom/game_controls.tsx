import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router";
import { Affix, App, Button, Dropdown, MenuProps, Popconfirm } from "antd";

import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  ExclamationCircleOutlined,
  LeftOutlined,
  RightOutlined,
} from "@ant-design/icons";
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useTentativeTileContext,
} from "../store/store";
import { EphemeralTile } from "../utils/cwgame/common";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";

const downloadGameImg = (downloadFilename: string) => {
  const link = document.createElement("a");
  link.href = new URL(
    `/gameimg/${encodeURIComponent(downloadFilename)}`,
    window.location.href,
  ).href;
  link.setAttribute("download", downloadFilename);
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

    const exportMenuItems = useMemo(() => {
      const items = [
        {
          label: "PNG",
          key: isAtLastTurn ? "download-png" : "download-png-turn",
          disabled: gameHasNotStarted,
        },
      ];

      if (!isAtLastTurn) {
        items.push({
          label: "Animated GIF to this position",
          key: "download-animated-gif-turn",
          disabled: gameHasNotStarted,
        });
      }
      if (gameDone) {
        items.push({
          label: "Animated GIF of complete game",
          key: "download-animated-gif",
          disabled: gameHasNotStarted,
        });
      }
      if (gameDone || props.editMode) {
        items.push({
          label: "GCG",
          key: "download-gcg",
          disabled: gameHasNotStarted,
          // add onclick to menu parent.
        });
      }
      return items;
    }, [gameDone, gameHasNotStarted, isAtLastTurn, props.editMode]);

    const exportMenuOnClick: MenuProps["onClick"] = ({ key }) => {
      // When at the last move, examineStoreContext.examinedTurn === Infinity.
      // To also detect new moves, we use examinableGameContext.turns.length.
      switch (key) {
        case "download-png":
          downloadGameImg(`${gameContext.gameID}${gameDone ? "-v2" : ""}.png`);
          break;
        case "download-png-turn":
          downloadGameImg(
            `${gameContext.gameID}${gameDone ? "-v2" : ""}-${
              examinableGameContext.turns.length + 1
            }.png`,
          );
          break;
        case "download-animated-gif-turn":
          downloadGameImg(
            `${gameContext.gameID}${gameDone ? "-v2-b" : "-a"}-${
              examinableGameContext.turns.length + 1
            }.gif`,
          );
          break;
        case "download-animated-gif":
          downloadGameImg(
            `${gameContext.gameID}${gameDone ? "-v2-b" : "-a"}.gif`,
          );
          break;
        case "download-gcg":
          props.onExportGCG();
          break;
      }
    };
    return (
      <Affix offsetTop={210} className="examiner-controls">
        <div className="game-controls">
          <Dropdown
            menu={{
              items: exportMenuItems,
              onClick: exportMenuOnClick,
              theme: props.darkMode ? "dark" : "light",
            }}
            trigger={["click"]}
            placement="topLeft"
            disabled={props.puzzleMode}
          >
            <Button>Export</Button>
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
  },
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
  tournamentSlug?: string;
  lexicon: string;
  challengeRule: ChallengeRule;
  puzzleMode?: boolean;
  setHandlePassShortcut:
    | ((
        makeNewValue:
          | ((oldValue: (() => void) | null) => (() => void) | null)
          | null,
      ) => void)
    | null;
  setHandleChallengeShortcut:
    | ((
        makeNewValue:
          | ((oldValue: (() => void) | null) => (() => void) | null)
          | null,
      ) => void)
    | null;
  setHandleNeitherShortcut:
    | ((
        makeNewValue:
          | ((oldValue: (() => void) | null) => (() => void) | null)
          | null,
      ) => void)
    | null;
  tournamentPairedMode?: boolean;
  isLeagueGame?: boolean;
  showNudge: boolean;
  showAbort: boolean;
  exitableExaminer?: boolean;
  boardEditingMode?: boolean;
};

const GameControls = React.memo((props: Props) => {
  const { gameContext } = useGameContextStoreContext();
  const gameHasNotStarted = gameContext.players.length === 0; // :shrug:

  // Poka-yoke against accidentally having multiple pop-ups active.
  const [actualCurrentPopUp, setCurrentPopUp] = useState<
    "NONE" | "CHALLENGE" | "PASS"
  >("NONE");
  // This should match disabled= and/or hidden= props.
  const currentPopUp =
    (actualCurrentPopUp === "CHALLENGE" &&
      (!props.myTurn || props.challengeRule === ChallengeRule.VOID)) ||
    (actualCurrentPopUp === "PASS" && !props.myTurn)
      ? "NONE"
      : actualCurrentPopUp;
  useEffect(() => {
    if (currentPopUp !== actualCurrentPopUp) {
      // Although this will still take another render, make this render correct
      // to avoid temporarily showing confirmation dialog.
      setCurrentPopUp(currentPopUp);
    }
  }, [currentPopUp, actualCurrentPopUp]);

  const passButton = useRef<HTMLButtonElement>(null);
  const challengeButton = useRef<HTMLButtonElement>(null);

  const darkMode = useMemo(
    () => localStorage?.getItem("darkMode") === "true",
    [],
  );

  const navigate = useNavigate();
  const handleExitToLobby = useCallback(() => {
    navigate(props.tournamentSlug || "/");
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
      if (v !== "PASS") return "PASS";
      if (onPass) onPass();
      return "NONE";
    });
  }, [hasRegularButtons, onPass]);
  const handleChallengeShortcut = useCallback(() => {
    if (!hasRegularButtons) return;
    if (!challengeButton.current) return;
    challengeButton.current.focus();
    setCurrentPopUp((v) => {
      if (v !== "CHALLENGE") return "CHALLENGE";
      if (onChallenge) onChallenge();
      return "NONE";
    });
  }, [hasRegularButtons, onChallenge]);
  const handleNeitherShortcut = useCallback(() => {
    if (!hasRegularButtons) return;
    setCurrentPopUp("NONE");
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

  const { modal } = App.useApp();

  const optionsMenuItems = useMemo(() => {
    const items = [];

    // Don't allow resign in league games
    if (!props.isLeagueGame) {
      items.push({
        label: "Resign",
        key: "resign",
      });
    }

    if (props.showAbort) {
      items.push({
        label: "Cancel Game",
        key: "abort",
      });
    }
    if (props.showNudge) {
      items.push({
        label: "Nudge",
        key: "nudge",
      });
    }
    items.push({
      label: "PNG",
      key: "download-png-turn",
    });
    items.push({
      label: "Animated GIF to this position",
      key: "download-animated-gif-turn",
    });
    return items;
  }, [props.showAbort, props.showNudge, props.isLeagueGame]);

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
          Analyze
        </Button>
        <Button onClick={handleExitToLobby}>Exit</Button>
      </div>
    );
  }

  const optionsMenuOnClick: MenuProps["onClick"] = ({ key }) => {
    switch (key) {
      case "resign":
        modal.confirm({
          title: (
            <p className="readable-text-color">
              Are you sure you wish to resign?
            </p>
          ),
          icon: <ExclamationCircleOutlined />,
          // XXX: what if it's unrated?
          content: (
            <p className="readable-text-color">
              If this is a rated game, your rating may be affected.
            </p>
          ),
          onOk() {
            props.onResign();
          },
        });
        break;
      case "abort":
        props.onRequestAbort();
        break;
      case "nudge":
        props.onNudge();
        break;
      case "download-png-turn":
        downloadGameImg(
          `${gameContext.gameID}${gameDone ? "-v2" : ""}-${
            gameContext.turns.length + 1
          }.png`,
        );
        break;
      case "download-animated-gif-turn":
        downloadGameImg(
          `${gameContext.gameID}${gameDone ? "-v2-b" : "-a"}-${
            gameContext.turns.length + 1
          }.gif`,
        );
        break;
    }
  };

  return (
    <div className={props.boardEditingMode ? "board-editor-controls" : ""}>
      <div className="game-controls">
        <div className="secondary-controls">
          {!props.puzzleMode && !props.boardEditingMode && (
            <Dropdown
              menu={{
                items: optionsMenuItems,
                onClick: optionsMenuOnClick,
                theme: darkMode ? "dark" : "light",
              }}
              trigger={["click"]}
              disabled={gameHasNotStarted}
              placement="topLeft"
            >
              <Button>Options</Button>
            </Dropdown>
          )}
          {!props.puzzleMode && (
            <Popconfirm
              title="Are you sure you wish to pass?"
              onCancel={() => {
                setCurrentPopUp("NONE");
              }}
              onConfirm={() => {
                props.onPass();
                setCurrentPopUp("NONE");
              }}
              onOpenChange={(visible) => {
                setCurrentPopUp(visible ? "PASS" : "NONE");
              }}
              okText="Yes"
              cancelText="No"
              open={currentPopUp === "PASS"}
            >
              <Button
                ref={passButton}
                onClick={() => {
                  if (currentPopUp === "PASS") {
                    props.onPass();
                    setCurrentPopUp("NONE");
                  }
                }}
                disabled={!props.myTurn}
                type={
                  props.finalPassOrChallenge && props.myTurn
                    ? "primary"
                    : "default"
                }
              >
                Pass
                <span className="key-command">2</span>
              </Button>
            </Popconfirm>
          )}
        </div>
        <div className="secondary-controls">
          {!props.puzzleMode &&
            (props.isExamining &&
            props.challengeRule === ChallengeRule.FIVE_POINT ? (
              // For editor mode + 5-point challenges, skip the confirmation popover
              // since the word selection modal already serves as confirmation
              <Button
                ref={challengeButton}
                onClick={props.onChallenge}
                disabled={!props.myTurn}
              >
                Challenge
                <span className="key-command">3</span>
              </Button>
            ) : (
              <Popconfirm
                title="Are you sure you wish to challenge?"
                onCancel={() => {
                  setCurrentPopUp("NONE");
                }}
                onConfirm={() => {
                  props.onChallenge();
                  setCurrentPopUp("NONE");
                }}
                onOpenChange={(visible) => {
                  setCurrentPopUp(visible ? "CHALLENGE" : "NONE");
                }}
                okText="Yes"
                cancelText="No"
                open={currentPopUp === "CHALLENGE"}
              >
                <Button
                  ref={challengeButton}
                  onClick={() => {
                    if (currentPopUp === "CHALLENGE") {
                      props.onChallenge();
                      setCurrentPopUp("NONE");
                    }
                  }}
                  disabled={!props.myTurn}
                  hidden={props.challengeRule === ChallengeRule.VOID}
                >
                  Challenge
                  <span className="key-command">3</span>
                </Button>
              </Popconfirm>
            ))}
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
          {props.puzzleMode ? "Solve" : "Play"}
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
  const [rematchDisabled, setRematchDisabled] = useState(false);
  const { gameContext } = useGameContextStoreContext();
  const gameHasNotStarted = gameContext.players.length === 0; // :shrug:
  const gameDone = true; // it is endgame controls after all

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        {!props.puzzleMode && (
          <Dropdown
            menu={{
              items: [
                {
                  key: "download-png",
                  label: "PNG",
                  disabled: gameHasNotStarted,
                },
                {
                  key: "download-animated-gif",
                  label: "Animated GIF of complete game",
                  disabled: gameHasNotStarted,
                },
                {
                  key: "download-gcg",
                  label: "GCG",
                  disabled: gameHasNotStarted,
                },
              ],
              onClick: ({ key }) => {
                switch (key) {
                  case "download-png":
                    downloadGameImg(
                      `${gameContext.gameID}${gameDone ? "-v2" : ""}.png`,
                    );
                    break;
                  case "download-animated-gif":
                    downloadGameImg(
                      `${gameContext.gameID}${gameDone ? "-v2-b" : "-a"}.gif`,
                    );
                    break;
                  case "download-gcg":
                    props.onExportGCG();
                    break;
                }
              },
            }}
            trigger={["click"]}
            placement="topLeft"
          >
            <Button>Export</Button>
          </Dropdown>
        )}
        <Button onClick={props.onExamine} disabled={gameHasNotStarted}>
          Analyze
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
