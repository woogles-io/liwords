import React from 'react';
import { cleanup, render } from '@testing-library/react';
import { Chat, Props } from './chat';
import axios from '../__mocks__/axios';

window.RUNTIME_CONFIGURATION = {};

function renderChat(props: Partial<Props> = {}) {
  const dummyFunction = () => {
    return;
  };
  const defaultProps: Props = {
    defaultChannel: 'lobby',
    defaultDescription: 'description',
    sendChat: dummyFunction,
  };
  return render(<Chat {...defaultProps} {...props} />);
}

afterEach(() => {
  axios.post.mockClear();
  cleanup();
});

it('renders the default description', async () => {
  axios.post.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: 'other',
    defaultDescription: 'testDescription',
  });
  const title = await findByTestId('description');
  expect(title).toContainHTML('testDescription');
});

it('renders an appropriate in game description', async () => {
  axios.post.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: 'chat.game.123',
    defaultDescription: 'Will',
  });
  const title = await findByTestId('description');
  expect(title).toContainHTML('Game chat with Will');
});

it('renders an appropriate gameTV description', async () => {
  axios.post.mockResolvedValueOnce({
    data: {
      messages: [],
    },
  });
  const { findByTestId } = renderChat({
    defaultChannel: 'chat.gametv.12345',
    defaultDescription: 'Will vs. César',
  });
  const title = await findByTestId('description');
  expect(title).toContainHTML('Game chat for Will vs. César');
});
