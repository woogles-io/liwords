import React, { createContext, useCallback, useContext, useMemo } from 'react';
import { useMountedState } from '../utils/mounted';

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
import { EphemeralTile } from '../utils/cwgame/common';
import { ActiveChatChannels } from '../gen/api/proto/user_service/user_service_pb';
import {
  defaultTournamentState,
  TournamentReducer,
  TournamentState,
} from './reducers/tournament_reducer';
import { defaultTimerContext, useTimer } from './use_timer';
import { MetaEventState, MetaStates } from './meta_game_events';

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
  channel: string;
};

export type PresenceEntity = {
  uuid: string;
  username: string;
  channel: string;
  anon: boolean;
  deleting: boolean;
};

const MaxChatLength = 150;

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
  excludedPlayersFetched: boolean;
  setExcludedPlayersFetched: React.Dispatch<React.SetStateAction<boolean>>;
  pendingBlockRefresh: boolean;
  setPendingBlockRefresh: React.Dispatch<React.SetStateAction<boolean>>;
};

type ChallengeResultEventStoreData = {
  challengeResultEvent: (sge: ServerChallengeResultEvent) => void;
};

type GameMetaEventStoreData = {
  gameMetaEventContext: MetaEventState;
  setGameMetaEventContext: React.Dispatch<React.SetStateAction<MetaEventState>>;
};

type GameContextStoreData = {
  gameContext: GameState;
  dispatchGameContext: (action: Action) => void;
};

type ChatStoreData = {
  addChat: (chat: ChatEntityObj) => void;
  addChats: (chats: Array<ChatEntityObj>) => void;
  clearChat: () => void;
  deleteChat: (id: string, channel: string) => void;
  chat: Array<ChatEntityObj>;
  chatChannels: ActiveChatChannels.AsObject | undefined;
  setChatChannels: (chatChannels: ActiveChatChannels.AsObject) => void;
};

type PresenceStoreData = {
  setPresence: (presence: PresenceEntity) => void;
  addPresences: (presences: Array<PresenceEntity>) => void;
  presences: Array<PresenceEntity>;
};

type TournamentStoreData = {
  tournamentContext: TournamentState;
  dispatchTournamentContext: (action: Action) => void;
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

type TentativePlayData = {
  placedTilesTempScore: number | undefined;
  placedTiles: Set<EphemeralTile>;
  displayedRack: string;
  setPlacedTilesTempScore: (s: number | undefined) => void;
  setPlacedTiles: (t: Set<EphemeralTile>) => void;
  setDisplayedRack: (l: string) => void;
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
  addHandleExaminer: (x: () => void) => void;
  removeHandleExaminer: (x: () => void) => void;
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
    path: '',
    perms: [],
  },
  dispatchLoginState: defaultFunction,
});

const LagContext = createContext<LagStoreData>({
  currentLagMs: NaN,
  setCurrentLagMs: defaultFunction,
});

const TentativePlayContext = createContext<TentativePlayData>({
  placedTilesTempScore: undefined,
  placedTiles: new Set<EphemeralTile>(),
  displayedRack: '',
  setPlacedTilesTempScore: defaultFunction,
  setPlacedTiles: defaultFunction,
  setDisplayedRack: defaultFunction,
});

const ExcludedPlayersContext = createContext<ExcludedPlayersStoreData>({
  // we do not see any messages from excludedPlayers
  excludedPlayers: new Set<string>(),
  setExcludedPlayers: defaultFunction,
  excludedPlayersFetched: false,
  setExcludedPlayersFetched: defaultFunction,
  pendingBlockRefresh: false,
  setPendingBlockRefresh: defaultFunction,
});

const ChallengeResultEventContext = createContext<
  ChallengeResultEventStoreData
>({
  challengeResultEvent: defaultFunction,
});

const GameMetaEventContext = createContext<GameMetaEventStoreData>({
  gameMetaEventContext: {
    curEvt: MetaStates.NO_ACTIVE_REQUEST,
    // timer: null,
  },
  setGameMetaEventContext: defaultFunction,
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
  chatChannels: undefined,
  deleteChat: defaultFunction,
  setChatChannels: defaultFunction,
});

const PresenceContext = createContext<PresenceStoreData>({
  setPresence: defaultFunction,
  addPresences: defaultFunction,
  presences: new Array<PresenceEntity>(),
});

const TournamentContext = createContext<TournamentStoreData>({
  tournamentContext: defaultTournamentState,
  dispatchTournamentContext: defaultFunction,
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
  addHandleExaminer: defaultFunction,
  removeHandleExaminer: defaultFunction,
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
) => ({
  ...defaultGameState,
  clockController,
  onClockTick,
  onClockTimeout,
});

// Support for examining. Must be nested deeper than the Real Stuffs.

const doNothing = () => {}; // defaultFunction currently is the same as this.

// CSS selectors that should support Examine shortcuts.
const WHERE_TO_ENABLE_EXAMINE_SHORTCUTS = [
  '.analyzer-card',
  '.analyzer-container',
  '.play-area',
];

const ExaminableStore = ({ children }: { children: React.ReactNode }) => {
  const { useState } = useMountedState();

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
        userID,
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
      const { userID } = ret.players[i];
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

  // There are two handlers (the Tablet view has its own Analyzer button).
  // Fortunately the second one will do nothing, so we just trigger both.
  const [handleExaminers, setHandleExaminers] = useState(
    new Array<() => void>()
  );
  const addHandleExaminer = useCallback((x) => {
    setHandleExaminers((a: Array<() => void>) => {
      if (!a.includes(x)) {
        a = [...a, x];
      }
      return a;
    });
  }, []);
  const removeHandleExaminer = useCallback((x) => {
    setHandleExaminers((a) => {
      const b = a.filter((y) => y !== x);
      return a.length === b.length ? a : b;
    });
  }, []);

  const shouldTrigger = useCallback((where) => {
    try {
      return (
        where &&
        WHERE_TO_ENABLE_EXAMINE_SHORTCUTS.some((selector) =>
          where.closest(selector)
        )
      );
    } catch (e) {
      return false;
    }
  }, []);

  const handleExamineShortcuts = useCallback(
    (evt) => {
      if (
        isExamining &&
        (shouldTrigger(document.activeElement) ||
          shouldTrigger(window.getSelection()?.focusNode?.parentElement))
      ) {
        if (evt.ctrlKey || evt.altKey || evt.metaKey) {
          // If a modifier key is held, never mind.
        } else {
          if (evt.key === '<' || evt.key === 'Home') {
            evt.preventDefault();
            handleExamineFirst();
          }
          if (evt.key === ',' || evt.key === 'PageUp') {
            evt.preventDefault();
            handleExaminePrev();
          }
          if (evt.key === '.' || evt.key === 'PageDown') {
            evt.preventDefault();
            handleExamineNext();
          }
          if (evt.key === '>' || evt.key === 'End') {
            evt.preventDefault();
            handleExamineLast();
          }
          if (evt.key === '/' || evt.key === '?') {
            evt.preventDefault();
            for (const handleExaminer of handleExaminers) {
              handleExaminer();
            }
          }
          if (evt.key === 'Escape') {
            evt.preventDefault();
            handleExamineEnd();
          }
        }
      }
    },
    [
      isExamining,
      shouldTrigger,
      handleExamineFirst,
      handleExaminePrev,
      handleExamineNext,
      handleExamineLast,
      handleExamineEnd,
      handleExaminers,
    ]
  );

  React.useEffect(() => {
    if (isExamining) {
      document.addEventListener('keydown', handleExamineShortcuts);
      return () => {
        document.removeEventListener('keydown', handleExamineShortcuts);
      };
    }
  }, [isExamining, handleExamineShortcuts]);

  const examineStore = useMemo(
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
      addHandleExaminer,
      removeHandleExaminer,
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
      addHandleExaminer,
      removeHandleExaminer,
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
  ret = <ExamineContext.Provider value={examineStore} children={ret} />;

  // typescript did not like "return ret;"
  return <React.Fragment children={ret} />;
};

// The Real Store.

const RealStore = ({ children, ...props }: Props) => {
  const { useState } = useMountedState();

  const [lobbyContext, setLobbyContext] = useState<LobbyState>({
    soughtGames: [],
    activeGames: [],
    matchRequests: [],
  });
  const dispatchLobbyContext = useCallback(
    (action) => setLobbyContext((state) => LobbyReducer(state, action)),
    []
  );
  const [loginState, setLoginState] = useState({
    username: '',
    userID: '',
    loggedIn: false,
    connectedToSocket: false,
    connID: '',
    path: '',
    perms: new Array<string>(),
  });
  const dispatchLoginState = useCallback(
    (action) => setLoginState((state) => LoginStateReducer(state, action)),
    []
  );

  const [tournamentContext, setTournamentContext] = useState(
    defaultTournamentState
  );
  const dispatchTournamentContext = useCallback((action) => {
    setTournamentContext((state) => {
      const newState = TournamentReducer(state, action);
      return newState;
    });
  }, []);

  const [currentLagMs, setCurrentLagMs] = useState(NaN);

  const [placedTilesTempScore, setPlacedTilesTempScore] = useState<
    number | undefined
  >(undefined);
  const [placedTiles, setPlacedTiles] = useState(new Set<EphemeralTile>());
  const [displayedRack, setDisplayedRack] = useState('');

  const {
    clockController,
    onClockTick,
    onClockTimeout,
    stopClock,
    timerContext,
    pTimedOut,
    setPTimedOut,
  } = useTimer();

  const [gameContext, setGameContext] = useState<GameState>(() =>
    gameStateInitializer(clockController, onClockTick, onClockTimeout)
  );
  const dispatchGameContext = useCallback(
    (action) => setGameContext((state) => GameReducer(state, action)),
    []
  );

  const [gameMetaEventContext, setGameMetaEventContext] = useState<
    MetaEventState
  >({
    curEvt: MetaStates.NO_ACTIVE_REQUEST,
    // clockController: null,
  });

  const [poolFormat, setPoolFormat] = useState<PoolFormatType>(
    PoolFormatType.Alphabet
  );

  const [gameEndMessage, setGameEndMessage] = useState('');
  const [rematchRequest, setRematchRequest] = useState(new MatchRequest());
  const [chat, setChat] = useState(new Array<ChatEntityObj>());
  const [chatChannels, setChatChannels] = useState<
    ActiveChatChannels.AsObject | undefined
  >(undefined);
  const [excludedPlayers, setExcludedPlayers] = useState(new Set<string>());
  const [excludedPlayersFetched, setExcludedPlayersFetched] = useState(false);
  const [pendingBlockRefresh, setPendingBlockRefresh] = useState(false);
  const [presences, setPresences] = useState(new Array<PresenceEntity>());

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
        channel: 'server',
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

  const deleteChat = useCallback((id: string, channel: string) => {
    setChat((oldChat) => {
      const chatCopy = oldChat.filter(
        (c) => !(c.id === id && c.channel === channel)
      );
      return chatCopy;
    });
  }, []);

  const setPresence = useCallback((entity: PresenceEntity) => {
    setPresences((prevPresences) => {
      // filter out the current entity then add it if we're not deleting
      // (prevents duplicates)
      const presencesCopy = prevPresences.filter(
        (p) => !(p.channel === entity.channel && p.uuid === entity.uuid)
      );
      if (!entity.deleting) {
        return presencesCopy.concat(entity);
      }
      return presencesCopy;
    });
  }, []);

  const addPresences = useCallback(
    (entities: Array<PresenceEntity>) => {
      entities.forEach((p) => {
        setPresence(p);
      });
    },
    [setPresence]
  );

  const lobbyStore = useMemo(
    () => ({
      lobbyContext,
      dispatchLobbyContext,
    }),
    [lobbyContext, dispatchLobbyContext]
  );
  const loginStateStore = useMemo(
    () => ({
      loginState,
      dispatchLoginState,
    }),
    [loginState, dispatchLoginState]
  );
  const tournamentStateStore = useMemo(
    () => ({
      tournamentContext,
      dispatchTournamentContext,
    }),
    [tournamentContext, dispatchTournamentContext]
  );
  const lagStore = useMemo(
    () => ({
      currentLagMs,
      setCurrentLagMs,
    }),
    [currentLagMs, setCurrentLagMs]
  );
  const tentativePlayStore = useMemo(
    () => ({
      placedTilesTempScore,
      placedTiles,
      displayedRack,
      setPlacedTilesTempScore,
      setPlacedTiles,
      setDisplayedRack,
    }),
    [
      placedTilesTempScore,
      placedTiles,
      displayedRack,
      setPlacedTilesTempScore,
      setPlacedTiles,
      setDisplayedRack,
    ]
  );
  const excludedPlayersStore = useMemo(
    () => ({
      excludedPlayers,
      setExcludedPlayers,
      excludedPlayersFetched,
      setExcludedPlayersFetched,
      pendingBlockRefresh,
      setPendingBlockRefresh,
    }),
    [
      excludedPlayers,
      setExcludedPlayers,
      excludedPlayersFetched,
      setExcludedPlayersFetched,
      pendingBlockRefresh,
      setPendingBlockRefresh,
    ]
  );
  const challengeResultEventStore = useMemo(
    () => ({
      challengeResultEvent,
    }),
    [challengeResultEvent]
  );
  const gameContextStore = useMemo(
    () => ({
      gameContext,
      dispatchGameContext,
    }),
    [gameContext, dispatchGameContext]
  );

  const gameMetaEventContextStore = useMemo(
    () => ({
      gameMetaEventContext,
      setGameMetaEventContext,
    }),
    [gameMetaEventContext, setGameMetaEventContext]
  );

  const chatStore = useMemo(
    () => ({
      addChat,
      addChats,
      clearChat,
      deleteChat,
      chat,
      chatChannels,
      setChatChannels,
    }),
    [
      addChat,
      addChats,
      clearChat,
      chat,
      chatChannels,
      deleteChat,
      setChatChannels,
    ]
  );
  const presenceStore = useMemo(
    () => ({
      setPresence,
      addPresences,
      presences,
    }),
    [setPresence, addPresences, presences]
  );
  const gameEndMessageStore = useMemo(
    () => ({
      gameEndMessage,
      setGameEndMessage,
    }),
    [gameEndMessage, setGameEndMessage]
  );
  const rematchRequestStore = useMemo(
    () => ({
      rematchRequest,
      setRematchRequest,
    }),
    [rematchRequest, setRematchRequest]
  );
  const timerStore = useMemo(
    () => ({
      // initClockController,
      stopClock,
      timerContext,
      pTimedOut,
      setPTimedOut,
    }),
    [
      // initClockController,
      stopClock,
      timerContext,
      pTimedOut,
      setPTimedOut,
    ]
  );
  const poolFormatStore = useMemo(
    () => ({
      poolFormat,
      setPoolFormat,
    }),
    [poolFormat, setPoolFormat]
  );

  let ret = <ExaminableStore children={children} />;
  ret = <LobbyContext.Provider value={lobbyStore} children={ret} />;
  ret = <LoginStateContext.Provider value={loginStateStore} children={ret} />;
  ret = <LagContext.Provider value={lagStore} children={ret} />;
  ret = (
    <TournamentContext.Provider value={tournamentStateStore} children={ret} />
  );
  ret = (
    <TentativePlayContext.Provider value={tentativePlayStore} children={ret} />
  );
  ret = (
    <ExcludedPlayersContext.Provider
      value={excludedPlayersStore}
      children={ret}
    />
  );
  ret = (
    <ChallengeResultEventContext.Provider
      value={challengeResultEventStore}
      children={ret}
    />
  );
  ret = <GameContextContext.Provider value={gameContextStore} children={ret} />;
  ret = (
    <GameMetaEventContext.Provider
      value={gameMetaEventContextStore}
      children={ret}
    />
  );
  ret = <ChatContext.Provider value={chatStore} children={ret} />;
  ret = <PresenceContext.Provider value={presenceStore} children={ret} />;
  ret = (
    <GameEndMessageContext.Provider
      value={gameEndMessageStore}
      children={ret}
    />
  );
  ret = (
    <RematchRequestContext.Provider
      value={rematchRequestStore}
      children={ret}
    />
  );
  ret = <TimerContext.Provider value={timerStore} children={ret} />;
  ret = <PoolFormatContext.Provider value={poolFormatStore} children={ret} />;

  // typescript did not like "return ret;"
  return <React.Fragment children={ret} />;
};

const ResetStoreContext = createContext({ resetStore: defaultFunction });
export const useResetStoreContext = () => useContext(ResetStoreContext);

export const Store = ({ children }: { children: React.ReactNode }) => {
  const { useState } = useMountedState();

  // In JS the | 0 loops within int32 and avoids reaching Number.MAX_SAFE_INTEGER.
  const [storeId, setStoreId] = useState(0);
  const resetStore = useCallback(() => setStoreId((n) => (n + 1) | 0), []);

  // Reset on browser navigation.
  React.useEffect(() => {
    const handleBrowserNavigation = (evt: PopStateEvent) => {
      resetStore();
    };
    window.addEventListener('popstate', handleBrowserNavigation);
    return () => {
      window.removeEventListener('popstate', handleBrowserNavigation);
    };
  }, [resetStore]);

  return (
    <ResetStoreContext.Provider value={{ resetStore }}>
      <RealStore key={storeId} children={children} />
    </ResetStoreContext.Provider>
  );
};

export const useLobbyStoreContext = () => useContext(LobbyContext);
export const useLoginStateStoreContext = () => useContext(LoginStateContext);
export const useLagStoreContext = () => useContext(LagContext);
export const useTournamentStoreContext = () => useContext(TournamentContext);
export const useTentativeTileContext = () => useContext(TentativePlayContext);
export const useExcludedPlayersStoreContext = () =>
  useContext(ExcludedPlayersContext);
export const useChallengeResultEventStoreContext = () =>
  useContext(ChallengeResultEventContext);
export const useGameContextStoreContext = () => useContext(GameContextContext);
export const useGameMetaEventContext = () => useContext(GameMetaEventContext);
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
