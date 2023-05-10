import React from 'react';

import Tile from './tile';
import {
  BlankMachineLetter,
  EmptyBoardSpaceMachineLetter,
  EphemeralTile,
  MachineLetter,
  PlayedTiles,
  PlayerOfTiles,
} from '../utils/cwgame/common';
import { PlacementArrow } from '../utils/cwgame/tile_placement';
import { Alphabet, machineWordToRunes, scoreFor } from '../constants/alphabets';

type Props = {
  tileColorId: number;
  gridDim: number;
  tilesLayout: Array<MachineLetter>;
  alphabet: Alphabet;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  onClick: (letter: MachineLetter) => void;
  placementArrow: PlacementArrow;
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
  handleTileDrop?: (
    row: number,
    col: number,
    rackIndex: number | undefined,
    tileIndex: number | undefined
  ) => void;
  recallOneTile?: (row: number, col: number) => void;
};

const Tiles = React.memo((props: Props) => {
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
          props.tilesLayout[y * props.gridDim + minCol] !==
            EmptyBoardSpaceMachineLetter
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
        props.tilesLayout[minRow * props.gridDim + x] !==
          EmptyBoardSpaceMachineLetter
      );
      tentativeScoreStyle = { row: minRow, col: x + 1 };
    } else if (isHorizontal === false) {
      let y = minRow;
      do --y;
      while (
        y >= 0 &&
        props.tilesLayout[y * props.gridDim + minCol] !==
          EmptyBoardSpaceMachineLetter
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
      const letter = props.tilesLayout[y * props.gridDim + x];
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
            const sh = new Array<MachineLetter>();
            {
              let i = x;
              while (
                i > 0 &&
                tentativeBoard[y][i - 1] !== EmptyBoardSpaceMachineLetter
              )
                --i;
              for (
                ;
                i < props.gridDim &&
                tentativeBoard[y][i] !== EmptyBoardSpaceMachineLetter;
                ++i
              ) {
                sh.push(tentativeBoard[y][i]);
              }
            }
            const sv = new Array<MachineLetter>();
            {
              let i = y;
              while (
                i > 0 &&
                tentativeBoard[i - 1][x] !== EmptyBoardSpaceMachineLetter
              )
                --i;
              for (
                ;
                i < props.gridDim &&
                tentativeBoard[i][x] !== EmptyBoardSpaceMachineLetter;
                ++i
              ) {
                sv.push(tentativeBoard[i][x]);
              }
            }
            const formedWords = [sh, sv].filter((word) => word.length >= 2);
            props.handleSetHover?.(
              x,
              y,
              formedWords.length
                ? formedWords.map((w) => machineWordToRunes(w, props.alphabet))
                : undefined
            );
          },
          onMouseLeave: (evt: React.MouseEvent<HTMLElement>) => {
            props.handleSetHover?.(x, y, undefined);
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

      if (letter !== EmptyBoardSpaceMachineLetter) {
        const lastPlayed = props.lastPlayedTiles[`R${y}C${x}`] === true;
        const playerOfTile = props.playerOfTileAt[`R${y}C${x}`];
        tiles.push(
          <Tile
            letter={letter}
            alphabet={props.alphabet}
            value={scoreFor(props.alphabet, letter)}
            lastPlayed={lastPlayed}
            playerOfTile={playerOfTile}
            key={`tile_${x}_${y}`}
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
              onContextMenu={(evt: React.MouseEvent<HTMLElement>) => {
                if (!evt.shiftKey) {
                  // Recall tile when not holding shift.
                  evt.preventDefault();
                  if (props.recallOneTile) {
                    props.recallOneTile(tentativeTile.row, tentativeTile.col);
                  }
                } else {
                  // Shift+RightClick accesses context menu.
                }
              }}
              onClick={() => {
                // This seems to be used only for undesignated blank.
                // Definition handler will take over for other letters.
                props.onClick(tentativeTile.letter);
              }}
              letter={tentativeTile.letter}
              value={scoreFor(props.alphabet, tentativeTile.letter)}
              lastPlayed={false}
              playerOfTile={props.tileColorId}
              key={`tileT_${tentativeTile.col}_${tentativeTile.row}`}
              tentative={true}
              x={x}
              y={y}
              tentativeScore={tentativeScoreHere}
              tentativeScoreIsHorizontal={tentativeScoreHereIsHorizontal}
              grabbable={true}
              handleTileDrop={props.handleTileDrop}
              alphabet={props.alphabet}
              {...(tentativeTile.letter !== BlankMachineLetter &&
                definitionHandlers)}
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
