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
} from '../gen/api/proto/game_service_pb';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  challengeRule: number;
  // rating: number;
  seekID: string;
};

const initialGameState = new GameState(EnglishCrosswordGameDistribution, []);

export type StoreData = {
  // Functions and data to deal with the lobby store.
  soughtGames: Array<SoughtGame>;
  addSoughtGame: (sg: SoughtGame) => void;
  removeGame: (index: number) => void;
  redirGame: string;
  setRedirGame: React.Dispatch<React.SetStateAction<string>>;
  gameHistoryRefresher: (ghr: GameHistoryRefresher) => void;
  gameState: GameState;
  processGameplayEvent: (sge: ServerGameplayEvent) => void;
};

export const Context = createContext<StoreData>({
  soughtGames: [],
  addSoughtGame: () => {},
  removeGame: () => {},
  redirGame: '',
  setRedirGame: () => {},
  gameHistoryRefresher: () => {},
  gameState: initialGameState,
  processGameplayEvent: () => {},
});

type Props = {
  children: React.ReactNode;
};

export const Store = ({ children, ...props }: Props) => {
  const [soughtGames, setSoughtGames] = useState(new Array<SoughtGame>());
  const [redirGame, setRedirGame] = useState('');
  const [gameState, setGameState] = useState(initialGameState);

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

  const store = {
    soughtGames,
    addSoughtGame,
    removeGame,
    redirGame,
    setRedirGame,
    gameState,
    gameHistoryRefresher,
    processGameplayEvent,
  };

  return <Context.Provider value={store}>{children}</Context.Provider>;
};

export function useStoreContext() {
  return useContext(Context);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
