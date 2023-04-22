import { someTileLayout, englishLetterToML } from './scoring.test';
import { getWordsFormed } from './tile_placement';
import { Board } from './board';
import { EphemeralTile } from './common';
import { StandardEnglishAlphabet } from '../../constants/alphabets';

it('getWordsFormed lists all words on board when no tiles are passed', () => {
  const board = new Board();
  board.setTileLayout(someTileLayout);
  const wordsFormed = getWordsFormed(board, undefined, StandardEnglishAlphabet);
  expect(wordsFormed).toHaveLength(8);
  expect(wordsFormed).toContain('ZoNATION');
  expect(wordsFormed).toContain('SI');
});

it('getWordsFormed lists only new words when tiles are passed', () => {
  const board = new Board();
  board.setTileLayout(someTileLayout);
  const horizontalPlay = new Set<EphemeralTile>();
  horizontalPlay.add({
    row: 5,
    col: 9,
    letter: englishLetterToML('E'),
  });
  horizontalPlay.add({
    row: 5,
    col: 10,
    letter: englishLetterToML('S'),
  });
  horizontalPlay.add({
    row: 5,
    col: 11,
    letter: englishLetterToML('T'),
  });
  const wordsFormedHoriz = getWordsFormed(
    board,
    horizontalPlay,
    StandardEnglishAlphabet
  );
  expect(wordsFormedHoriz).toHaveLength(3);
  expect(wordsFormedHoriz).toContain('TEST');
  expect(wordsFormedHoriz).toContain('RE');
  expect(wordsFormedHoriz).toContain('OS');

  const verticalPlay = new Set<EphemeralTile>();
  verticalPlay.add({
    row: 10,
    col: 3,
    letter: englishLetterToML('T'),
  });
  verticalPlay.add({
    row: 11,
    col: 3,
    letter: englishLetterToML('E'),
  });
  verticalPlay.add({
    row: 12,
    col: 3,
    letter: englishLetterToML('s'),
  });
  verticalPlay.add({
    row: 13,
    col: 3,
    letter: englishLetterToML('T'),
  });

  const wordsFormedVertical = getWordsFormed(
    board,
    verticalPlay,
    StandardEnglishAlphabet
  );
  expect(wordsFormedVertical).toHaveLength(3);
  expect(wordsFormedVertical).toContain('TEsT');
  expect(wordsFormedVertical).toContain('TO');
  expect(wordsFormedVertical).toContain('EN');
});
