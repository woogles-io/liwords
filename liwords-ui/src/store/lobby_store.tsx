import React, { createContext, useContext, useState } from 'react';

export type SoughtGame = {
  seeker: string;
  lexicon: string;
  initialTimeSecs: number;
  challengeRule: number;
  // rating: number;
  // soughtID: string;
};

export type LobbyStoreData = {
  // Functions and data to deal with the lobby store.
  soughtGames: Array<SoughtGame>;
  addSoughtGame: (sg: SoughtGame) => void;
  removeGame: (index: number) => void;
};

export const LobbyContext = createContext<LobbyStoreData>({
  soughtGames: [],
  addSoughtGame: () => {},
  removeGame: () => {},
});

type Props = {
  children: React.ReactNode;
};

export const LobbyStore = ({ children, ...props }: Props) => {
  console.log('initalizing lobbystore');
  const [soughtGames, setSoughtGames] = useState(new Array<SoughtGame>());

  const addSoughtGame = (sg: SoughtGame) => {
    console.log('adding southg game', sg, 'southggames are', soughtGames);
    setSoughtGames([...soughtGames, sg]);
  };

  const removeGame = (index: number) => {
    console.log('removing game at index', index);
    const copy = [...soughtGames];
    copy.splice(index, 1);
    setSoughtGames(copy);
  };

  const store = {
    soughtGames,
    addSoughtGame,
    removeGame,
  };

  return (
    <LobbyContext.Provider value={store}>{children}</LobbyContext.Provider>
  );
};

export function useLobbyContext() {
  return useContext(LobbyContext);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
