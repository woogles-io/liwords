import React, {
  createContext,
  useCallback,
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
import { LoginState, LoginStateReducer } from './login_state';

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

// Functions and data to deal with the global store.

type LobbyStoreData = {
  lobbyContext: LobbyState;
  dispatchLobbyContext: (action: Action) => void;
};

type LoginStateStoreData = {
  loginState: LoginState;
  dispatchLoginState: (action: Action) => void;
};

type RedirGameStoreData = {
  redirGame: string;
  setRedirGame: React.Dispatch<React.SetStateAction<string>>;
};

type ChallengeResultEventStoreData = {
  challengeResultEvent: (sge: ServerChallengeResultEvent) => void;
};

type GameContextStoreData = {
  gameContext: GameState;
  dispatchGameContext: (action: Action) => void;
};

type ChatStoreData = {
  addChat: (chat: ChatEntityObj) => void;
  addChats: (chats: Array<ChatEntityObj>) => void;
  clearChat: () => void;
  chat: Array<ChatEntityObj>;
};

type PresenceStoreData = {
  setPresence: (presence: PresenceEntity) => void;
  addPresences: (presences: Array<PresenceEntity>) => void;
  presences: { [uuid: string]: PresenceEntity };
};

type GameEndMessageStoreData = {
  // This variable is set when the game just ended.
  gameEndMessage: string;
  setGameEndMessage: React.Dispatch<React.SetStateAction<string>>;
};

type RematchRequestStoreData = {
  rematchRequest: MatchRequest;
  setRematchRequest: React.Dispatch<React.SetStateAction<MatchRequest>>;
};

type TimerStoreData = {
  // initClockController: (
  //   ghr: GameHistoryRefresher,
  //   onTimeout: () => void
  // ) => void;
  stopClock: () => void;
  // setClock: (sge: ServerGameplayEvent, delay: Centis) => void;
  timerContext: Times;
  pTimedOut: PlayerOrder | undefined;
  setPTimedOut: (p: PlayerOrder | undefined) => void;
};

type PoolFormatStoreData = {
  poolFormat: PoolFormatType;
  setPoolFormat: (format: PoolFormatType) => void;
};

const defaultGameState = startingGameState(
  EnglishCrosswordGameDistribution,
  [],
  ''
);

// This is annoying, but we have to add a default for everything in this
// declaration. Declaring it as a Partial<StoreData> breaks things elsewhere.
// For context, these used to be a single StoreData that contained everything.

const LobbyContext = createContext<LobbyStoreData>({
  lobbyContext: {
    soughtGames: [],
    activeGames: [],
    matchRequests: [],
  },
  dispatchLobbyContext: defaultFunction,
});

const LoginStateContext = createContext<LoginStateStoreData>({
  loginState: {
    username: '',
    userID: '',
    loggedIn: false,
    connectedToSocket: false,
    connID: '',
  },
  dispatchLoginState: defaultFunction,
});

const RedirGameContext = createContext<RedirGameStoreData>({
  redirGame: '',
  setRedirGame: defaultFunction,
});

const ChallengeResultEventContext = createContext<
  ChallengeResultEventStoreData
>({
  challengeResultEvent: defaultFunction,
});

const GameContextContext = createContext<GameContextStoreData>({
  gameContext: defaultGameState,
  dispatchGameContext: defaultFunction,
});

const ChatContext = createContext<ChatStoreData>({
  addChat: defaultFunction,
  addChats: defaultFunction,
  clearChat: defaultFunction,
  chat: [],
});

const PresenceContext = createContext<PresenceStoreData>({
  setPresence: defaultFunction,
  addPresences: defaultFunction,
  presences: {},
});

const GameEndMessageContext = createContext<GameEndMessageStoreData>({
  gameEndMessage: '',
  setGameEndMessage: defaultFunction,
});

const RematchRequestContext = createContext<RematchRequestStoreData>({
  rematchRequest: new MatchRequest(),
  setRematchRequest: defaultFunction,
});

const TimerContext = createContext<TimerStoreData>({
  // initClockController: defaultFunction,
  stopClock: defaultFunction,
  // setClock: defaultFunction,
  timerContext: defaultTimerContext,
  pTimedOut: undefined,
  setPTimedOut: defaultFunction,
});

const PoolFormatContext = createContext<PoolFormatStoreData>({
  poolFormat: PoolFormatType.Alphabet,
  setPoolFormat: defaultFunction,
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

  const onClockTick = useCallback((p: PlayerOrder, t: Millis) => {
    if (!clockController || !clockController.current) {
      return;
    }
    const newCtx = { ...clockController.current!.times, [p]: t };
    setTimerContext(newCtx);
  }, []);

  const onClockTimeout = useCallback((p: PlayerOrder) => {
    setPTimedOut(p);
  }, []);

  const [lobbyContext, dispatchLobbyContext] = useReducer(LobbyReducer, {
    soughtGames: [],
    activeGames: [],
    matchRequests: [],
  });
  const [loginState, dispatchLoginState] = useReducer(LoginStateReducer, {
    username: '',
    userID: '',
    loggedIn: false,
    connectedToSocket: false,
    connID: '',
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

  const addChat = useCallback((entity: ChatEntityObj) => {
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
  }, []);

  const challengeResultEvent = useCallback(
    (sge: ServerChallengeResultEvent) => {
      console.log('sge', sge);
      addChat({
        entityType: ChatEntityType.ServerMsg,
        sender: '',
        message: sge.getValid()
          ? 'Challenged play was valid'
          : 'Play was challenged off the board!',
        id: randomID(),
      });
    },
    [addChat]
  );

  const addChats = useCallback((entities: Array<ChatEntityObj>) => {
    setChat([...entities]);
  }, []);

  const clearChat = useCallback(() => {
    setChat([]);
  }, []);

  const setPresence = useCallback((entity: PresenceEntity) => {
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
  }, []);

  const addPresences = useCallback((entities: Array<PresenceEntity>) => {
    const presencesCopy = {} as { [uuid: string]: PresenceEntity };
    entities.forEach((p) => {
      presencesCopy[p.uuid] = p;
    });
    console.log('in addPresences', presencesCopy);

    setPresences(presencesCopy);
  }, []);

  const stopClock = useCallback(() => {
    if (!clockController.current) {
      return;
    }
    clockController.current.stopClock();
    setTimerContext({ ...clockController.current.times });
  }, []);

  let ret = children;
  ret = (
    <LobbyContext.Provider
      value={{
        lobbyContext,
        dispatchLobbyContext,
      }}
      children={ret}
    />
  );
  ret = (
    <LoginStateContext.Provider
      value={{
        loginState,
        dispatchLoginState,
      }}
      children={ret}
    />
  );
  ret = (
    <RedirGameContext.Provider
      value={{
        redirGame,
        setRedirGame,
      }}
      children={ret}
    />
  );
  ret = (
    <ChallengeResultEventContext.Provider
      value={{
        challengeResultEvent,
      }}
      children={ret}
    />
  );
  ret = (
    <GameContextContext.Provider
      value={{
        gameContext,
        dispatchGameContext,
      }}
      children={ret}
    />
  );
  ret = (
    <ChatContext.Provider
      value={{
        addChat,
        addChats,
        clearChat,
        chat,
      }}
      children={ret}
    />
  );
  ret = (
    <PresenceContext.Provider
      value={{
        setPresence,
        addPresences,
        presences,
      }}
      children={ret}
    />
  );
  ret = (
    <GameEndMessageContext.Provider
      value={{
        gameEndMessage,
        setGameEndMessage,
      }}
      children={ret}
    />
  );
  ret = (
    <RematchRequestContext.Provider
      value={{
        rematchRequest,
        setRematchRequest,
      }}
      children={ret}
    />
  );
  ret = (
    <TimerContext.Provider
      value={{
        // initClockController,
        stopClock,
        timerContext,
        pTimedOut,
        setPTimedOut,
      }}
      children={ret}
    />
  );
  ret = (
    <PoolFormatContext.Provider
      value={{
        poolFormat,
        setPoolFormat,
      }}
      children={ret}
    />
  );

  // typescript did not like "return ret;"
  return <React.Fragment children={ret} />;
};

export const useLobbyStoreContext = () => useContext(LobbyContext);
export const useLoginStateStoreContext = () => useContext(LoginStateContext);
export const useRedirGameStoreContext = () => useContext(RedirGameContext);
export const useChallengeResultEventStoreContext = () =>
  useContext(ChallengeResultEventContext);
export const useGameContextStoreContext = () => useContext(GameContextContext);
export const useChatStoreContext = () => useContext(ChatContext);
export const usePresenceStoreContext = () => useContext(PresenceContext);
export const useGameEndMessageStoreContext = () =>
  useContext(GameEndMessageContext);
export const useRematchRequestStoreContext = () =>
  useContext(RematchRequestContext);
export const useTimerStoreContext = () => useContext(TimerContext);
export const usePoolFormatStoreContext = () => useContext(PoolFormatContext);

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
