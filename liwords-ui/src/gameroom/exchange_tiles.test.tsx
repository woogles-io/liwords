import React from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';

import { ExchangeTiles } from './exchange_tiles';
import { MachineLetter, MachineWord } from '../utils/cwgame/common';
import { StandardEnglishAlphabet } from '../constants/alphabets';
import { DndProvider } from 'react-dnd';
import { TouchBackend } from 'react-dnd-touch-backend';
import { act } from 'react-dom/test-utils';

function renderExchangeTiles(callback: (t: MachineWord) => void) {
  jest.useFakeTimers();
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
    jest.advanceTimersByTime(500);
  });
  return ret;
}

afterEach(cleanup);

it('exchanges the right tiles', async () => {
  const cb = jest.fn();
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
  const cb = jest.fn();

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
  const cb = jest.fn();

  const { findByRole } = renderExchangeTiles(cb);

  const exchButton = await findByRole('button', { name: 'Exchange' });
  expect(exchButton).toBeVisible();

  fireEvent.keyDown(document.activeElement || document.body, { key: 'M' });
  fireEvent.keyUp(document.activeElement || document.body, { key: 'M' });
  expect(exchButton).toBeDisabled();
  fireEvent.click(exchButton);
  expect(cb).toBeCalledTimes(0);
});
