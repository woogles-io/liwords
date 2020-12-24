import React from 'react';
import { cleanup, fireEvent, render } from '@testing-library/react';

import sinon from 'sinon';
import { act } from 'react-dom/test-utils';
import ReactDOM from 'react-dom';
import { Chat, Props } from './chat';

const dummyFn = () => {};

const defaultChatProps: Props = {
  sendChat: dummyFn,
  peopleOnlineContext: (n: number) => `${n} kinsfolk`,
  defaultChannel: '',
  defaultDescription: '',
};

// function renderChat(props: Partial<Props> = {}) {
//   return render(<Chat {...defaultProps} {...props} />);
// }

afterEach(() => {
  cleanup();
  sinon.restore();
});

it('does things', async () => {
  const server = sinon.createFakeServer();

  // server.respondWith(
  //   '/twirp/user_service.SocializeService/GetActiveChatChannels',
  //   `{"channels": ["display_name": "CatCam Tourney", "last_update": "1608756033",
  //       "has_update": true, "last_message": "kitties", "name": "chat.tournament.cats"]}`
  // );
  const chatsData = {
    messages: [
      {
        username: 'cesar',
        channel: 'chat.tournament.cats',
        message: 'a cat is here',
        timestamp: '123456',
        user_id: 'abcdef',
      },
    ],
  };

  server.respondWith(
    'POST',
    '/twirp/user_service.SocializeService/GetChatsForChannel',
    [200, { 'Content-Type': 'application/json' }, JSON.stringify(chatsData)]
  );

  server.respondImmediately = true;
  let findByTestId;

  const el = document.createElement('div');

  await act(async () => {
    ReactDOM.render(
      <Chat
        {...defaultChatProps}
        {...{ defaultChannel: 'chat.tournament.cats' }}
      />,
      el
    );
  });

  const chats = el.querySelector('.entities');

  expect(chats).toBeVisible();
  expect(chats).toContainHTML(
    '<p class="listing-name" data-testid="listing-name">CatCam Tourney</p>'
  );
  server.restore(); // ?
});
