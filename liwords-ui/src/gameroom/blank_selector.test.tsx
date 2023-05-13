import React from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';
import { MachineLetter } from '../utils/cwgame/common';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { BlankSelector } from './blank_selector';
import {
  Alphabet,
  StandardEnglishAlphabet,
  StandardNorwegianAlphabet,
} from '../constants/alphabets';

function renderSelectTiles(
  alphabet: Alphabet,
  callback: (letter: MachineLetter) => void
) {
  return render(
    <DndProvider backend={TouchBackend}>
      <BlankSelector
        tileColorId={1}
        handleSelection={callback}
        alphabet={alphabet}
      />
    </DndProvider>
  );
}

afterEach(cleanup);

it('selects norwegian tile', async () => {
  const cb = jest.fn();

  const { findByText } = renderSelectTiles(StandardNorwegianAlphabet, cb);

  const Øtile = await findByText('Ø');
  fireEvent.click(Øtile);
  expect(cb).toHaveBeenCalledWith(30);
});

it('selects first english tile', async () => {
  const cb = jest.fn();

  const { findByText } = renderSelectTiles(StandardEnglishAlphabet, cb);

  const atile = await findByText('A');
  fireEvent.click(atile);
  expect(cb).toHaveBeenCalledWith(1);
});
