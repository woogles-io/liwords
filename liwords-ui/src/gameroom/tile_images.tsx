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
  // Note: We use Chrome on Mac. Other browsers and other OSes have similar
  // functionalities, but the rendering captured will be slightly different.

  // Steps:
  //  1. Install netpbm. On Mac, this is "brew install netpbm".
  //  2. Adjust browser to use 800x600, 100%, DPR: 2.
  //     On Chrome, go to Inspect mode, activate the Device Toolbar. Set:
  //     Dimensions: Responsive, 800x600, 100%, DPR: 2.
  //     (To set DPR choose "Add device pixel ratio" from the overflow menu.)
  //  3. For each of these:
  //       http://liwords.localhost/tile_images
  //       http://liwords.localhost/tile_images/french
  //       http://liwords.localhost/tile_images/german
  //       http://liwords.localhost/tile_images/norwegian
  //    3.1. In Inspect Elements, find the relevant node (marked in the source),
  //         right-click it, and choose "Capture node screenshot".
  //         Be sure that no selection is highlighted when capturing the screenshot.
  //    3.2. Consolidate the console output into a text file.
  //         These would be useful for pasting into golang.
  //  4. Put the PNGs in a new directory. Quantize and assign transparency.
  //     To use the same palette, use ppmquantall. In bash (and likely zsh):
  //       for x in *.png; do pngtopnm "$x" > "${x%.png}.ppm"; done
  //       ppmquantall 48 *.ppm
  //       mkdir o
  //       for x in *.ppm; do pnmtopng -transparent '=#000000' "$x" > "o/${x%.ppm}.png"; done
  //  5. Check if the images make sense (there's no guarantee #000000 is chosen).
  //  6. Rename and move accordingly to pkg/memento.

  const eachWidth = 34;
  const retina = 2; // device pixel ratio, so 2 = retina, 1 = no retina
  const squareDim = retina * eachWidth;
  const expectedWidth = 20 * squareDim; // retina-adjusted

  const fontSize = 20; // px
  const lineHeight = 1.5;
  const monospacedFontWidth = fontSize * 0.6; // seems correct for "most" monospaced fonts
  const monospacedFontDimY = fontSize * lineHeight * retina;
  const monospacedFontDimX = monospacedFontWidth * retina;

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

  // Not all these characters are immediately needed.
  // The order has no consequence, so optimize for easier eyeballing.
  const textChars = Array.from(
    new Set([
      // digits (48 to 57)
      ...Array.from(new Array(1 + 57 - 48), (_, i) =>
        String.fromCodePoint(48 + i)
      ),
      // runes in the alphabet
      ...alphabet.letters.flatMap(({ rune }) =>
        rune === Blank ? [] : [rune, rune.toLowerCase()]
      ),
      // other ASCII characters (32 to 126)
      ...Array.from(new Array(1 + 126 - 32), (_, i) =>
        String.fromCodePoint(32 + i)
      ),
    ])
  );

  let y = 0;
  let x = 0;
  let yOffset = 0;
  let curDimY = squareDim;
  let curDimX = squareDim;
  let curNumCols = Math.floor(expectedWidth / curDimX);
  const numTileCols = curNumCols;
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
    currentLine += `'${escape(c)}': {${x * curDimX}, ${
      yOffset + y * curDimY
    }},`;
    ++x;
    if (x >= curNumCols) {
      x = 0;
      ++y;
    }
    if (x % 5 === 0) commitLine();
  };

  const groupName = props.letterDistribution || 'english';
  commitLine(`//go:embed tiles-${groupName}.png`);
  commitLine(`var ${groupName}TilesBytes []byte`);
  commitLine('');
  commitLine(`const squareDim = ${squareDim}`);
  commitLine(`const monospacedFontDimX = ${monospacedFontDimX}`);
  commitLine(`const monospacedFontDimY = ${monospacedFontDimY}`);
  commitLine('');
  ++indentLevel;
  commitLine(`${JSON.stringify(groupName)}: {`);
  ++indentLevel;
  commitLine(`TilesBytes: ${groupName}TilesBytes,`);
  commitLine('Tile0Src: map[rune][2]int{');
  for (const c of shownRunes) recordPos(c);
  commitLine('},');
  commitLine('Tile1Src: map[rune][2]int{');
  for (const c of shownRunes) recordPos(c);
  commitLine('},');
  commitLine('BoardSrc: map[rune][2]int{');
  for (const c of bonusTypes) recordPos(c);
  commitLine('},');
  if (x !== 0) {
    ++y;
    x = 0;
  }
  yOffset = y * squareDim;
  y = 0;
  curDimY = monospacedFontDimY;
  curDimX = monospacedFontDimX;
  curNumCols = Math.floor(expectedWidth / curDimX);
  const numTextCols = curNumCols;
  commitLine('TextXSrc: map[rune][2]int{');
  for (const c of textChars) recordPos(c);
  commitLine('},');
  commitLine('Text0Src: map[rune][2]int{');
  for (const c of textChars) recordPos(c);
  commitLine('},');
  commitLine('Text1Src: map[rune][2]int{');
  for (const c of textChars) recordPos(c);
  commitLine('},');
  const nRows = y + (x !== 0 ? 1 : 0);
  commitLine(
    `ExpDimXY: [2]int{${expectedWidth}, ${yOffset + nRows * curDimY}},`
  );
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
          className="CAPTURE-NODE-SCREENSHOT" // To help find this node in Inspector.
          style={{
            background: '#000000',
            display: 'flex',
            flexDirection: 'column',
            width: `${expectedWidth / retina}px`,
          }}
        >
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: `repeat(${numTileCols}, ${eachWidth}px)`,
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
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: `repeat(${numTextCols}, ${monospacedFontWidth}px)`,
            }}
          >
            {Array.from(
              [
                {
                  className: 'tile-images chars',
                },
                {
                  className: 'tile-images chars p0',
                },
                {
                  className: 'tile-images chars p1',
                },
              ],
              (things, idx) => (
                <React.Fragment key={idx}>
                  {Array.from(textChars, (c, i) => (
                    <React.Fragment key={i}>
                      <div
                        style={{
                          fontSize: `${fontSize}px`,
                          fontWeight: 'normal',
                          lineHeight: `${lineHeight}`,
                          whiteSpace: 'pre',
                        }}
                        {...things}
                      >
                        {c}
                      </div>
                    </React.Fragment>
                  ))}
                </React.Fragment>
              )
            )}
          </div>
        </div>
      </div>
    </DndProvider>
  );
});

const TileImages = React.memo((props: {}) => {
  const { letterDistribution } = useParams();
  return <TileImagesSingle letterDistribution={letterDistribution || ''} />;
});

export default TileImages;
