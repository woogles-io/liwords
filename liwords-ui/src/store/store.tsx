import React, { createContext, useContext, useState } from 'react';
import {
  GameState,
  StateFromHistoryRefresher,
  StateForwarder,
  turnSummary,
} from '../utils/cwgame/game';
import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import {
  GameHistoryRefresher,
  ServerGameplayEvent,
  ServerChallengeResultEvent,
} from '../gen/api/proto/game_service_pb';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';

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
  addSoughtGames: (sgs: Array<SoughtGame>) => void;
  removeGame: (id: string) => void;
  redirGame: string;
  setRedirGame: React.Dispatch<React.SetStateAction<string>>;
  gameHistoryRefresher: (ghr: GameHistoryRefresher) => void;
  gameState: GameState;
  processGameplayEvent: (sge: ServerGameplayEvent) => void;
  challengeResultEvent: (sge: ServerChallengeResultEvent) => void;
  addChat: (chat: ChatEntityObj) => void;
  chat: Array<ChatEntityObj>;
  timers: { [username: string]: number | undefined };
  setTimer: (username: string, timeMsec: number) => void;
};

export const Context = createContext<StoreData>({
  soughtGames: [],
  chat: [],
  addChat: () => {},
  addSoughtGame: () => {},
  addSoughtGames: () => {},
  removeGame: () => {},
  redirGame: '',
  setRedirGame: () => {},
  gameHistoryRefresher: () => {},
  gameState: initialGameState,
  processGameplayEvent: () => {},
  challengeResultEvent: () => {},
  timers: {},
  setTimer: () => {},
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
  const [timers, setTimer] = useState({});

  const addSoughtGame = (sg: SoughtGame) => {
    setSoughtGames((state) => [...state, sg]);
  };

  const addSoughtGames = (sgs: Array<SoughtGame>) => {
    setSoughtGames(sgs);
  };

  const removeGame = (id: string) => {
    setSoughtGames((state) => {
      const newArr = state.filter((sg) => {
        return sg.seekID !== id;
      });
      return newArr;
    });
  };

  const gameHistoryRefresher = (ghr: GameHistoryRefresher) => {
    setGameState(StateFromHistoryRefresher(ghr));
  };

  const processGameplayEvent = (sge: ServerGameplayEvent) => {
    setGameState((gs) => {
      return StateForwarder(sge, gs);
    });
    addChat({
      entityType: ChatEntityType.ServerMsg,
      sender: '',
      message: turnSummary(sge),
      id: randomID(),
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
    // XXX: This should be sped up.
    const chatCopy = [...chat];
    chatCopy.push(entity);
    if (chatCopy.length > MaxChatLength) {
      chatCopy.shift();
    }
    setChat(chatCopy);
  };

  const store = {
    soughtGames,
    addSoughtGame,
    addSoughtGames,
    removeGame,
    redirGame,
    setRedirGame,
    gameState,
    gameHistoryRefresher,
    processGameplayEvent,
    challengeResultEvent,
    addChat,
    chat,
    timers,
    setTimer,
  };

  return <Context.Provider value={store}>{children}</Context.Provider>;
};

export function useStoreContext() {
  return useContext(Context);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
