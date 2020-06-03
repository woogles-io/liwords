import React, { createContext, useContext, useState } from 'react';
import {
  GameState,
  StateFromHistoryRefresher,
  StateForwarder,
} from '../utils/cwgame/game';
import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import {
  GameHistoryRefresher,
  ServerGameplayEvent,
  ServerChallengeResultEvent,
} from '../gen/api/proto/game_service_pb';

export enum ChatEntityType {
  UserChat,
  ServerMsg,
  ErrorMsg,
}

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  challengeRule: number;
  // rating: number;
  seekID: string;
};

export type ChatEntityObj = {
  entityType: ChatEntityType;
  sender: string;
  message: string;
  id?: string;
};

const MaxChatLength = 250;

const initialGameState = new GameState(EnglishCrosswordGameDistribution, []);

export type StoreData = {
  // Functions and data to deal with the global store.
  soughtGames: Array<SoughtGame>;
  addSoughtGame: (sg: SoughtGame) => void;
  removeGame: (index: number) => void;
  redirGame: string;
  setRedirGame: React.Dispatch<React.SetStateAction<string>>;
  gameHistoryRefresher: (ghr: GameHistoryRefresher) => void;
  gameState: GameState;
  processGameplayEvent: (sge: ServerGameplayEvent) => void;
  challengeResultEvent: (sge: ServerChallengeResultEvent) => void;
  addChat: (chat: ChatEntityObj) => void;
  chat: Array<ChatEntityObj>;
};

export const Context = createContext<StoreData>({
  soughtGames: [],
  chat: [],
  addChat: () => {},
  addSoughtGame: () => {},
  removeGame: () => {},
  redirGame: '',
  setRedirGame: () => {},
  gameHistoryRefresher: () => {},
  gameState: initialGameState,
  processGameplayEvent: () => {},
  challengeResultEvent: () => {},
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
  const [soughtGames, setSoughtGames] = useState(new Array<SoughtGame>());
  const [redirGame, setRedirGame] = useState('');
  const [gameState, setGameState] = useState(initialGameState);
  const [chat, setChat] = useState(new Array<ChatEntityObj>());

  const addSoughtGame = (sg: SoughtGame) => {
    setSoughtGames((state) => [...state, sg]);
  };

  const removeGame = (index: number) => {
    setSoughtGames((state) => {
      const copy = [...state];
      copy.splice(index, 1);
      return copy;
    });
  };

  const gameHistoryRefresher = (ghr: GameHistoryRefresher) => {
    setGameState(StateFromHistoryRefresher(ghr));
  };

  const processGameplayEvent = (sge: ServerGameplayEvent) => {
    setGameState((gs) => {
      return StateForwarder(sge, gs);
    });
  };

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
    chat.push(entity);
    if (chat.length > MaxChatLength) {
      chat.shift();
    }
    setChat(chat);
  };

  const store = {
    soughtGames,
    addSoughtGame,
    removeGame,
    redirGame,
    setRedirGame,
    gameState,
    gameHistoryRefresher,
    processGameplayEvent,
    challengeResultEvent,
    addChat,
    chat,
  };

  return <Context.Provider value={store}>{children}</Context.Provider>;
};

export function useStoreContext() {
  return useContext(Context);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
