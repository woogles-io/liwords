import React from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';
import GameControls, { Props } from './game_controls';

function renderGameControls(props: Partial<Props> = {}) {
  const dummyFunction = () => {
    return;
  };
  const defaultProps: Props = {
    challengeRule: 'FIVE_POINT',
    exchangeAllowed: false,
    isExamining: false,
    finalPassOrChallenge: false,
    myTurn: false,
    observer: false,
    lexicon: 'dummy',
    showExchangeModal: dummyFunction,
    onExportGCG: dummyFunction,
    onPass: dummyFunction,
    onResign: dummyFunction,
    onRecall: dummyFunction,
    onChallenge: dummyFunction,
    onCommit: dummyFunction,
    onExamine: dummyFunction,
    onRematch: dummyFunction,
    gameEndControls: false,
    showRematch: false,
    currentRack: '',
  };
  return render(<GameControls {...defaultProps} {...props} />);
}

afterEach(cleanup);

it('fires clicks on rematch only once', async () => {
  const onRematch = jest.fn();
  const { findByTestId } = renderGameControls({
    gameEndControls: true,
    onRematch: onRematch,
    showRematch: true,
  });
  const rematchButton = await findByTestId('rematch-button');
  expect(rematchButton).toBeVisible();
  fireEvent.click(rematchButton);
  fireEvent.click(rematchButton);
  expect(onRematch).toHaveBeenCalledTimes(1);
});
