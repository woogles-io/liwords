// Static import — Three.js is bundled into this chunk.
// game_controls.tsx loads this file lazily via React.lazy, so Three.js
// only downloads when the user first opens the 3D view.
import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Button, Select } from "antd";
import { Board3DScene } from "./scene";
import { Board3DData } from "./types";

const tileColorOptions = [
  { value: "orange", label: "Orange" },
  { value: "yellow", label: "Yellow" },
  { value: "pink", label: "Pink" },
  { value: "red", label: "Red" },
  { value: "blue", label: "Blue" },
  { value: "black", label: "Black" },
  { value: "white", label: "White" },
];

const boardColorOptions = [
  { value: "jade", label: "Jade" },
  { value: "teal", label: "Teal" },
  { value: "blue", label: "Blue" },
  { value: "purple", label: "Purple" },
  { value: "green", label: "Green" },
  { value: "yellow", label: "Yellow" },
  { value: "red", label: "Red" },
  { value: "slate", label: "Slate" },
];

type Props = {
  open: boolean;
  onClose: () => void;
  data: Board3DData | null;
  filename?: string;
};

export const Board3DModal = React.memo((props: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [tileColor, setTileColor] = useState("orange");
  const [boardColor, setBoardColor] = useState("jade");
  const [isSpinning, setIsSpinning] = useState(false);
  const sceneRef = useRef<Board3DScene | null>(null);
  const isOpenRef = useRef(false);

  const destroyScene = useCallback(() => {
    if (sceneRef.current) {
      sceneRef.current.dispose();
      sceneRef.current = null;
    }
    setIsSpinning(false);
  }, []);

  const createScene = useCallback(() => {
    const container = containerRef.current;
    if (!container || !props.data) return;
    destroyScene();
    try {
      const data: Board3DData = { ...props.data, tileColor, boardColor };
      sceneRef.current = new Board3DScene(container, data);
    } catch (e) {
      console.error("Failed to create 3D scene:", e);
    }
  }, [props.data, tileColor, boardColor, destroyScene]);

  // afterOpenChange fires after the open animation completes, guaranteeing
  // that the container div has its final layout dimensions.
  const handleAfterOpenChange = useCallback(
    (visible: boolean) => {
      isOpenRef.current = visible;
      if (visible) {
        createScene();
      } else {
        destroyScene();
      }
    },
    [createScene, destroyScene],
  );

  // Recreate when colors change while the modal is open
  useEffect(() => {
    if (isOpenRef.current) {
      createScene();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tileColor, boardColor]);

  const handleSavePNG = useCallback(() => {
    sceneRef.current?.saveAsPNG(props.filename);
  }, [props.filename]);

  const handleToggleSpin = useCallback(() => {
    const next = sceneRef.current?.toggleSpin();
    if (next !== undefined) setIsSpinning(next);
  }, []);

  return (
    <Modal
      title="3D Board View"
      open={props.open}
      onCancel={props.onClose}
      width="90vw"
      destroyOnClose
      afterOpenChange={handleAfterOpenChange}
      footer={<Button onClick={props.onClose}>Close</Button>}
    >
      {/* Controls toolbar — lives in the body so it scrolls with content on mobile
          and doesn't fight the footer's fixed-position mobile CSS. */}
      <div
        style={{
          display: "flex",
          gap: 8,
          alignItems: "center",
          flexWrap: "wrap",
          padding: "8px 0 8px",
        }}
      >
        <span>Tile:</span>
        <Select
          value={tileColor}
          onChange={setTileColor}
          options={tileColorOptions}
          style={{ width: 100 }}
        />
        <span>Board:</span>
        <Select
          value={boardColor}
          onChange={setBoardColor}
          options={boardColorOptions}
          style={{ width: 100 }}
        />
        <Button onClick={handleToggleSpin}>
          {isSpinning ? "Stop spin" : "Spin board"}
        </Button>
        <Button onClick={handleSavePNG}>Save as PNG</Button>
      </div>
      <div
        ref={containerRef}
        style={{ width: "100%", height: "min(70vh, calc(100dvh - 220px))" }}
      />
    </Modal>
  );
});

export default Board3DModal;
