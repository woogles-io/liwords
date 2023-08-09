import { StandardEnglishAlphabet } from '../../constants/alphabets';
import { Board, toFen } from './board';
import { someTileLayout } from './scoring.test';

it('tests fen', () => {
  const board = new Board();
  board.setTileLayout(someTileLayout);

  const fen = toFen(board, StandardEnglishAlphabet);
  expect(fen).toEqual(
    '9RADIOS/9E5/6R1SI5/6U1E6/4ZINGARO4/4o3T6/4N10/3WASTE7/4T10/4I10/4O10/4N10/15/15/15'
  );
});
