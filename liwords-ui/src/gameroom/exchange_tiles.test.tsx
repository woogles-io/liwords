import React from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';

import { ExchangeTiles } from './exchange_tiles';
import { MachineLetter, MachineWord } from '../utils/cwgame/common';
import {
  StandardCatalanAlphabet,
  StandardEnglishAlphabet,
} from '../constants/alphabets';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { act } from 'react-dom/test-utils';

function renderExchangeTiles(callback: (t: MachineWord) => void) {
  vi.useFakeTimers();
  const ret = render(
    <DndProvider backend={TouchBackend}>
      <ExchangeTiles
        tileColorId={1}
        rack={[1, 2, 5, 8, 12, 12, 12]} // abehlll
        alphabet={StandardEnglishAlphabet}
        onOk={callback}
        onCancel={() => {}}
        modalVisible={true}
      />
    </DndProvider>
  );
  // there's a delay in ExchangeTiles before it becomes interactive.
  // simulate that here.
  act(() => {
    vi.advanceTimersByTime(500);
  });
  return ret;
}

function renderExchangeCatalanTiles(callback: (t: MachineWord) => void) {
  vi.useFakeTimers();
  const ret = render(
    <DndProvider backend={TouchBackend}>
      <ExchangeTiles
        tileColorId={1}
        rack={[1, 13, 19, 6, 21, 17, 10]} // A L·L QU E S O I
        alphabet={StandardCatalanAlphabet}
        onOk={callback}
        onCancel={() => {}}
        modalVisible={true}
      />
    </DndProvider>
  );
  act(() => {
    vi.advanceTimersByTime(500);
  });
  return ret;
}

afterEach(cleanup);

it('exchanges the right tiles', async () => {
  const cb = vi.fn();
  const { findByRole } = renderExchangeTiles(cb);
  const exchButton = await findByRole('button', { name: 'Exchange' });
  expect(exchButton).toBeVisible();

  fireEvent.keyDown(document.activeElement || document.body, { key: 'B' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'B' });
  fireEvent.keyDown(document.activeElement || document.body, { key: 'E' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'E' });
  expect(exchButton).toBeEnabled();
  fireEvent.click(exchButton);
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(2, 5));
});

it('exchanges repeated tile', async () => {
  const cb = vi.fn();

  const { findByRole } = renderExchangeTiles(cb);

  const exchButton = await findByRole('button', { name: 'Exchange' });
  expect(exchButton).toBeVisible();

  fireEvent.keyDown(document.activeElement || document.body, { key: 'L' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'L' });
  fireEvent.keyDown(document.activeElement || document.body, { key: 'L' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'L' });
  expect(exchButton).toBeEnabled();
  fireEvent.click(exchButton);
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(12, 12));
});

it('ignores non-existing tiles', async () => {
  const cb = vi.fn();

  const { findByRole } = renderExchangeTiles(cb);

  const exchButton = await findByRole('button', { name: 'Exchange' });
  expect(exchButton).toBeVisible();

  fireEvent.keyDown(document.activeElement || document.body, { key: 'M' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'M' });
  expect(exchButton).toBeDisabled();
  fireEvent.click(exchButton);
  expect(cb).toBeCalledTimes(0);
});

it('works with multi-letter tiles and shortcut/alias', async () => {
  const cb = vi.fn();

  const { findByRole } = renderExchangeCatalanTiles(cb);

  const exchButton = await findByRole('button', { name: 'Exchange' });
  expect(exchButton).toBeVisible();

  fireEvent.keyDown(document.activeElement || document.body, { key: 'W' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'W' });
  fireEvent.keyDown(document.activeElement || document.body, { key: 'Q' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'Q' });

  expect(exchButton).toBeEnabled();
  fireEvent.click(exchButton);
  expect(cb).toHaveBeenCalledWith(new Array<MachineLetter>(13, 19));
});
