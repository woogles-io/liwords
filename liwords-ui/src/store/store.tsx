import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  useReducer,
  useRef,
} from 'react';

import { EnglishCrosswordGameDistribution } from '../constants/tile_distributions';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
import {
  ServerChallengeResultEvent,
  MatchRequest,
} from '../gen/api/proto/realtime/realtime_pb';
import { LobbyState, LobbyReducer } from './reducers/lobby_reducer';
import { Action } from '../actions/actions';
import {
  GameState,
  pushTurns,
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
  senderId?: string;
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

type LagStoreData = {
  currentLagMs: number;
  setCurrentLagMs: React.Dispatch<React.SetStateAction<number>>;
};

type ExcludedPlayersStoreData = {
  excludedPlayers: Set<string>;
  setExcludedPlayers: React.Dispatch<React.SetStateAction<Set<string>>>;
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

type ExamineStoreData = {
  isExamining: boolean;
  examinedTurn: number;
  handleExamineStart: () => void;
  handleExamineEnd: () => void;
  handleExamineFirst: () => void;
  handleExaminePrev: () => void;
  handleExamineNext: () => void;
  handleExamineLast: () => void;
  handleExamineGoTo: (x: number) => void;
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

const LagContext = createContext<LagStoreData>({
  currentLagMs: NaN,
  setCurrentLagMs: defaultFunction,
});

const ExcludedPlayersContext = createContext<ExcludedPlayersStoreData>({
  // we do not see any messages from excludedPlayers
  excludedPlayers: new Set<string>(),
  setExcludedPlayers: defaultFunction,
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

const [GameContextContext, ExaminableGameContextContext] = Array.from(
  new Array(2),
  () =>
    createContext<GameContextStoreData>({
      gameContext: defaultGameState,
      dispatchGameContext: defaultFunction,
    })
);

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

const [GameEndMessageContext, ExaminableGameEndMessageContext] = Array.from(
  new Array(2),
  () =>
    createContext<GameEndMessageStoreData>({
      gameEndMessage: '',
      setGameEndMessage: defaultFunction,
    })
);

const RematchRequestContext = createContext<RematchRequestStoreData>({
  rematchRequest: new MatchRequest(),
  setRematchRequest: defaultFunction,
});

const [TimerContext, ExaminableTimerContext] = Array.from(new Array(2), () =>
  createContext<TimerStoreData>({
    // initClockController: defaultFunction,
    stopClock: defaultFunction,
    // setClock: defaultFunction,
    timerContext: defaultTimerContext,
    pTimedOut: undefined,
    setPTimedOut: defaultFunction,
  })
);

const PoolFormatContext = createContext<PoolFormatStoreData>({
  poolFormat: PoolFormatType.Alphabet,
  setPoolFormat: defaultFunction,
});

const ExamineContext = createContext<ExamineStoreData>({
  isExamining: false,
  examinedTurn: Infinity,
  handleExamineStart: defaultFunction,
  handleExamineEnd: defaultFunction,
  handleExamineFirst: defaultFunction,
  handleExaminePrev: defaultFunction,
  handleExamineNext: defaultFunction,
  handleExamineLast: defaultFunction,
  handleExamineGoTo: defaultFunction,
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

// Support for examining. Must be nested deeper than the Real Stuffs.

const doNothing = () => {}; // defaultFunction currently is the same as this.

const ExaminableStore = ({ children }: { children: React.ReactNode }) => {
  const gameContextStore = useGameContextStoreContext();
  const gameEndMessageStore = useGameEndMessageStoreContext();
  const timerStore = useTimerStoreContext();

  const { gameContext } = gameContextStore;
  const numberOfTurns = gameContext.turns.length;
  const [isExamining, setIsExamining] = useState(false);
  const [examinedTurn, setExaminedTurn] = useState(Infinity);
  const handleExamineStart = useCallback(() => {
    setIsExamining(true);
  }, []);
  const handleExamineEnd = useCallback(() => {
    setIsExamining(false);
  }, []);
  const handleExamineFirst = useCallback(() => {
    setExaminedTurn(0);
  }, []);
  const handleExaminePrev = useCallback(() => {
    setExaminedTurn((x) => Math.max(Math.min(x, numberOfTurns) - 1, 0));
  }, [numberOfTurns]);
  const handleExamineNext = useCallback(() => {
    setExaminedTurn((x) => (x >= numberOfTurns - 1 ? Infinity : x + 1));
  }, [numberOfTurns]);
  const handleExamineLast = useCallback(() => {
    setExaminedTurn(Infinity);
  }, []);
  const handleExamineGoTo = useCallback(
    (x) => {
      if (x >= numberOfTurns) {
        setExaminedTurn(Infinity);
      } else {
        setExaminedTurn(Math.max(Math.min(x, numberOfTurns), 0));
      }
    },
    [numberOfTurns]
  );

  const examinableGameContext = useMemo(() => {
    if (!isExamining) return gameContext;
    const ret = startingGameState(
      gameContext.tileDistribution,
      gameContext.players.map(({ userID }) => ({
        userID: userID,
        score: 0,
        onturn: false,
        currentRack: '',
      })),
      gameContext.gameID
    );
    ret.nickToPlayerOrder = gameContext.nickToPlayerOrder;
    ret.uidToPlayerOrder = gameContext.uidToPlayerOrder;
    const replayedTurns = gameContext.turns.slice(0, examinedTurn);
    pushTurns(ret, replayedTurns);

    // Fix players and clockController.
    const times = { p0: 0, p1: 0, lastUpdate: 0 };
    for (let i = 0; i < ret.players.length; ++i) {
      const userID = ret.players[i].userID;
      const playerOrder = gameContext.uidToPlayerOrder[userID];
      let nickname = '';
      for (const nick in gameContext.nickToPlayerOrder) {
        if (playerOrder === gameContext.nickToPlayerOrder[nick]) {
          nickname = nick;
          break;
        }
      }

      // Score comes from the most recent past.
      let score = 0;
      for (let j = replayedTurns.length; --j >= 0; ) {
        const turn = gameContext.turns[j];
        if (turn.getNickname() === nickname) {
          score = turn.getCumulative();
          break;
        }
      }
      ret.players[i].score = score;

      // The last few turns may have zero time.
      // Find out the last turn that has nonzero time.
      // (This will also incorrectly eliminate unlikely legitimate events
      // at zero millisecond time at the end.)
      let timeCutoff = gameContext.turns.length;
      while (
        timeCutoff > 0 &&
        gameContext.turns[timeCutoff - 1].getMillisRemaining() === 0
      ) {
        --timeCutoff;
      }

      // Time comes from the most recent past.
      // But may belong to either player, depending on event type.
      let time = Infinity; // No gameInfo here, patch in PlayerCard.
      for (let j = Math.min(timeCutoff, replayedTurns.length); --j >= 0; ) {
        const turn = gameContext.turns[j];

        // Logic from game_reducer setClock.
        let flipTimeRemaining = false;
        if (
          turn.getType() === GameEvent.Type.CHALLENGE_BONUS ||
          turn.getType() === GameEvent.Type.PHONY_TILES_RETURNED
        ) {
          // For these particular two events, the time remaining is for the CHALLENGER.
          // Therefore, it's not the time remaining of the player whose nickname is
          // in the event, so we must flip the times here.
          flipTimeRemaining = true;
        }

        if ((turn.getNickname() === nickname) !== flipTimeRemaining) {
          time = turn.getMillisRemaining();
          break;
        }
      }
      times[playerOrder] = time;

      ret.players[i].onturn = i === ret.onturn;

      // Rack comes from the closest future.
      let rack = gameContext.players[i].currentRack;
      for (let j = replayedTurns.length; j < gameContext.turns.length; ++j) {
        const turn = gameContext.turns[j];
        if (turn.getNickname() === nickname) {
          rack = turn.getRack();
          break;
        }
      }
      ret.players[i].currentRack = rack;
    }
    ret.clockController = {
      current: new ClockController(times, doNothing, doNothing),
    };
    return ret;
  }, [isExamining, examinedTurn, gameContext]);

  const isShowingLatest = !isExamining || examinedTurn >= numberOfTurns;
  const examinableGameContextStore = useMemo(() => {
    return {
      gameContext: examinableGameContext,
      dispatchGameContext: doNothing,
    };
  }, [examinableGameContext]);

  const { gameEndMessage } = gameEndMessageStore;
  const examinableGameEndMessageStore = useMemo(() => {
    return {
      gameEndMessage: isShowingLatest ? gameEndMessage : '',
      setGameEndMessage: doNothing,
    };
  }, [isShowingLatest, gameEndMessage]);

  const shownTimes = isExamining
    ? examinableGameContext.clockController!.current!.times
    : timerStore.timerContext;
  const examinableTimerStore = useMemo(() => {
    return {
      stopClock: doNothing,
      timerContext: shownTimes,
      pTimedOut: undefined,
      setPTimedOut: doNothing,
    };
  }, [shownTimes]);

  const examineContext = useMemo(
    () => ({
      isExamining,
      examinedTurn,
      handleExamineStart,
      handleExamineEnd,
      handleExamineFirst,
      handleExaminePrev,
      handleExamineNext,
      handleExamineLast,
      handleExamineGoTo,
    }),
    [
      isExamining,
      examinedTurn,
      handleExamineStart,
      handleExamineEnd,
      handleExamineFirst,
      handleExaminePrev,
      handleExamineNext,
      handleExamineLast,
      handleExamineGoTo,
    ]
  );

  let ret = children;
  ret = (
    <ExaminableGameContextContext.Provider
      value={examinableGameContextStore}
      children={ret}
    />
  );
  ret = (
    <ExaminableGameEndMessageContext.Provider
      value={examinableGameEndMessageStore}
      children={ret}
    />
  );
  ret = (
    <ExaminableTimerContext.Provider
      value={examinableTimerStore}
      children={ret}
    />
  );
  ret = <ExamineContext.Provider value={examineContext} children={ret} />;

  // typescript did not like "return ret;"
  return <React.Fragment children={ret} />;
};

// The Real Store.

export const Store = ({ children, ...props }: Props) => {
  const stillMountedRef = React.useRef(true);
  React.useEffect(() => () => void (stillMountedRef.current = false), []);

  const clockController = useRef<ClockController | null>(null);

  const onClockTick = useCallback((p: PlayerOrder, t: Millis) => {
    if (!clockController || !clockController.current) {
      return;
    }
    const newCtx = { ...clockController.current!.times, [p]: t };
    if (stillMountedRef.current) {
      setTimerContext(newCtx);
    }
  }, []);

  const onClockTimeout = useCallback((p: PlayerOrder) => {
    if (stillMountedRef.current) {
      setPTimedOut(p);
    }
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
  const [currentLagMs, setCurrentLagMs] = useState(NaN);

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
  const [excludedPlayers, setExcludedPlayers] = useState(new Set<string>());
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

  let ret = <ExaminableStore children={children} />;
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
    <LagContext.Provider
      value={{
        currentLagMs,
        setCurrentLagMs,
      }}
      children={ret}
    />
  );
  ret = (
    <ExcludedPlayersContext.Provider
      value={{
        excludedPlayers,
        setExcludedPlayers,
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
export const useLagStoreContext = () => useContext(LagContext);
export const useExcludedPlayersStoreContext = () =>
  useContext(ExcludedPlayersContext);
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

export const useExaminableGameContextStoreContext = () =>
  useContext(ExaminableGameContextContext);
export const useExaminableGameEndMessageStoreContext = () =>
  useContext(ExaminableGameEndMessageContext);
export const useExaminableTimerStoreContext = () =>
  useContext(ExaminableTimerContext);
export const useExamineStoreContext = () => useContext(ExamineContext);

// https://dev.to/nazmifeeroz/using-usecontext-and-usestate-hooks-as-a-store-mnm
