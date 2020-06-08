import React, { createContext, useContext, useState, useReducer } from 'react';
// import {
//   GameState,
//   StateFromHistoryRefresher,
//   StateForwarder,
//   turnSummary,
// } from '../utils/cwgame/game';
import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import { ServerChallengeResultEvent } from '../gen/api/proto/game_service_pb';
// import { Reducer } from './reducers/main';
import { LobbyState, LobbyReducer } from './reducers/lobby_reducer';
import { Action } from '../actions/actions';
import {
  GameState,
  startingGameState,
  GameReducer,
} from './reducers/game_reducer';

export enum ChatEntityType {
  UserChat,
  ServerMsg,
  ErrorMsg,
}

export type ChatEntityObj = {
  entityType: ChatEntityType;
  sender: string;
  message: string;
  id?: string;
};

const MaxChatLength = 250;
const defaultGameState = startingGameState(
  EnglishCrosswordGameDistribution,
  [],
  ''
);
const defaultFunction = () => {};

export type StoreData = {
  // Functions and data to deal with the global store.
  lobbyContext: LobbyState;
  dispatchLobbyContext: (action: Action) => void;
  redirGame: string;
  setRedirGame: React.Dispatch<React.SetStateAction<string>>;

  challengeResultEvent: (sge: ServerChallengeResultEvent) => void;

  gameContext: GameState;
  dispatchGameContext: (action: Action) => void;

  addChat: (chat: ChatEntityObj) => void;
  chat: Array<ChatEntityObj>;
};

// This is annoying, but we have to add a default for everything in this
// declaration. Declaring it as a Partial<StoreData> breaks things elsewhere.
export const Context = createContext<StoreData>({
  lobbyContext: { soughtGames: [] },
  dispatchLobbyContext: defaultFunction,
  redirGame: '',
  setRedirGame: defaultFunction,
  challengeResultEvent: defaultFunction,
  gameContext: defaultGameState,
  dispatchGameContext: defaultFunction,

  addChat: defaultFunction,
  chat: [],
});

type Props = {
  children: React.ReactNode;
};

const randomID = () => {
  // Math.random should be unique because of its seeding algorithm.
  // Convert it to base 36 (numbers + letters), and grab the first 9 characters
  // after the decimal.
  return `_${Math.random().toString(36).substr(2, 9)}`;
};

export const Store = ({ children, ...props }: Props) => {
  const [lobbyContext, dispatchLobbyContext] = useReducer(LobbyReducer, {
    soughtGames: [],
  });

  const [gameContext, dispatchGameContext] = useReducer(
    GameReducer,
    defaultGameState
  );

  const [redirGame, setRedirGame] = useState('');
  const [chat, setChat] = useState(new Array<ChatEntityObj>());

  const challengeResultEvent = (sge: ServerChallengeResultEvent) => {
    addChat({
      entityType: ChatEntityType.ServerMsg,
      sender: '',
      message: sge.getValid()
        ? 'Challenged play was valid'
        : 'Play was challenged off the board!',
      id: randomID(),
    });
  };

  const addChat = (entity: ChatEntityObj) => {
    if (!entity.id) {
      // eslint-disable-next-line no-param-reassign
      entity.id = randomID();
    }
    // XXX: This should be sped up.
    const chatCopy = [...chat];
    chatCopy.push(entity);
    if (chatCopy.length > MaxChatLength) {
      chatCopy.shift();
    }
    setChat(chatCopy);
  };

  const store = {
    lobbyContext,
    dispatchLobbyContext,
    gameContext,
    dispatchGameContext,
    redirGame,
    setRedirGame,
    challengeResultEvent,
    addChat,
    chat,
    // timers,
    // setTimer,
  };

  return <Context.Provider value={store}>{children}</Context.Provider>;
};

export function useStoreContext() {
  return useContext(Context);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
