import React from 'react';
import { TopBar } from '../navigation/topbar';
import { Chat } from '../chat/chat';
import { Well } from './well';
import './hook.scss';

type Props = {
  sendChat: (msg: string, chan: string) => void;
};

export const HookGame = React.memo((props: Props) => {
  return (
    <>
      <div className="game-container hook-game-container">
        <TopBar />
        <div className="game-table">
          <div className="chat-area" id="left-sidebar">
            <Chat
              sendChat={props.sendChat}
              defaultChannel="lobby"
              defaultDescription=""
              channelTypeOverride="hooks"
              suppressDefault
            />
          </div>
          <div className="play-area game-container">
            <Well />
          </div>
          <div className="data-area" id="right-sidebar"></div>
        </div>
      </div>
    </>
  );
});
