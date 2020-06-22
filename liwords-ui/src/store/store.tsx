import React, {
  createContext,
  useContext,
  useState,
  useReducer,
  useRef,
} from 'react';

import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import { ServerChallengeResultEvent } from '../gen/api/proto/game_service_pb';
import { LobbyState, LobbyReducer } from './reducers/lobby_reducer';
import { Action } from '../actions/actions';
import {
  GameState,
  startingGameState,
  GameReducer,
} from './reducers/game_reducer';
import { ClockController, Times, Millis } from './timer_controller';
import { PlayerOrder } from './constants';
import { PoolFormatType } from '../constants/pool_formats'

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

const defaultTimerContext = {
  p0: 0,
  p1: 0,
  activePlayer: 'p0' as PlayerOrder,
  lastUpdate: 0,
};
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
  clearChat: () => void;
  chat: Array<ChatEntityObj>;

  // initClockController: (
  //   ghr: GameHistoryRefresher,
  //   onTimeout: () => void
  // ) => void;
  stopClock: () => void;
  // setClock: (sge: ServerGameplayEvent, delay: Centis) => void;
  timerContext: Times;
  poolFormat: PoolFormatType;
  setPoolFormat: (format: PoolFormatType) => void;
  pTimedOut: PlayerOrder | undefined;
  setPTimedOut: (p: PlayerOrder | undefined) => void;
};

const defaultGameState = startingGameState(
  EnglishCrosswordGameDistribution,
  [],
  ''
);

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
  clearChat: defaultFunction,
  chat: [],

  // initClockController: defaultFunction,
  stopClock: defaultFunction,
  // setClock: defaultFunction,
  timerContext: defaultTimerContext,
  poolFormat: PoolFormatType.Alphabet,
  setPoolFormat: defaultFunction,
  pTimedOut: undefined,
  setPTimedOut: defaultFunction,
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

const gameStateInitializer = (
  clockController: React.MutableRefObject<ClockController | null>,
  onClockTick: (p: PlayerOrder, t: Millis) => void,
  onClockTimeout: (p: PlayerOrder) => void
) => {
  const state = defaultGameState;
  state.clockController = clockController;
  state.onClockTick = onClockTick;
  state.onClockTimeout = onClockTimeout;
  return state;
};

export const Store = ({ children, ...props }: Props) => {
  const clockController = useRef<ClockController | null>(null);

  const onClockTick = (p: PlayerOrder, t: Millis) => {
    if (!clockController || !clockController.current) {
      return;
    }
    const newCtx = { ...clockController.current!.times, [p]: t };
    setTimerContext(newCtx);
  };

  const onClockTimeout = (p: PlayerOrder) => {
    setPTimedOut(p);
  };

  const [lobbyContext, dispatchLobbyContext] = useReducer(LobbyReducer, {
    soughtGames: [],
  });

  const [gameContext, dispatchGameContext] = useReducer(GameReducer, null, () =>
    gameStateInitializer(clockController, onClockTick, onClockTimeout)
  );

  const [timerContext, setTimerContext] = useState<Times>(defaultTimerContext);
  const [pTimedOut, setPTimedOut] = useState<PlayerOrder | undefined>(
    undefined
  );

  const [poolFormat, setPoolFormat] = useState<PoolFormatType>(PoolFormatType.Detail);

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

  const clearChat = () => {
    setChat([]);
  };

  const stopClock = () => {
    if (!clockController.current) {
      return;
    }
    clockController.current.stopClock();
    setTimerContext({ ...clockController.current.times });
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
    clearChat,
    chat,

    // initClockController,
    poolFormat,
    setPoolFormat,
    pTimedOut,
    setPTimedOut,
    stopClock,
    timerContext,
  };

  return <Context.Provider value={store}>{children}</Context.Provider>;
};

export function useStoreContext() {
  return useContext(Context);
}

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
