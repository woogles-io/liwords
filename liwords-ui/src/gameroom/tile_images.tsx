import React from 'react';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { useParams } from 'react-router-dom';
import BoardSpace from './board_space';
import Tile from './tile';
import { BonusType } from '../constants/board_layout';
import { alphabetFromName, runeToValues } from '../constants/alphabets';
import { Blank } from '../utils/cwgame/common';

const TileImagesSingle = React.memo((props: { letterDistribution: string }) => {
  // 1. Go to this page (see App.tsx for url).
  //    Note: use 800x600 responsive mode to get compatible styles.
  // 2. Screen-capture normally. Any white border can be cropped later.
  //    On Mac: cmd+shift+4.
  // 3. Crop the white border and assign transparency.
  //    On Mac: brew install netpbm
  // pngtopnm img.png | pnmcrop | pnmtopng -transparent '=#000000' > tiles.png

  const nCols = 15;
  const eachWidth = 34;

  const alphabet = alphabetFromName(props.letterDistribution);
  const letters: Array<string> = [];
  const blankLetters: Array<string> = [];
  const blanks: Array<string> = [];
  for (const { rune } of alphabet.letters) {
    if (rune !== Blank) {
      letters.push(rune);
      blankLetters.push(rune.toLowerCase());
    } else {
      blanks.push(rune);
    }
  }
  const shownRunes = [...letters, ...blankLetters, ...blanks];

  const bonusTypes = [
    BonusType.DoubleWord,
    BonusType.TripleWord,
    //BonusType.QuadrupleWord,
    BonusType.DoubleLetter,
    BonusType.TripleLetter,
    //BonusType.QuadrupleLetter,
    BonusType.StartingSquare,
    BonusType.NoBonus,
  ];

  let y = 0;
  let x = 0;
  let golang: Array<string> = [];
  let currentLine = '';
  let indentLevel = 0;

  const escape = (s: string) => {
    if (s === "'") return "\\'";
    if (s === '"') return '"';
    const t = JSON.stringify(s);
    return t.substring(1, t.length - 1);
  };
  const commitLine = (s?: string) => {
    if (currentLine) golang.push(`${'\t'.repeat(indentLevel)}${currentLine}`);
    if (s != null) golang.push(`${'\t'.repeat(s ? indentLevel : 0)}${s}`);
    currentLine = '';
  };
  const recordPos = (c: string) => {
    if (currentLine) currentLine += ' ';
    else currentLine = '\t';
    currentLine += `'${escape(c)}': {${y}, ${x}},`;
    ++x;
    if (x === nCols) {
      x = 0;
      ++y;
    }
    if (x % 5 === 0) commitLine();
  };

  const groupName = props.letterDistribution || 'english';
  commitLine('// Doubled because of retina screen.');
  commitLine(`const squareDim = 2 * ${eachWidth}`);
  commitLine('');
  ++indentLevel;
  commitLine(`${JSON.stringify(groupName)}: {`);
  ++indentLevel;
  commitLine(`Tile0Src: map[rune][2]int{`);
  for (const c of shownRunes) recordPos(c);
  commitLine('},');
  commitLine(`Tile1Src: map[rune][2]int{`);
  for (const c of shownRunes) recordPos(c);
  commitLine('},');
  commitLine(`BoardSrc: map[rune][2]int{`);
  for (const c of bonusTypes) recordPos(c);
  commitLine('},');
  const nRows = y + (x !== 0 ? 1 : 0);
  commitLine(`NColRows: [2]int{${nCols}, ${nRows}},`);
  --indentLevel;
  commitLine('},');
  console.log(golang.join('\n'));

  return (
    <DndProvider backend={TouchBackend}>
      <div
        style={{
          alignItems: 'center',
          background: '#ffffff',
          display: 'flex',
          height: '100vh',
          justifyContent: 'center',
        }}
      >
        <div
          style={{
            background: '#000000',
            display: 'flex',
            flexWrap: 'wrap',
            width: `${nCols * eachWidth}px`,
          }}
        >
          {Array.from(
            [{ lastPlayed: false }, { lastPlayed: true }],
            (things, idx) => (
              <React.Fragment key={idx}>
                {Array.from(shownRunes, (ch) => (
                  <div key={ch}>
                    <Tile
                      rune={ch}
                      value={runeToValues(alphabet, ch)}
                      playerOfTile={0}
                      key={ch}
                      grabbable={false}
                      {...things}
                    />
                  </div>
                ))}
              </React.Fragment>
            )
          )}
          {bonusTypes.map((bonusType) => (
            <div style={{ minWidth: `${eachWidth}px` }} key={bonusType}>
              <BoardSpace
                bonusType={
                  bonusType === BonusType.StartingSquare
                    ? BonusType.DoubleWord
                    : bonusType
                }
                key={bonusType}
                arrow={false}
                arrowHoriz={false}
                startingSquare={bonusType === BonusType.StartingSquare}
                clicked={() => {}}
                handleTileDrop={(e: any) => {}}
              />
            </div>
          ))}
        </div>
      </div>
    </DndProvider>
  );
});

const TileImages = React.memo((props: {}) => {
  const { letterDistribution } = useParams();
  return <TileImagesSingle letterDistribution={letterDistribution} />;
});

export default TileImages;
