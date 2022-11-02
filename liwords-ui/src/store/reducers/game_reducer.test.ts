import { GameReducer, startingGameState } from './game_reducer';
import { ActionType } from '../../actions/actions';

import {
  GameHistory,
  PlayerInfo,
  GameEvent,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { StandardEnglishAlphabet } from '../../constants/alphabets';
import {
  GameHistoryRefresher,
  ServerGameplayEvent,
} from '../../gen/api/proto/ipc/omgwords_pb';
import { GameEvent_Type } from '../../gen/macondo/api/proto/macondo/macondo_pb';

const historyRefresher = () => {
  return new GameHistoryRefresher({
    history: new GameHistory({
      players: [
        new PlayerInfo({ nickname: 'césar', userId: 'cesar123' }),
        new PlayerInfo({ nickname: 'mina', userId: 'mina123' }),
      ],
      lastKnownRacks: ['CDEIPTV', 'FIMRSUU'],
      uid: 'game42',
    }),
  });
};

it('tests refresher', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher(),
  });
  expect(newState.players[0].currentRack).toBe('CDEIPTV');
  expect(newState.players[0].userID).toBe('cesar123');
  expect(newState.players[1].currentRack).toBe('FIMRSUU');
  expect(newState.players[1].userID).toBe('mina123');
  expect(newState.onturn).toBe(0);
  expect(newState.turns.length).toBe(0);
});

it('tests addevent', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher(),
  });

  const evt = new GameEvent({
    playerIndex: 0,
    rack: 'CDEIPTV',
    cumulative: 26,
    row: 7,
    column: 3,
    position: '8D',
    playedTiles: 'DEPICT',
    score: 26,
  });

  const sge = new ServerGameplayEvent({
    newRack: 'EFIKNNV',
    event: evt,
    gameId: 'game42',
  });
  const newState2 = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });
  expect(newState2.players[0].currentRack).toBe('EFIKNNV');
  expect(newState2.players[0].userID).toBe('cesar123');
  expect(newState2.players[1].currentRack).toBe('FIMRSUU');
  expect(newState2.players[1].userID).toBe('mina123');
  expect(newState2.onturn).toBe(1);
  expect(newState2.turns.length).toBe(1);
});

it('tests addevent with different id', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher(),
  });

  const evt = new GameEvent({
    type: GameEvent_Type.TILE_PLACEMENT_MOVE,
    playerIndex: 0,
    rack: 'CDEIPTV',
    cumulative: 26,
    row: 7,
    column: 3,
    position: '8D',
    playedTiles: 'DEPICT',
    score: 26,
  });
  const sge = new ServerGameplayEvent({
    newRack: 'EFIKNNV',
    event: evt,
    gameId: 'anotherone',
  });

  const newState2 = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });
  // No change
  expect(newState2.players[0].currentRack).toBe('CDEIPTV');
  expect(newState2.players[0].userID).toBe('cesar123');
  expect(newState2.players[1].currentRack).toBe('FIMRSUU');
  expect(newState2.players[1].userID).toBe('mina123');
  expect(newState2.onturn).toBe(0);
  expect(newState2.turns.length).toBe(0);
});

const historyRefresher2 = () => {
  return new GameHistoryRefresher({
    history: new GameHistory({
      players: [
        new PlayerInfo({ nickname: 'césar', userId: 'cesar123' }),
        new PlayerInfo({ nickname: 'mina', userId: 'mina123' }),
      ],
      lastKnownRacks: ['EFMPRST', 'AEELRX?'],
      uid: 'game63',
    }),
  });
};

const historyRefresher3 = () => {
  return new GameHistoryRefresher({
    history: new GameHistory({
      players: [
        new PlayerInfo({ nickname: 'mina', userId: 'mina123' }),
        new PlayerInfo({ nickname: 'césar', userId: 'cesar123' }),
      ],
      lastKnownRacks: ['AEELRX?', 'EFMPRST'],
      uid: 'game63',
    }),
  });
};

const historyRefresher3AfterChallenge = () => {
  // {"history":{"turns":[{"events":[{"nickname":"mina","rack":"?AEELRX","cumulative":92,"row":7,"column":7,
  // "position": "8H", "played_tiles": "RELAXEs", "score": 92
  // }, { "nickname":"mina", "type":3, "cumulative":97, "bonus":5}]}], "players": [{ "nickname": "césar", "real_name": "césar" }, { "nickname": "mina", "real_name": "mina" }], "id_auth": "org.aerolith", "uid": "kqVFQ7PXG3Es3gn9jNX5p9", "description": "Created with Macondo", "last_known_racks": ["EFMPRST", "EEJNNOQ"]

  return new GameHistoryRefresher({
    history: new GameHistory({
      players: [
        new PlayerInfo({ nickname: 'mina', userId: 'mina123' }),
        new PlayerInfo({ nickname: 'césar', userId: 'cesar123' }),
      ],
      lastKnownRacks: ['EEJNNOQ', 'EFMPRST'],
      uid: 'game63',
      events: [
        new GameEvent({
          playerIndex: 0,
          rack: '?AEELRX',
          cumulative: 92,
          row: 7,
          column: 7,
          position: '8H',
          playedTiles: 'RELAXEs',
          score: 92,
        }),
        new GameEvent({
          playerIndex: 0,
          type: GameEvent_Type.CHALLENGE_BONUS,
          cumulative: 97,
          bonus: 5,
        }),
      ],
    }),
  });
};

it('tests challenge with refresher event afterwards', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher3(),
  });

  const sge = new ServerGameplayEvent({
    newRack: 'EEJNNOQ',
    gameId: 'game63',
    event: new GameEvent({
      playerIndex: 0,
      rack: '?AEELRX',
      cumulative: 92,
      row: 7,
      column: 7,
      position: '8H',
      playedTiles: 'RELAXEs',
      score: 92,
    }),
  });

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });
  expect(newState.players[0].currentRack).toBe('EEJNNOQ');
  expect(newState.players[0].userID).toBe('mina123');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('cesar123');
  expect(newState.onturn).toBe(1);
  expect(newState.turns.length).toBe(1);
  // Now césar challenges RELAXEs (who knows why, it looks phony)
  newState = GameReducer(newState, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher3AfterChallenge(),
  });
  expect(newState.players[0].currentRack).toBe('EEJNNOQ');
  expect(newState.players[0].userID).toBe('mina123');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('cesar123');
  expect(newState.players[0].score).toBe(97);
  expect(newState.players[1].score).toBe(0);
  // It is still César's turn
  expect(newState.onturn).toBe(1);
  expect(newState.turns.length).toBe(2);
});

it('tests challenge with challenge event afterwards', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher3(),
  });

  const sge = new ServerGameplayEvent({
    newRack: 'EEJNNOQ',
    event: new GameEvent({
      playerIndex: 0,
      rack: '?AEELRX',
      cumulative: 92,
      row: 7,
      column: 7,
      position: '8H',
      playedTiles: 'RELAXEs',
      score: 92,
    }),
    gameId: 'game63',
  });

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });

  // Now add a challenge event.
  const sge2 = new ServerGameplayEvent({
    event: new GameEvent({
      playerIndex: 0,
      type: GameEvent_Type.CHALLENGE_BONUS,
      cumulative: 97,
      bonus: 5,
    }),
    gameId: 'game63',
  });

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge2,
  });
  expect(newState.players[0].currentRack).toBe('EEJNNOQ');
  expect(newState.players[0].userID).toBe('mina123');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('cesar123');
  expect(newState.players[0].score).toBe(97);
  expect(newState.players[1].score).toBe(0);
  // It is still César's turn
  expect(newState.onturn).toBe(1);
  expect(newState.turns.length).toBe(2);
});

const historyRefresherWithPlay = () => {
  return new GameHistoryRefresher({
    history: new GameHistory({
      players: [
        new PlayerInfo({ nickname: 'césar', userId: 'cesar123' }),
        new PlayerInfo({ nickname: 'mina', userId: 'mina123' }),
      ],
      lastKnownRacks: ['', 'DEIMNRU'],
      events: [
        new GameEvent({
          column: 6,
          row: 7,
          score: 12,
          position: '8G',
          playedTiles: 'WIT',
          playerIndex: 0,
          cumulative: 12,
        }),
      ],
      uid: 'game42',
    }),
  });
};

it('tests deduplication of event', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresherWithPlay(),
  });

  const sge = new ServerGameplayEvent({
    event: new GameEvent({
      cumulative: 12,
      row: 7,
      column: 6,
      position: '8G',
      playedTiles: 'WIT',
      score: 12,
      playerIndex: 0,
    }),
    newRack: '',
    gameId: 'game42',
  });

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });
  expect(newState.pool['W']).toBe(1);
  expect(newState.pool['I']).toBe(8);
  expect(newState.pool['T']).toBe(5);
  expect(newState.onturn).toBe(1);
});
