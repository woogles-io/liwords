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
  { value: "black", label: "Black" },
];

type Props = {
  open: boolean;
  onClose: () => void;
  data: Board3DData | null;
};

export const Board3DModal = React.memo((props: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [tileColor, setTileColor] = useState("orange");
  const [boardColor, setBoardColor] = useState("jade");
  const sceneRef = useRef<Board3DScene | null>(null);
  const isOpenRef = useRef(false);

  const destroyScene = useCallback(() => {
    if (sceneRef.current) {
      sceneRef.current.dispose();
      sceneRef.current = null;
    }
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
    sceneRef.current?.saveAsPNG();
  }, []);

  return (
    <Modal
      title="3D Board View"
      open={props.open}
      onCancel={props.onClose}
      width="90vw"
      destroyOnClose
      afterOpenChange={handleAfterOpenChange}
      footer={
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <span>Tile color:</span>
          <Select
            value={tileColor}
            onChange={setTileColor}
            options={tileColorOptions}
            style={{ width: 110 }}
          />
          <span>Board color:</span>
          <Select
            value={boardColor}
            onChange={setBoardColor}
            options={boardColorOptions}
            style={{ width: 120 }}
          />
          <Button onClick={handleSavePNG}>Save as PNG</Button>
          <Button onClick={props.onClose}>Close</Button>
        </div>
      }
    >
      <div
        ref={containerRef}
        style={{ width: "100%", height: "75vh" }}
      />
    </Modal>
  );
});

export default Board3DModal;
