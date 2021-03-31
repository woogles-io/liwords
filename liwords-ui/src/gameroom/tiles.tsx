import React from 'react';

import {
  CrosswordGameTileValues,
  runeToValues,
} from '../constants/tile_values';
import Tile from './tile';
import {
  Blank,
  EmptySpace,
  EphemeralTile,
  PlayedTiles,
  PlayerOfTiles,
} from '../utils/cwgame/common';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import { useExaminableGameContextStoreContext } from '../store/store';

type Props = {
  gridDim: number;
  tilesLayout: string;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  onClick: (rune: string) => void;
  placementArrow: PlacementArrow;
  scaleTiles: boolean;
  tentativeTiles: Set<EphemeralTile>;
  tentativeTileScore: number | undefined;
  returnToRack?: (
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  handleSetHover?: (
    x: number,
    y: number,
    words: Array<string> | undefined
  ) => void;
  handleUnsetHover?: () => void;
  definitionPopover?:
    | { x: number; y: number; content: React.ReactNode }
    | undefined;
};

const Tiles = React.memo((props: Props) => {
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();

  const tiles = [];
  if (!props.tilesLayout || props.tilesLayout.length === 0) {
    return null;
  }
  // Sort the tentative tiles
  const tentativeTiles = Array.from(props.tentativeTiles.values());
  tentativeTiles.sort((a, b) => {
    if (a.col === b.col) {
      return a.row - b.row;
    }
    return a.col - b.col;
  });

  let tentativeScoreStyle;
  let isHorizontal;
  if (tentativeTiles.length > 0) {
    let minRow = tentativeTiles[0].row;
    let maxRow = minRow;
    let minCol = tentativeTiles[0].col;
    let maxCol = minCol;
    for (let i = 1; i < tentativeTiles.length; ++i) {
      const { row, col } = tentativeTiles[i];
      minRow = Math.min(minRow, row);
      maxRow = Math.max(maxRow, row);
      minCol = Math.min(minCol, col);
      maxCol = Math.max(maxCol, col);
    }
    if (minRow === maxRow) {
      isHorizontal = true;
      // TODO: put Board in useContext and use letterAt instead of indexing directly into tilesLayout?
      if (
        minCol === maxCol &&
        props.placementArrow?.show &&
        !props.placementArrow.horizontal &&
        props.placementArrow.col === minCol
      ) {
        // only one tile was placed, use vertical iff placementArrow says so
        let y = maxRow;
        do ++y;
        while (
          y < props.gridDim &&
          props.tilesLayout[y * props.gridDim + minCol] !== EmptySpace
        );
        if (y === props.placementArrow.row) isHorizontal = false;
      }
    } else if (minCol === maxCol) {
      isHorizontal = false;
    }

    if (isHorizontal === true) {
      let x = minCol;
      do --x;
      while (
        x >= 0 &&
        props.tilesLayout[minRow * props.gridDim + x] !== EmptySpace
      );
      tentativeScoreStyle = { row: minRow, col: x + 1 };
    } else if (isHorizontal === false) {
      let y = minRow;
      do --y;
      while (
        y >= 0 &&
        props.tilesLayout[y * props.gridDim + minCol] !== EmptySpace
      );
      tentativeScoreStyle = { row: y + 1, col: minCol };
    }
  }

  const tentativeBoard = Array.from(new Array(props.gridDim), (_, y) =>
    Array.from(
      new Array(props.gridDim),
      (_, x) => props.tilesLayout[y * props.gridDim + x]
    )
  );
  for (const { row, col, letter } of tentativeTiles) {
    tentativeBoard[row][col] = letter;
  }

  for (let y = 0; y < props.gridDim; y += 1) {
    for (let x = 0; x < props.gridDim; x += 1) {
      const rune = props.tilesLayout[y * props.gridDim + x];
      const tentativeScoreIsHere =
        tentativeScoreStyle &&
        tentativeScoreStyle.row === y &&
        tentativeScoreStyle.col === x;
      const tentativeScoreHere = tentativeScoreIsHere
        ? props.tentativeTileScore
        : undefined;
      const tentativeScoreHereIsHorizontal = tentativeScoreIsHere
        ? isHorizontal
        : undefined;
      const definitionHandlers = {
        ...(props.handleSetHover && {
          onClick: (evt: React.MouseEvent<HTMLElement>) => {
            // if the pointer stays on a tile when a word is played through
            // it, the words being defined are not updated until the
            // pointer is moved out of the tile and back in. this is an
            // intentional design decision to improve usability and
            // responsiveness.
            let sh = '';
            {
              let i = x;
              while (i > 0 && tentativeBoard[y][i - 1] !== EmptySpace) --i;
              for (
                ;
                i < props.gridDim && tentativeBoard[y][i] !== EmptySpace;
                ++i
              ) {
                sh += tentativeBoard[y][i];
              }
            }
            let sv = '';
            {
              let i = y;
              while (i > 0 && tentativeBoard[i - 1][x] !== EmptySpace) --i;
              for (
                ;
                i < props.gridDim && tentativeBoard[i][x] !== EmptySpace;
                ++i
              ) {
                sv += tentativeBoard[i][x];
              }
            }
            const formedWords = [sh, sv].filter((word) => word.length >= 2);
            props.handleSetHover!(
              x,
              y,
              formedWords.length ? formedWords : undefined
            );
          },
          onMouseLeave: (evt: React.MouseEvent<HTMLElement>) => {
            props.handleSetHover!(x, y, undefined);
          },
        }),
        ...(props.definitionPopover &&
          props.definitionPopover.x === x &&
          props.definitionPopover.y === y && {
            onPopoverClick: (evt: React.MouseEvent<HTMLElement>) => {
              props.handleUnsetHover?.();
            },
            popoverContent: props.definitionPopover.content,
          }),
      };

      if (rune !== ' ') {
        const lastPlayed = props.lastPlayedTiles[`R${y}C${x}`] === true;
        const playerOfTile = props.playerOfTileAt[`R${y}C${x}`];
        tiles.push(
          <Tile
            rune={rune}
            value={runeToValues(rune, CrosswordGameTileValues)}
            lastPlayed={lastPlayed}
            playerOfTile={playerOfTile}
            key={`tile_${x}_${y}`}
            scale={props.scaleTiles}
            tentativeScore={tentativeScoreHere}
            tentativeScoreIsHorizontal={tentativeScoreHereIsHorizontal}
            grabbable={false}
            {...definitionHandlers}
          />
        );
      } else {
        const tentativeTile = tentativeTiles.find(
          (tile) => tile.col === x && tile.row === y
        );
        if (tentativeTile) {
          tiles.push(
            <Tile
              onClick={() => {
                // This seems to be used only for undesignated blank.
                // Definition handler will take over for other letters.
                props.onClick(tentativeTile.letter);
              }}
              rune={tentativeTile.letter}
              value={runeToValues(
                tentativeTile.letter,
                CrosswordGameTileValues
              )}
              lastPlayed={false}
              playerOfTile={examinableGameContext.onturn}
              key={`tileT_${tentativeTile.col}_${tentativeTile.row}`}
              scale={false}
              tentative={true}
              x={x}
              y={y}
              tentativeScore={tentativeScoreHere}
              tentativeScoreIsHorizontal={tentativeScoreHereIsHorizontal}
              grabbable={true}
              {...(tentativeTile.letter !== Blank && definitionHandlers)}
            />
          );
        } else {
          tiles.push(
            <div className="empty-space" key={`tile_${x}_${y}`}>
              &nbsp;
            </div>
          );
        }
      }
    }
  }

  return <div className="tiles">{tiles}</div>;
});

export default Tiles;
