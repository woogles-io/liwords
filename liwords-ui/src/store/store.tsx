import React, {
  createContext,
  useContext,
  useState,
  useReducer,
  useRef,
} from 'react';

import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import {
  ServerChallengeResultEvent,
  MatchRequest,
} from '../gen/api/proto/realtime/realtime_pb';
import { LobbyState, LobbyReducer } from './reducers/lobby_reducer';
import { Action } from '../actions/actions';
import {
  GameState,
  startingGameState,
  GameReducer,
} from './reducers/game_reducer';
import { ClockController, Times, Millis } from './timer_controller';
import { PlayerOrder } from './constants';
import { PoolFormatType } from '../constants/pool_formats';

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
  timestamp?: number;
};

export type PresenceEntity = {
  uuid: string;
  username: string;
  channel: string;
  anon: boolean;
};

const MaxChatLength = 100;

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
  addChats: (chats: Array<ChatEntityObj>) => void;
  clearChat: () => void;
  chat: Array<ChatEntityObj>;

  setPresence: (presence: PresenceEntity) => void;
  addPresences: (presences: Array<PresenceEntity>) => void;
  presences: { [uuid: string]: PresenceEntity };

  // This variable is set when the game just ended.
  gameEndMessage: string;
  setGameEndMessage: React.Dispatch<React.SetStateAction<string>>;

  rematchRequest: MatchRequest;
  setRematchRequest: React.Dispatch<React.SetStateAction<MatchRequest>>;

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
  lobbyContext: { soughtGames: [], activeGames: [], matchRequests: [] },
  dispatchLobbyContext: defaultFunction,
  redirGame: '',
  setRedirGame: defaultFunction,

  challengeResultEvent: defaultFunction,
  gameContext: defaultGameState,
  dispatchGameContext: defaultFunction,

  addChat: defaultFunction,
  addChats: defaultFunction,
  clearChat: defaultFunction,
  chat: [],

  setPresence: defaultFunction,
  addPresences: defaultFunction,
  presences: {},

  gameEndMessage: '',
  setGameEndMessage: defaultFunction,

  rematchRequest: new MatchRequest(),
  setRematchRequest: defaultFunction,

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

export const randomID = () => {
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
    activeGames: [],
    matchRequests: [],
  });

  const [gameContext, dispatchGameContext] = useReducer(GameReducer, null, () =>
    gameStateInitializer(clockController, onClockTick, onClockTimeout)
  );

  const [timerContext, setTimerContext] = useState<Times>(defaultTimerContext);
  const [pTimedOut, setPTimedOut] = useState<PlayerOrder | undefined>(
    undefined
  );

  const [poolFormat, setPoolFormat] = useState<PoolFormatType>(
    PoolFormatType.Alphabet
  );

  const [redirGame, setRedirGame] = useState('');
  const [gameEndMessage, setGameEndMessage] = useState('');
  const [rematchRequest, setRematchRequest] = useState(new MatchRequest());
  const [chat, setChat] = useState(new Array<ChatEntityObj>());
  const [presences, setPresences] = useState(
    {} as { [uuid: string]: PresenceEntity }
  );

  const challengeResultEvent = (sge: ServerChallengeResultEvent) => {
    console.log('sge', sge);
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
    setChat((oldChat) => {
      if (!entity.id) {
        // eslint-disable-next-line no-param-reassign
        entity.id = randomID();
      }
      // XXX: This should be sped up.
      const chatCopy = [...oldChat];
      chatCopy.push(entity);
      if (chatCopy.length > MaxChatLength) {
        chatCopy.shift();
      }
      return chatCopy;
    });
  };

  const addChats = (entities: Array<ChatEntityObj>) => {
    setChat([...entities]);
  };

  const clearChat = () => {
    setChat([]);
  };

  const setPresence = (entity: PresenceEntity) => {
    // XXX: This looks slow.
    setPresences((prevPresences) => {
      const presencesCopy = { ...prevPresences };
      if (entity.channel === '') {
        // This user signed off; remove
        delete presencesCopy[entity.uuid];
      } else {
        presencesCopy[entity.uuid] = entity;
      }
      return presencesCopy;
    });
  };

  const addPresences = (entities: Array<PresenceEntity>) => {
    const presencesCopy = {} as { [uuid: string]: PresenceEntity };
    entities.forEach((p) => {
      presencesCopy[p.uuid] = p;
    });
    console.log('in addPresences', presencesCopy);

    setPresences(presencesCopy);
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
    gameEndMessage,
    setGameEndMessage,
    challengeResultEvent,
    addChat,
    addChats,
    clearChat,
    chat,
    setPresence,
    addPresences,
    presences,

    rematchRequest,
    setRematchRequest,

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
