import React, {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
} from 'react';
import { useMountedState } from '../utils/mounted';

import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
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
import { MetaEventState, MetaStates } from './meta_game_events';
import { StandardEnglishAlphabet } from '../constants/alphabets';
import {
  LiwordsSocket,
  LiwordsSocketContext,
  LiwordsSocketValues,
  OnSocketMsgType,
  useLiwordsSocketContext,
} from '../socket/socket';
import { useOnSocketMsg } from './socket_handlers';
import { SeekRequest } from '../gen/api/proto/ipc/omgseeks_pb';
import { ServerChallengeResultEvent } from '../gen/api/proto/ipc/omgwords_pb';

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

const defaultTimerContext = {
  p0: 0,
  p1: 0,
  activePlayer: 'p0' as PlayerOrder,
  lastUpdate: 0,
};

const defaultFunction = () => {};

// Functions and data to deal with the global store.

type ContextMatchStoreData = {
  handleContextMatches: Array<(s: string) => void>;
  addHandleContextMatch: (x: (s: string) => void) => void;
  removeHandleContextMatch: (x: (s: string) => void) => void;
};

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

export type FriendUser = {
  username: string;
  uuid: string;
  channel: string[];
};

type FriendsStoreData = {
  friends: { [uuid: string]: FriendUser };
  setFriends: React.Dispatch<
    React.SetStateAction<{ [uuid: string]: FriendUser }>
  >;
  pendingFriendsRefresh: boolean;
  setPendingFriendsRefresh: React.Dispatch<React.SetStateAction<boolean>>;
};

type ModeratorsStoreData = {
  moderators: Set<string>;
  setModerators: React.Dispatch<React.SetStateAction<Set<string>>>;
  admins: Set<string>;
  setAdmins: React.Dispatch<React.SetStateAction<Set<string>>>;
  modsFetched: boolean;
  setModsFetched: React.Dispatch<React.SetStateAction<boolean>>;
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
  rematchRequest: SeekRequest;
  setRematchRequest: React.Dispatch<React.SetStateAction<SeekRequest>>;
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
  blindfoldCommand: string;
  blindfoldUseNPA: boolean;
  setPlacedTilesTempScore: (s: number | undefined) => void;
  setPlacedTiles: (t: Set<EphemeralTile>) => void;
  setDisplayedRack: (l: string) => void;
  setBlindfoldCommand: (l: string) => void;
  setBlindfoldUseNPA: (l: boolean) => void;
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
  doneButtonRef: React.MutableRefObject<HTMLElement | null>;
};

const defaultGameState = startingGameState(StandardEnglishAlphabet, [], '');

// This is annoying, but we have to add a default for everything in this
// declaration. Declaring it as a Partial<StoreData> breaks things elsewhere.
// For context, these used to be a single StoreData that contained everything.

const ContextMatchContext = createContext<ContextMatchStoreData>({
  handleContextMatches: [],
  addHandleContextMatch: defaultFunction,
  removeHandleContextMatch: defaultFunction,
});

const LobbyContext = createContext<LobbyStoreData>({
  lobbyContext: {
    soughtGames: [],
    activeGames: [],
    matchRequests: [],
    profile: { ratings: {} },
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
  blindfoldCommand: '',
  blindfoldUseNPA: false,
  setPlacedTilesTempScore: defaultFunction,
  setPlacedTiles: defaultFunction,
  setDisplayedRack: defaultFunction,
  setBlindfoldCommand: defaultFunction,
  setBlindfoldUseNPA: defaultFunction,
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

const FriendsContext = createContext<FriendsStoreData>({
  friends: {},
  setFriends: defaultFunction,
  pendingFriendsRefresh: false,
  setPendingFriendsRefresh: defaultFunction,
});

const ModeratorsContext = createContext<ModeratorsStoreData>({
  // used for displaying mod status to other users, should not be trusted for actually granting powers
  moderators: new Set<string>(),
  setModerators: defaultFunction,
  admins: new Set<string>(),
  setAdmins: defaultFunction,
  modsFetched: false,
  setModsFetched: defaultFunction,
});

const ChallengeResultEventContext = createContext<
  ChallengeResultEventStoreData
>({
  challengeResultEvent: defaultFunction,
});

const GameMetaEventContext = createContext<GameMetaEventStoreData>({
  gameMetaEventContext: {
    curEvt: MetaStates.NO_ACTIVE_REQUEST,
    initialExpiry: 0,
    evtId: '',
    evtCreator: '',
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
  rematchRequest: new SeekRequest(),
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
  doneButtonRef: { current: null },
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

  const { gameContext } = gameContextStore;
  const numberOfTurns = gameContext.turns.length;
  const [isExamining, setIsExamining] = useState(false);
  const doneButtonRef = useRef<HTMLElement | null>(null);
  const [examinedTurn, setExaminedTurnRaw] = useState(Infinity);
  const setExaminedTurn = useCallback(
    (x: ((x: number) => number) | number) => {
      // Check if shortcuts were working when setting the examined turn.
      if (doneButtonRef.current && shouldTrigger(document.activeElement)) {
        // If so, they should remain working after.
        setTimeout(() => {
          // Examining the last turn disables ">".
          // In Chrome the body becomes the activeElement.
          // In Firefox it is necessary to just check .disabled.
          const dae = document.activeElement;
          if (
            doneButtonRef.current &&
            ((dae as any)?.disabled || !shouldTrigger(dae))
          ) {
            // Focusing on the Done button reenables first/prev shortcuts.
            doneButtonRef.current.focus();
          }
        }, 0);
      }
      setExaminedTurnRaw(x);
    },
    [shouldTrigger]
  );
  const handleExamineStartUnconditionally = useCallback(() => {
    setIsExamining(true);
  }, []);
  const handleExamineEnd = useCallback(() => {
    setIsExamining(false);
  }, []);
  const handleExamineFirst = useCallback(() => {
    setExaminedTurn(0);
  }, [setExaminedTurn]);
  const handleExaminePrev = useCallback(() => {
    setExaminedTurn((x) => Math.max(Math.min(x, numberOfTurns) - 1, 0));
  }, [setExaminedTurn, numberOfTurns]);
  const handleExamineNext = useCallback(() => {
    setExaminedTurn((x) => (x >= numberOfTurns - 1 ? Infinity : x + 1));
  }, [setExaminedTurn, numberOfTurns]);
  const handleExamineLast = useCallback(() => {
    setExaminedTurn(Infinity);
  }, [setExaminedTurn]);
  const handleExamineGoTo = useCallback(
    (x) => {
      if (x >= numberOfTurns) {
        setExaminedTurn(Infinity);
      } else {
        setExaminedTurn(Math.max(Math.min(x, numberOfTurns), 0));
      }
    },
    [setExaminedTurn, numberOfTurns]
  );

  const examinableGameContext = useMemo(() => {
    if (!isExamining) return gameContext;
    const ret = startingGameState(
      gameContext.alphabet,
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

      // Time comes from the most recent past.
      // But may belong to either player, depending on event type.
      let time = Infinity; // No gameInfo here, patch in PlayerCard.
      for (let j = replayedTurns.length; --j >= 0; ) {
        const turn = gameContext.turns[j];
        if (
          turn.getType() === GameEvent.Type.END_RACK_PTS ||
          turn.getType() === GameEvent.Type.END_RACK_PENALTY
        ) {
          continue;
        }

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
        if (
          turn.getType() === GameEvent.Type.END_RACK_PTS ||
          turn.getType() === GameEvent.Type.END_RACK_PENALTY
        ) {
          continue;
        }

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
  const handleExamineStart =
    examinableGameContext.players.length > 0
      ? handleExamineStartUnconditionally
      : handleExamineEnd;

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
  // They are functionally the same.
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

  const handleExamineShortcuts = useCallback(
    (evt) => {
      if (isExamining && shouldTrigger(document.activeElement)) {
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
              break; // They are functionally the same, trigger either one.
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
      doneButtonRef,
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
      doneButtonRef,
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

// The Real LoginState Store.

const RealLoginStateStore = ({ children, ...props }: Props) => {
  const { useState } = useMountedState();

  const [loginState, setLoginState] = useState({
    username: '',
    userID: '',
    loggedIn: false,
    connectedToSocket: false,
    connID: '',
    perms: new Array<string>(),
  });
  const dispatchLoginState = useCallback(
    (action) => setLoginState((state) => LoginStateReducer(state, action)),
    []
  );

  const loginStateStore = useMemo(
    () => ({
      loginState,
      dispatchLoginState,
    }),
    [loginState, dispatchLoginState]
  );

  return (
    <LoginStateContext.Provider value={loginStateStore} children={children} />
  );
};

// The Real LiwordsSocket Store.

const RealLiwordsSocketStore = ({
  resetLiwordsSocketStore,
  children,
  ...props
}: Props & {
  resetLiwordsSocketStore: () => void;
}) => {
  const { useState } = useMountedState();

  const [onSocketMsg, setOnSocketMsg] = useState<OnSocketMsgType>(
    () => defaultFunction
  );

  const [liwordsSocketValues, setLiwordsSocketValues] = useState<
    LiwordsSocketValues
  >({
    sendMessage: defaultFunction,
    justDisconnected: false,
  });

  const liwordsSocketStore = useMemo(
    () => ({
      liwordsSocketValues,
      onSocketMsg,
      resetLiwordsSocketStore,
      setLiwordsSocketValues,
      setOnSocketMsg,
    }),
    [
      liwordsSocketValues,
      onSocketMsg,
      resetLiwordsSocketStore,
      setLiwordsSocketValues,
      setOnSocketMsg,
    ]
  );

  return (
    <LiwordsSocketContext.Provider
      value={liwordsSocketStore}
      children={children}
    />
  );
};

// The Real Rest Of Store.

const RealRestOfStore = ({ children, ...props }: Props) => {
  const { useState } = useMountedState();

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

  const [lobbyContext, setLobbyContext] = useState<LobbyState>({
    soughtGames: [],
    activeGames: [],
    matchRequests: [],
    profile: { ratings: {} },
  });
  const dispatchLobbyContext = useCallback(
    (action) => setLobbyContext((state) => LobbyReducer(state, action)),
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
  const [blindfoldCommand, setBlindfoldCommand] = useState('');
  const [blindfoldUseNPA, setBlindfoldUseNPA] = useState(false);

  const [gameContext, setGameContext] = useState<GameState>(() =>
    gameStateInitializer(clockController, onClockTick, onClockTimeout)
  );
  const dispatchGameContext = useCallback(
    (action) => setGameContext((state) => GameReducer(state, action)),
    []
  );

  const [timerContext, setTimerContext] = useState<Times>(defaultTimerContext);
  const [pTimedOut, setPTimedOut] = useState<PlayerOrder | undefined>(
    undefined
  );

  const [gameMetaEventContext, setGameMetaEventContext] = useState<
    MetaEventState
  >({
    curEvt: MetaStates.NO_ACTIVE_REQUEST,
    initialExpiry: 0,
    evtId: '',
    evtCreator: '',
    // clockController: null,
  });

  const [poolFormat, setPoolFormat] = useState<PoolFormatType>(
    PoolFormatType.Alphabet
  );

  const [gameEndMessage, setGameEndMessage] = useState('');
  const [rematchRequest, setRematchRequest] = useState(new SeekRequest());
  const [chat, setChat] = useState(new Array<ChatEntityObj>());
  const [chatChannels, setChatChannels] = useState<
    ActiveChatChannels.AsObject | undefined
  >(undefined);
  const [excludedPlayers, setExcludedPlayers] = useState(new Set<string>());
  const [friends, setFriends] = useState({});
  const [pendingFriendsRefresh, setPendingFriendsRefresh] = useState(false);
  const [excludedPlayersFetched, setExcludedPlayersFetched] = useState(false);
  const [pendingBlockRefresh, setPendingBlockRefresh] = useState(false);
  const [moderators, setModerators] = useState(new Set<string>());
  const [admins, setAdmins] = useState(new Set<string>());
  const [modsFetched, setModsFetched] = useState(false);
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

  const stopClock = useCallback(() => {
    if (!clockController.current) {
      return;
    }
    clockController.current.stopClock();
    setTimerContext({ ...clockController.current.times });
  }, []);

  const [handleContextMatches, setHandleContextMatches] = useState(
    new Array<(s: string) => void>()
  );
  const addHandleContextMatch = useCallback((x) => {
    setHandleContextMatches((a: Array<(s: string) => void>) => {
      if (!a.includes(x)) {
        a = [...a, x];
      }
      return a;
    });
  }, []);
  const removeHandleContextMatch = useCallback((x) => {
    setHandleContextMatches((a) => {
      const b = a.filter((y) => y !== x);
      return a.length === b.length ? a : b;
    });
  }, []);

  const contextMatchStore = useMemo(
    () => ({
      handleContextMatches,
      addHandleContextMatch,
      removeHandleContextMatch,
    }),
    [handleContextMatches, addHandleContextMatch, removeHandleContextMatch]
  );
  const lobbyStore = useMemo(
    () => ({
      lobbyContext,
      dispatchLobbyContext,
    }),
    [lobbyContext, dispatchLobbyContext]
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
      blindfoldCommand,
      blindfoldUseNPA,
      setPlacedTilesTempScore,
      setPlacedTiles,
      setDisplayedRack,
      setBlindfoldCommand,
      setBlindfoldUseNPA,
    }),
    [
      placedTilesTempScore,
      placedTiles,
      displayedRack,
      blindfoldCommand,
      blindfoldUseNPA,
      setPlacedTilesTempScore,
      setPlacedTiles,
      setDisplayedRack,
      setBlindfoldCommand,
      setBlindfoldUseNPA,
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
  const friendsStore = useMemo(
    () => ({
      friends,
      setFriends,
      pendingFriendsRefresh,
      setPendingFriendsRefresh,
    }),
    [friends, setFriends, pendingFriendsRefresh, setPendingFriendsRefresh]
  );
  const moderatorsStore = useMemo(
    () => ({
      moderators,
      setModerators,
      admins,
      setAdmins,
      modsFetched,
      setModsFetched,
    }),
    [moderators, setModerators, admins, setAdmins, modsFetched, setModsFetched]
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
  ret = (
    <ContextMatchContext.Provider value={contextMatchStore} children={ret} />
  );
  ret = <LobbyContext.Provider value={lobbyStore} children={ret} />;
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
  ret = <FriendsContext.Provider value={friendsStore} children={ret} />;
  ret = <ModeratorsContext.Provider value={moderatorsStore} children={ret} />;
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

// This needs to be nested inside the Rest Of Store.

const InstallOnSocketMsg = ({ children }: { children: React.ReactNode }) => {
  const { onSocketMsg, setOnSocketMsg } = useLiwordsSocketContext();

  const newOnSocketMsg = useOnSocketMsg();

  const oldOnSocketMsgRef = useRef(onSocketMsg);
  oldOnSocketMsgRef.current = onSocketMsg;

  React.useEffect(() => {
    const old = oldOnSocketMsgRef.current;
    setOnSocketMsg(() => newOnSocketMsg);
    return () => {
      setOnSocketMsg(() => old);
    };
  }, [newOnSocketMsg, setOnSocketMsg]);

  return <React.Fragment children={children} />;
};

const ResetStoreContext = createContext({
  resetLoginStateStore: defaultFunction,
  resetRestOfStore: defaultFunction,
});
export const useResetStoreContext = () => useContext(ResetStoreContext);

// Now includes the Socket.

export const Store = ({ children }: { children: React.ReactNode }) => {
  const { useState } = useMountedState();

  // In JS the | 0 loops within int32 and avoids reaching Number.MAX_SAFE_INTEGER.
  const [loginStateStoreId, setLoginStateStoreId] = useState(0);
  const resetLoginStateStore = useCallback(
    () => setLoginStateStoreId((n) => (n + 1) | 0),
    []
  );
  const [liwordsSocketStoreId, setLiwordsSocketStoreId] = useState(0);
  const resetLiwordsSocketStore = useCallback(
    () => setLiwordsSocketStoreId((n) => (n + 1) | 0),
    []
  );
  const [restOfStoreId, setRestOfStoreId] = useState(0);
  const resetRestOfStore = useCallback(
    () => setRestOfStoreId((n) => (n + 1) | 0),
    []
  );

  // Reset on browser navigation.
  React.useEffect(() => {
    const handleBrowserNavigation = (evt: PopStateEvent) => {
      resetRestOfStore();
    };
    window.addEventListener('popstate', handleBrowserNavigation);
    return () => {
      window.removeEventListener('popstate', handleBrowserNavigation);
    };
  }, [resetRestOfStore]);

  const resetStore = useMemo(
    () => ({
      resetLoginStateStore,
      resetRestOfStore,
    }),
    [resetLoginStateStore, resetRestOfStore]
  );

  return (
    <ResetStoreContext.Provider value={resetStore}>
      <RealLoginStateStore key={loginStateStoreId}>
        <RealLiwordsSocketStore
          key={liwordsSocketStoreId}
          resetLiwordsSocketStore={resetLiwordsSocketStore}
        >
          <LiwordsSocket />
          <RealRestOfStore key={restOfStoreId}>
            <InstallOnSocketMsg>{children}</InstallOnSocketMsg>
          </RealRestOfStore>
        </RealLiwordsSocketStore>
      </RealLoginStateStore>
    </ResetStoreContext.Provider>
  );
};

export const useContextMatchContext = () => useContext(ContextMatchContext);
export const useLobbyStoreContext = () => useContext(LobbyContext);
export const useLoginStateStoreContext = () => useContext(LoginStateContext);
export const useLagStoreContext = () => useContext(LagContext);
export const useTournamentStoreContext = () => useContext(TournamentContext);
export const useTentativeTileContext = () => useContext(TentativePlayContext);
export const useExcludedPlayersStoreContext = () =>
  useContext(ExcludedPlayersContext);
export const useFriendsStoreContext = () => useContext(FriendsContext);
export const useModeratorStoreContext = () => useContext(ModeratorsContext);
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
