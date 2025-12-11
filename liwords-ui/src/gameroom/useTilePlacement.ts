import { useCallback } from "react";
import {
  MachineLetter,
  EphemeralTile,
  EmptyBoardSpaceMachineLetter,
  EmptyRackSpaceMachineLetter,
} from "../utils/cwgame/common";
import { Board } from "../utils/cwgame/board";
import { stableInsertRack } from "../utils/cwgame/tile_placement";

type UseTilePlacementParams = {
  arrowProperties: {
    row: number;
    col: number;
    horizontal: boolean;
    show: boolean;
  };
  setArrowProperties: (props: {
    row: number;
    col: number;
    horizontal: boolean;
    show: boolean;
  }) => void;
  placedTiles: Set<EphemeralTile>;
  setPlacedTiles: (tiles: Set<EphemeralTile>) => void;
  setPlacedTilesTempScore: (score: number | undefined) => void;
  displayedRack: Array<MachineLetter>;
  setDisplayedRack: (rack: Array<MachineLetter>) => void;
  board: Board;
  currentRack: Array<MachineLetter>;
  setPendingExchangeTiles?: (tiles: Array<MachineLetter> | null) => void;
};

export function useTilePlacement(params: UseTilePlacementParams) {
  const {
    arrowProperties,
    setArrowProperties,
    placedTiles,
    setPlacedTiles,
    setPlacedTilesTempScore,
    displayedRack,
    setDisplayedRack,
    board,
    currentRack,
    setPendingExchangeTiles,
  } = params;

  // Recall all tiles to rack
  const recallTiles = useCallback(() => {
    if (arrowProperties.show) {
      let { row, col } = arrowProperties;
      const { horizontal } = arrowProperties;
      const matchesLocation = ({
        row: tentativeRow,
        col: tentativeCol,
      }: {
        row: number;
        col: number;
      }) => row === tentativeRow && col === tentativeCol;
      if (
        horizontal &&
        row >= 0 &&
        row < board.dim &&
        col > 0 &&
        col <= board.dim
      ) {
        const placedTilesArray = Array.from(placedTiles);
        let best = col;
        while (col > 0) {
          --col;
          if (
            board.letters[row * board.dim + col] !==
            EmptyBoardSpaceMachineLetter
          ) {
            // continue
          } else if (placedTilesArray.some(matchesLocation)) {
            best = col;
          } else {
            break;
          }
        }
        if (best !== arrowProperties.col) {
          setArrowProperties({ ...arrowProperties, col: best });
        }
      } else if (
        !horizontal &&
        col >= 0 &&
        col < board.dim &&
        row > 0 &&
        row <= board.dim
      ) {
        const placedTilesArray = Array.from(placedTiles);
        let best = row;
        while (row > 0) {
          --row;
          if (
            board.letters[row * board.dim + col] !==
            EmptyBoardSpaceMachineLetter
          ) {
            // continue
          } else if (placedTilesArray.some(matchesLocation)) {
            best = row;
          } else {
            break;
          }
        }
        if (best !== arrowProperties.row) {
          setArrowProperties({ ...arrowProperties, row: best });
        }
      }
    }

    setPlacedTilesTempScore(0);
    setPlacedTiles(new Set<EphemeralTile>());
    setDisplayedRack(currentRack);
    // Clear pending exchange tiles if present
    if (setPendingExchangeTiles) {
      setPendingExchangeTiles(null);
    }
  }, [
    arrowProperties,
    placedTiles,
    board.dim,
    board.letters,
    currentRack,
    setPlacedTilesTempScore,
    setPlacedTiles,
    setDisplayedRack,
    setArrowProperties,
    setPendingExchangeTiles,
  ]);

  // Shuffle the rack
  const shuffleTiles = useCallback(() => {
    // This assumes a shuffleLetters util is available in scope
    // You may need to import it if not already
    // setDisplayedRack(shuffleLetters(displayedRack));
    // For now, just randomize
    const alistWithGaps = [...displayedRack];
    const alist = alistWithGaps.filter(
      (x) => x !== EmptyRackSpaceMachineLetter,
    );
    const n = alist.length;

    let somethingChanged = false;
    for (let i = n - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      if (alist[i] !== alist[j]) {
        somethingChanged = true;
        const tmp = alist[i];
        alist[i] = alist[j];
        alist[j] = tmp;
      }
    }

    if (!somethingChanged) {
      const j = Math.floor(Math.random() * n);
      const x = [];
      for (let i = 0; i < n; ++i) {
        if (alist[i] !== alist[j]) {
          x.push(i);
        }
      }

      if (x.length > 0) {
        const i = x[Math.floor(Math.random() * x.length)];
        const tmp = alist[i];
        alist[i] = alist[j];
        alist[j] = tmp;
      }
    }

    let r = 0;
    setDisplayedRack(
      alistWithGaps.map((x) =>
        x === EmptyRackSpaceMachineLetter ? x : alist[r++],
      ),
    );
  }, [displayedRack, setDisplayedRack]);

  // Move a tile within the rack
  const moveRackTile = useCallback(
    (newIndex: number | undefined, oldIndex: number | undefined) => {
      if (typeof newIndex === "number" && typeof oldIndex === "number") {
        const leftIndex = Math.min(oldIndex, newIndex);
        const rightIndex = Math.max(oldIndex, newIndex) + 1;
        setDisplayedRack(
          displayedRack
            .slice(0, leftIndex)
            .concat(
              stableInsertRack(
                displayedRack
                  .slice(leftIndex, oldIndex)
                  .concat(EmptyRackSpaceMachineLetter)
                  .concat(displayedRack.slice(oldIndex + 1, rightIndex)),
                newIndex - leftIndex,
                displayedRack[oldIndex],
              ),
            )
            .concat(displayedRack.slice(rightIndex)),
        );
      }
    },
    [displayedRack, setDisplayedRack],
  );

  return {
    recallTiles,
    shuffleTiles,
    moveRackTile,
  };
}
