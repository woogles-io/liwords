import React, { useCallback, useState } from "react";
import { Col, Row, Select, Switch } from "antd";
import {
  preferredSortOrder,
  setPreferredSortOrder,
  setSharedEnableAutoShuffle,
  sharedEnableAutoShuffle,
} from "../store/constants";
import "../gameroom/scss/gameroom.scss";
import { BoardPreview } from "./board_preview";
import { MatchLexiconDisplay, puzzleLexica } from "../shared/lexicon_display";
import { TouchBackend } from "react-dnd-touch-backend";
import { DndProvider } from "react-dnd";
import { useUIStore } from "../stores/ui-store";
import {
  getTurnNotificationPreference,
  setTurnNotificationPreference,
  requestNotificationPermission,
  getNotificationPermissionState,
} from "../utils/notifications";

const previewTilesLayout = [
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
  "  WOOGLES OmG  ",
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
  "               ",
];

const KNOWN_TILE_ORDERS = [
  {
    name: "Alphabetical",
    value: "",
  },
  {
    name: "Vowels first",
    value: "AÄEIOÖUÜÆØÅ",
  },
  {
    name: "Consonants first",
    value: "BCÇDFGHJKLMNPQRSTVWXYZ",
  },
  {
    name: "Descending points",
    value: "QZJXKFHVWYBCMPDG",
  },
  {
    name: "Blanks first",
    value: "?",
  },
];

const KNOWN_BOARD_STYLES = [
  {
    name: "Default",
    value: "",
  },
  {
    name: "Cheery",
    value: "cheery",
  },
  {
    name: "Almost Colorless",
    value: "charcoal",
  },
  {
    name: "Forest",
    value: "forest",
  },
  {
    name: "Aflame",
    value: "aflame",
  },
  {
    name: "Teal and Plum",
    value: "tealish",
  },
  {
    name: "Pastel",
    value: "pastel",
  },
  {
    name: "Vintage",
    value: "vintage",
  },
  {
    name: "Balsa",
    value: "balsa",
  },
  {
    name: "Mahogany",
    value: "mahogany",
  },
  {
    name: "Metallic",
    value: "metallic",
  },
];

const KNOWN_TILE_STYLES = [
  {
    name: "Default",
    value: "",
  },
  {
    name: "Charcoal",
    value: "charcoal",
  },
  {
    name: "White",
    value: "whitish",
  },
  {
    name: "Mahogany",
    value: "mahogany",
  },
  {
    name: "Balsa",
    value: "balsa",
  },
  {
    name: "Brick",
    value: "brick",
  },
  {
    name: "Forest",
    value: "forest",
  },
  {
    name: "Teal",
    value: "tealish",
  },
  {
    name: "Plum",
    value: "plumish",
  },
  {
    name: "Pastel",
    value: "pastel",
  },
  {
    name: "Fuchsia",
    value: "fuchsiaish",
  },
  {
    name: "Blue",
    value: "blueish",
  },
  {
    name: "Metallic",
    value: "metallic",
  },
];

const makeTileOrderValue = (tileOrder: string, autoShuffle: boolean) =>
  JSON.stringify({ tileOrder, autoShuffle });

export const Preferences = React.memo(() => {
  const themeMode = useUIStore((state) => state.themeMode);
  const setThemeMode = useUIStore((state) => state.setThemeMode);

  const initialTileStyle = localStorage?.getItem("userTile") || "Default";
  const [userTile, setUserTile] = useState<string>(initialTileStyle);

  const initialBoardStyle = localStorage?.getItem("userBoard") || "Default";
  const [userBoard, setUserBoard] = useState<string>(initialBoardStyle);

  const initialPuzzleLexicon =
    localStorage?.getItem("puzzleLexicon") || undefined;
  const [puzzleLexicon, setPuzzleLexicon] = useState<string | undefined>(
    initialPuzzleLexicon,
  );

  const [turnNotifications, setTurnNotifications] = useState(
    getTurnNotificationPreference(),
  );
  const [permissionState, setPermissionState] = useState(
    getNotificationPermissionState(),
  );

  const handleUserTileChange = useCallback((tileStyle: string) => {
    const classes = document?.body?.className
      .split(" ")
      .filter((c) => !c.startsWith("tile--"));
    document.body.className = classes.join(" ").trim();
    if (tileStyle) {
      localStorage.setItem("userTile", tileStyle);
      document?.body?.classList?.add(`tile--${tileStyle}`);
    } else {
      localStorage.removeItem("userTile");
    }
    setUserTile(tileStyle);
  }, []);

  const handlePuzzleLexiconChange = useCallback((lexicon: string) => {
    localStorage.setItem("puzzleLexicon", lexicon);
    setPuzzleLexicon(lexicon);
  }, []);

  // Auto-request permission on mount if not already decided
  React.useEffect(() => {
    const currentPermission = getNotificationPermissionState();

    // If permission is default (not yet decided), auto-request
    if (currentPermission === "default") {
      requestNotificationPermission().then((granted) => {
        if (granted) {
          setTurnNotificationPreference(true);
          setTurnNotifications(true);
          setPermissionState("granted");
        } else {
          setPermissionState("denied");
        }
      });
    } else if (currentPermission === "granted") {
      // Check if user previously denied but then re-enabled in browser settings
      // If so, and we don't have the preference set, auto-enable
      const currentPref = getTurnNotificationPreference();
      if (!currentPref) {
        // Permission is granted but preference is off - could be newly granted
        // Don't auto-enable in this case, let user control it
      }
      setPermissionState("granted");
    } else {
      setPermissionState(currentPermission);
    }
  }, []);

  const handleTurnNotificationsChange = useCallback(
    async (checked: boolean) => {
      if (checked) {
        const currentPermission = getNotificationPermissionState();

        if (currentPermission === "denied") {
          // Show alert about blocked permission
          alert(
            "Notifications are blocked. Please enable them in your browser settings.",
          );
          return;
        }

        if (currentPermission === "default") {
          // Request permission first
          const granted = await requestNotificationPermission();
          if (granted) {
            setTurnNotificationPreference(true);
            setTurnNotifications(true);
            setPermissionState("granted");
          } else {
            setTurnNotificationPreference(false);
            setTurnNotifications(false);
            setPermissionState("denied");
          }
        } else if (currentPermission === "granted") {
          // Permission already granted, just update preference
          setTurnNotificationPreference(true);
          setTurnNotifications(true);
        }
      } else {
        setTurnNotificationPreference(false);
        setTurnNotifications(false);
      }
    },
    [],
  );

  const handleUserBoardChange = useCallback((boardStyle: string) => {
    const classes = document?.body?.className
      .split(" ")
      .filter((c) => !c.startsWith("board--"));
    document.body.className = classes.join(" ").trim();
    if (boardStyle) {
      localStorage.setItem("userBoard", boardStyle);
      document?.body?.classList?.add(`board--${boardStyle}`);
    } else {
      localStorage.removeItem("userBoard");
    }
    setUserBoard(boardStyle);
  }, []);

  const [reevaluateTileOrderOptions, setReevaluateTileOrderOptions] =
    useState(0);
  const [tileOrder, setTileOrder] = useState(preferredSortOrder ?? "");
  const handleTileOrderAndAutoShuffleChange = useCallback((value: string) => {
    try {
      const parsedStuff = JSON.parse(value);
      const { tileOrder: newTileOrder, autoShuffle: newAutoShuffle } =
        parsedStuff;
      setTileOrder(newTileOrder);
      setPreferredSortOrder(newTileOrder);
      setSharedEnableAutoShuffle(newAutoShuffle);
      setReevaluateTileOrderOptions((x) => (x + 1) | 0);
    } catch (e) {
      console.error(e);
    }
  }, []);
  const localEnableAutoShuffle = sharedEnableAutoShuffle;
  const tileOrderValue = React.useMemo(
    () => makeTileOrderValue(tileOrder, localEnableAutoShuffle),
    [tileOrder, localEnableAutoShuffle],
  );
  const tileOrderOptions = React.useMemo(() => {
    void reevaluateTileOrderOptions;
    const ret: Array<{ name: React.ReactElement<Element>; value: string }> = [];
    const pushTileOrder = (
      name: string,
      value: string,
      autoShuffle: boolean,
    ) => {
      // Design only wants "Random" for "Alphabetical".
      if (
        !(
          (tileOrder === value && autoShuffle === localEnableAutoShuffle) ||
          name === "Alphabetical" ||
          !autoShuffle
        )
      ) {
        return;
      }
      let nameText = name;
      let hoverHelp = null;
      if (autoShuffle !== localEnableAutoShuffle) {
        hoverHelp = `(turn ${autoShuffle ? "on" : "off"} auto-shuffle)`;
      }
      if (name === "Alphabetical" && autoShuffle) {
        nameText = "Random";
        hoverHelp = "(automatically shuffle tiles at every turn)";
      } else if (autoShuffle) {
        // The page should still work logically if this is set manually.
        // localStorage.enableAutoShuffle = true;
        nameText = `${nameText} (auto-shuffle)`;
      }
      ret.push({
        name: (
          <React.Fragment>
            {nameText}
            {hoverHelp && (
              <React.Fragment>
                {" "}
                <span className="hover-help">{hoverHelp}</span>
              </React.Fragment>
            )}
          </React.Fragment>
        ),
        value: makeTileOrderValue(value, autoShuffle),
      });
    };
    const addTileOrder = ({ name, value }: { name: string; value: string }) => {
      pushTileOrder(name, value, false);
      pushTileOrder(name, value, true);
    };
    let found = false;
    for (const { name, value } of KNOWN_TILE_ORDERS) {
      if (value === tileOrder) found = true;
      addTileOrder({ name, value });
    }
    if (!found) {
      addTileOrder({ name: "Custom", value: tileOrder });
    }
    return ret;
  }, [reevaluateTileOrderOptions, tileOrder, localEnableAutoShuffle]);

  return (
    <div className="preferences">
      <h3>Preferences</h3>
      <div className="section-header">Display</div>
      <div className="toggles-section">
        <div>
          <div className="toggle-section">
            <div className="title">Dark mode</div>
            <div>
              <div>Use the dark version of the Woogles UI on Woogles.io</div>
              <Switch
                checked={themeMode === 'dark'}
                onChange={(checked: boolean) => setThemeMode(checked ? 'dark' : 'light')}
                className="dark-toggle"
              />
            </div>
          </div>
          <div className="toggle-section">
            <div className="title">Turn notifications</div>
            <div>
              <div>
                Get a notification when it's your turn (works when tab is
                unfocused)
              </div>
              <Switch
                checked={turnNotifications}
                onChange={handleTurnNotificationsChange}
                disabled={
                  permissionState === "denied" ||
                  permissionState === "unsupported"
                }
                className="notification-toggle"
              />
              {permissionState === "denied" && (
                <div style={{ marginTop: 8, fontSize: 12, color: "#ff4d4f" }}>
                  Permission blocked.{" "}
                  <a
                    href="#"
                    onClick={(e) => {
                      e.preventDefault();
                      alert(
                        "To enable notifications:\n\n" +
                          "Chrome/Edge: Click the lock icon in the address bar → Site settings → Notifications → Allow\n\n" +
                          "Firefox: Click the lock icon → Clear permissions → Refresh page\n\n" +
                          "Safari: Safari menu → Settings → Websites → Notifications → woogles.io → Allow",
                      );
                    }}
                  >
                    How to enable in browser settings
                  </a>
                </div>
              )}
              {permissionState === "unsupported" && (
                <div style={{ marginTop: 8, fontSize: 12, color: "#8c8c8c" }}>
                  Your browser doesn't support notifications
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
      <div className="section-header">OMGWords settings</div>
      <Row>
        <Col span={12}>
          <div className="tile-order">Default tile order</div>
          <Select
            className="tile-order-select"
            size="large"
            defaultValue={tileOrderValue}
            onChange={handleTileOrderAndAutoShuffleChange}
          >
            {tileOrderOptions.map(({ name, value }) => (
              <Select.Option value={value} key={value}>
                {name}
              </Select.Option>
            ))}
          </Select>
          <div className="tile-order">Tile style</div>
          <div className="tile-selection">
            <Select
              className="tile-style-select"
              size="large"
              defaultValue={userTile}
              onChange={handleUserTileChange}
            >
              {KNOWN_TILE_STYLES.map(({ name, value }) => (
                <Select.Option value={value} key={value}>
                  {name}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="board-style">Board style</div>
          <div className="board-selection">
            <Select
              className="board-style-select"
              size="large"
              defaultValue={userBoard}
              onChange={handleUserBoardChange}
            >
              {KNOWN_BOARD_STYLES.map(({ name, value }) => (
                <Select.Option value={value} key={value}>
                  {name}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="previewer">
            <DndProvider backend={TouchBackend}>
              <BoardPreview
                tilesLayout={previewTilesLayout}
                lastPlayedTiles={{
                  R7C10: true,
                  R7C11: true,
                  R7C12: true,
                }}
              />
            </DndProvider>
          </div>
        </Col>
      </Row>
      <div className="section-header">OMGWords Puzzle Mode settings</div>
      <Row>
        <Col span={12}>
          <Select
            className="puzzle-lexicon-selection"
            size="large"
            onChange={handlePuzzleLexiconChange}
            defaultValue={puzzleLexicon}
          >
            {puzzleLexica.map((k) => (
              <Select.Option key={k} value={k}>
                <MatchLexiconDisplay lexiconCode={k} useShortDescription />
              </Select.Option>
            ))}
          </Select>
        </Col>
      </Row>
    </div>
  );
});
