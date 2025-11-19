/**
 * Preferences V2 - Mantine-based Preferences Panel
 *
 * Pixel-perfect recreation of the preferences panel
 */

import React, { useState, useCallback, useEffect } from "react";
import { Box, Switch, Select, Title, Text, Group, Grid, Stack } from "@mantine/core";
import { DndProvider } from "react-dnd";
import { TouchBackend } from "react-dnd-touch-backend";
import { useUIStore } from "../stores/ui-store";
import {
  preferredSortOrder,
  setPreferredSortOrder,
  setSharedEnableAutoShuffle,
  sharedEnableAutoShuffle,
} from "../store/constants";
import {
  getTurnNotificationPreference,
  setTurnNotificationPreference,
  requestNotificationPermission,
  getNotificationPermissionState,
} from "../utils/notifications";
import { BoardPreview } from "../settings/board_preview";
import { puzzleLexica } from "../shared/lexicon_display";

// Tile order options
const KNOWN_TILE_ORDERS = [
  { name: "Alphabetical", value: "" },
  { name: "Vowels first", value: "AÄEIOÖUÜÆØÅ" },
  { name: "Consonants first", value: "BCÇDFGHJKLMNPQRSTVWXYZ" },
  { name: "Descending points", value: "QZJXKFHVWYBCMPDG" },
  { name: "Blanks first", value: "?" },
];

// Board style options
const KNOWN_BOARD_STYLES = [
  { name: "Default", value: "" },
  { name: "Cheery", value: "cheery" },
  { name: "Almost Colorless", value: "charcoal" },
  { name: "Forest", value: "forest" },
  { name: "Aflame", value: "aflame" },
  { name: "Teal and Plum", value: "tealish" },
  { name: "Pastel", value: "pastel" },
  { name: "Vintage", value: "vintage" },
  { name: "Balsa", value: "balsa" },
];

// Tile style options
const KNOWN_TILE_STYLES = [
  { name: "Default", value: "" },
  { name: "Purple", value: "purple" },
  { name: "Gray", value: "gray" },
  { name: "Blue", value: "blue" },
  { name: "Green", value: "green" },
  { name: "Light Gray", value: "lightgray" },
  { name: "Pink", value: "pink" },
  { name: "Slate", value: "slate" },
  { name: "Teal", value: "teal" },
];

// Section header component to avoid repetition
const SectionHeader: React.FC<{ children: React.ReactNode; themeMode: 'light' | 'dark' }> = ({ children, themeMode }) => (
  <Text
    mt={24}
    mb={12}
    fz={12}
    fw="bold"
    tt="uppercase"
    c={themeMode === "dark" ? "gray.0" : "gray.7"}
    style={{ letterSpacing: "0.16em" }}
  >
    {children}
  </Text>
);

export const PreferencesV2: React.FC = () => {
  // Theme state from Zustand
  const themeMode = useUIStore((state) => state.themeMode);
  const setThemeMode = useUIStore((state) => state.setThemeMode);
  const boardMode = useUIStore((state) => state.boardMode);
  const setBoardMode = useUIStore((state) => state.setBoardMode);
  const tileMode = useUIStore((state) => state.tileMode);
  const setTileMode = useUIStore((state) => state.setTileMode);

  // Local state
  const [turnNotifications, setTurnNotifications] = useState(
    getTurnNotificationPreference(),
  );
  const [permissionState, setPermissionState] = useState(
    getNotificationPermissionState(),
  );
  const [puzzleLexicon, setPuzzleLexicon] = useState(
    localStorage?.getItem("puzzleLexicon") || "NWL23",
  );
  const [tileOrder, setTileOrder] = useState(preferredSortOrder ?? "");
  const [reevaluateTileOrderOptions, setReevaluateTileOrderOptions] =
    useState(0);

  // Auto-request notification permission on mount
  useEffect(() => {
    const currentPermission = getNotificationPermissionState();
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
      setPermissionState("granted");
    } else {
      setPermissionState(currentPermission);
    }
  }, []);

  // Handlers
  const handleTurnNotificationsChange = useCallback(
    async (checked: boolean) => {
      if (checked) {
        const currentPermission = getNotificationPermissionState();

        if (currentPermission === "denied") {
          alert(
            "Notifications are blocked. Please enable them in your browser settings.",
          );
          return;
        }

        if (currentPermission === "default") {
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

  const handlePuzzleLexiconChange = useCallback((lexicon: string | null) => {
    if (!lexicon) return;
    localStorage.setItem("puzzleLexicon", lexicon);
    setPuzzleLexicon(lexicon);
  }, []);

  const handleTileOrderAndAutoShuffleChange = useCallback(
    (value: string | null) => {
      if (value === null) return;
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
    },
    [],
  );

  const localEnableAutoShuffle = sharedEnableAutoShuffle;

  const makeTileOrderValue = (tileOrder: string, autoShuffle: boolean) =>
    JSON.stringify({ tileOrder, autoShuffle });

  const tileOrderValue = React.useMemo(
    () => makeTileOrderValue(tileOrder, localEnableAutoShuffle),
    [tileOrder, localEnableAutoShuffle],
  );

  const tileOrderOptions = React.useMemo(() => {
    void reevaluateTileOrderOptions;
    const ret: Array<{ label: string; value: string }> = [];
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
      let hoverHelp = "";
      if (autoShuffle !== localEnableAutoShuffle) {
        hoverHelp = ` (turn ${autoShuffle ? "on" : "off"} auto-shuffle)`;
      }
      if (name === "Alphabetical" && autoShuffle) {
        nameText = "Random";
        hoverHelp = " (automatically shuffle tiles at every turn)";
      } else if (autoShuffle) {
        nameText = `${nameText} (auto-shuffle)`;
      }
      ret.push({
        label: nameText + hoverHelp,
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
    <Box className="preferences">
      {/* h3 title */}
      <Title order={3} fw={700} mb={24} style={{ letterSpacing: 0 }}>
        Preferences
      </Title>

      {/* Display section */}
      <SectionHeader themeMode={themeMode}>Display</SectionHeader>

      <Stack gap={24}>
        {/* Dark mode toggle */}
        <Group gap={12} align="flex-start">
          <Box style={{ flex: 1 }}>
            <Text fw="bold" mb={8} fz={16}>
              Dark mode
            </Text>
            <Text>Use the dark version of the Woogles UI on Woogles.io</Text>
          </Box>
          <Switch
            checked={themeMode === "dark"}
            onChange={(event) =>
              setThemeMode(event.currentTarget.checked ? "dark" : "light")
            }
          />
        </Group>

        {/* Turn notifications */}
        <Group gap={12} align="flex-start">
          <Box style={{ flex: 1 }}>
            <Text fw="bold" mb={8} fz={16}>
              Turn notifications
            </Text>
            <Text>
              Get a notification when it's your turn (works when tab is
              unfocused)
            </Text>
            {permissionState === "denied" && (
              <Text mt={8} fz={12} c="red.6">
                Permission blocked. Please enable in browser settings.
              </Text>
            )}
          </Box>
          <Switch
            checked={turnNotifications}
            onChange={(event) =>
              handleTurnNotificationsChange(event.currentTarget.checked)
            }
            disabled={
              permissionState === "denied" ||
              permissionState === "unsupported"
            }
          />
        </Group>
      </Stack>

      {/* OMGWords settings section */}
      <SectionHeader themeMode={themeMode}>OMGWords settings</SectionHeader>

      {/* Grid layout - 50% width like original Col span={12} */}
      <Grid>
        <Grid.Col span={6}>
          {/* Default tile order */}
          <Box mb={18}>
            <Text fw="bold" mb={6} fz={16}>
              Default tile order
            </Text>
            <Select
              data={tileOrderOptions}
              value={tileOrderValue}
              onChange={handleTileOrderAndAutoShuffleChange}
            />
          </Box>

          {/* Tile style */}
          <Box mb={18}>
            <Text fw="bold" mb={6} fz={16}>
              Tile style
            </Text>
            <Select
              data={KNOWN_TILE_STYLES.map((s) => ({
                value: s.value,
                label: s.name,
              }))}
              value={tileMode}
              onChange={(value) => value !== null && setTileMode(value)}
            />
          </Box>

          {/* Board style */}
          <Box mb={18}>
            <Text fw="bold" mb={6} fz={16}>
              Board style
            </Text>
            <Select
              data={KNOWN_BOARD_STYLES.map((s) => ({
                value: s.value,
                label: s.name,
              }))}
              value={boardMode}
              onChange={(value) => value !== null && setBoardMode(value)}
            />
          </Box>

          {/* Board Preview */}
          <Box mb={18}>
            <DndProvider
              backend={TouchBackend}
              options={{ enableMouseEvents: true }}
            >
              <BoardPreview />
            </DndProvider>
          </Box>
        </Grid.Col>
      </Grid>

      {/* OMGWords Puzzle Mode settings section */}
      <SectionHeader themeMode={themeMode}>OMGWords Puzzle Mode Settings</SectionHeader>

      {/* Puzzle lexicon */}
      <Grid>
        <Grid.Col span={6}>
          <Box mb={18}>
            <Select
              data={puzzleLexica}
              value={puzzleLexicon}
              onChange={handlePuzzleLexiconChange}
            />
          </Box>
        </Grid.Col>
      </Grid>
    </Box>
  );
};
