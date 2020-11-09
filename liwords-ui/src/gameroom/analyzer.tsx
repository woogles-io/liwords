import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import { Button, Card } from 'antd';
import { BulbOutlined } from '@ant-design/icons';
import {
  useExaminableGameContextStoreContext,
  useTentativeTileContext,
} from '../store/store';
import { getMacondo } from '../wasm/loader';
import { useMountedState } from '../utils/mounted';
import { RedoOutlined } from '@ant-design/icons/lib';
import { EmptySpace, EphemeralTile } from '../utils/cwgame/common';

type AnalyzerProps = {
  includeCard?: boolean;
  style?: React.CSSProperties;
  lexicon: string;
};

// See analyzer/analyzer.go JsonMove.
type JsonMove = {
  Action: string;
  Row: number; // int
  Column: number; // int
  Vertical: boolean;
  DisplayCoordinates: string;
  Tiles: string;
  Leave: string;
  Equity: number; // float64
  Score: number; // int
};

type AnalyzerMove = {
  displayMove: string;
  coordinates: string;
  leave: string;
  score: number;
  equity: string;
  row: number;
  col: number;
  vertical: boolean;
  tiles: string;
  isExchange: boolean;
};

export const analyzerMoveFromJsonMove = (
  move: JsonMove,
  dim: number,
  letters: string
): AnalyzerMove => {
  let displayMove = '';
  let isExchange = false;
  switch (move.Action) {
    case 'Play': {
      let r = move.Row;
      let c = move.Column;
      let inParen = false;
      for (const t of move.Tiles) {
        if (t === '.') {
          if (!inParen) {
            displayMove += '(';
            inParen = true;
          }
          displayMove += letters[r * dim + c];
        } else {
          if (inParen) {
            displayMove += ')';
            inParen = false;
          }
          displayMove += t;
        }
        if (move.Vertical) ++r;
        else ++c;
      }
      if (inParen) displayMove += ')';
      break;
    }
    case 'Exchange': {
      displayMove = `Exch. ${move.Tiles}`;
      isExchange = true;
      break;
    }
    case 'Pass': {
      displayMove = `Pass`;
      break;
    }
    default: {
      break;
    }
  }
  return {
    displayMove,
    coordinates: move.DisplayCoordinates,
    leave: move.Leave,
    vertical: move.Vertical,
    col: move.Column,
    row: move.Row,
    score: move.Score,
    equity: move.Equity.toFixed(2),
    tiles: move.Tiles,
    isExchange,
  };
};

export const Analyzer = React.memo((props: AnalyzerProps) => {
  const { lexicon } = props;
  const { useState } = useMountedState();
  const [moves, setMoves] = useState(new Array<AnalyzerMove>());
  const [examinerLoading, setExaminerLoading] = useState(false);
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();
  const {
    setDisplayedRack,
    setPlacedTiles,
    setPlacedTilesTempScore,
  } = useTentativeTileContext();

  const examinerId = useRef(0);

  useEffect(() => {
    setExaminerLoading(false);
    examinerId.current = (examinerId.current + 1) | 0;
  }, [moves]);

  const placeMove = useCallback(
    (move) => {
      const {
        board: { dim, letters },
      } = examinableGameContext;
      let newPlacedTiles = new Set<EphemeralTile>();
      let row = move.row;
      let col = move.col;
      let vertical = move.vertical;
      if (move.isExchange) {
        row = 0;
        col = 0;
        vertical = false;
      }
      for (const t of move.tiles) {
        if (move.isExchange) {
          while (letters[row * dim + col] !== EmptySpace) {
            ++col;
            if (col >= dim) {
              ++row;
              if (row >= dim) {
                // Cannot happen with the standard number of tiles and squares.
                row = dim - 1;
                col = dim - 1;
                break;
              }
              col = 0;
            }
          }
        }
        if (t !== '.') {
          newPlacedTiles.add({
            row,
            col,
            letter: t,
          });
        }
        if (vertical) ++row;
        else ++col;
      }
      setDisplayedRack(move.leave);
      setPlacedTiles(newPlacedTiles);
      setPlacedTilesTempScore(move.score);
    },
    [
      examinableGameContext,
      setDisplayedRack,
      setPlacedTiles,
      setPlacedTilesTempScore,
    ]
  );

  const handleExaminer = React.useCallback(() => {
    if (examinerLoading) {
      return;
    }
    const examinerIdAtStart = examinerId.current;
    (async () => {
      const {
        board: { dim, letters },
        onturn,
        players,
      } = examinableGameContext;
      setExaminerLoading(true);
      const boardObj = {
        size: dim,
        rack: players[onturn].currentRack,
        board: Array.from(new Array(dim), (_, row) =>
          letters.substr(row * dim, dim)
        ),
        lexicon,
      };

      const macondo = await getMacondo();
      if (examinerIdAtStart !== examinerId.current) return;
      await macondo.loadLexicon(lexicon);
      if (examinerIdAtStart !== examinerId.current) return;

      const boardStr = JSON.stringify(boardObj);
      const movesStr = await macondo.analyze(boardStr);
      if (examinerIdAtStart !== examinerId.current) return;
      const movesObj = JSON.parse(movesStr) as Array<JsonMove>;

      const formattedMoves = movesObj.map((move) =>
        analyzerMoveFromJsonMove(move, dim, letters)
      );
      setMoves(formattedMoves);
    })();
  }, [examinableGameContext, lexicon, examinerLoading]);

  // When at the last move, examineStoreContext.examinedTurn === Infinity.
  // To also detect new moves, we use examinableGameContext.turns.length.
  useEffect(() => {
    setMoves(new Array<AnalyzerMove>());
  }, [examinableGameContext.turns.length]);

  const renderAnalyzerMoves = useMemo(
    () =>
      moves.map((m: AnalyzerMove, idx) => (
        <tr
          key={idx}
          onClick={() => {
            placeMove(m);
          }}
        >
          <td className="move-coords">{m.coordinates}</td>
          <td className="move">{m.displayMove}</td>
          <td className="move-score">{m.score}</td>
          <td className="move-leave">{m.leave}</td>
          <td className="move-equity">{m.equity}</td>
        </tr>
      )),
    [moves, placeMove]
  );

  const analyzerContainer = (
    <div className="analyzer-container">
      {!examinerLoading ? (
        <div className="suggestions" style={props.style}>
          <table>
            <tbody>{renderAnalyzerMoves}</tbody>
          </table>
        </div>
      ) : (
        <RedoOutlined spin />
      )}
      {!props.includeCard ? (
        <Button
          shape="circle"
          icon={<BulbOutlined />}
          type="primary"
          onClick={handleExaminer}
          disabled={examinerLoading}
        />
      ) : null}
    </div>
  );
  if (props.includeCard) {
    return (
      <Card
        title="Analyzer"
        className="analyzer-card"
        extra={
          <Button
            shape="circle"
            icon={<BulbOutlined />}
            type="primary"
            onClick={handleExaminer}
            disabled={examinerLoading}
          />
        }
      >
        {analyzerContainer}
      </Card>
    );
  }
  return analyzerContainer;
});
