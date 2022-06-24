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

const historyRefresher = () => {
  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setNickname('césar');
  player2.setNickname('mina');
  player1.setUserId('cesar123');
  player2.setUserId('mina123');
  his.setPlayersList([player1, player2]);
  his.setLastKnownRacksList(['CDEIPTV', 'FIMRSUU']);
  his.setUid('game42');
  ghr.setHistory(his);

  return ghr;
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

  const sge = new ServerGameplayEvent();
  const evt = new GameEvent();
  evt.setNickname('césar');
  evt.setRack('CDEIPTV');
  evt.setCumulative(26);
  evt.setRow(7);
  evt.setColumn(3);
  evt.setPosition('8D');
  evt.setPlayedTiles('DEPICT');
  evt.setScore(26);
  sge.setNewRack('EFIKNNV');
  sge.setEvent(evt);
  sge.setGameId('game42');

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

  const sge = new ServerGameplayEvent();
  const evt = new GameEvent();
  evt.setType(GameEvent.Type.TILE_PLACEMENT_MOVE);
  evt.setNickname('césar');
  evt.setRack('CDEIPTV');
  evt.setCumulative(26);
  evt.setRow(7);
  evt.setColumn(3);
  evt.setPosition('8D');
  evt.setPlayedTiles('DEPICT');
  evt.setScore(26);
  sge.setNewRack('EFIKNNV');
  sge.setEvent(evt);
  sge.setGameId('anotherone'); // This ID is not the same as the historyRefresher's

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
  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setUserId('cesar123');
  player1.setNickname('césar');
  player2.setUserId('mina123');
  player2.setNickname('mina');
  his.setPlayersList([player1, player2]);
  his.setLastKnownRacksList(['EFMPRST', 'AEELRX?']);
  his.setUid('game63');
  his.setSecondWentFirst(true);
  ghr.setHistory(his);
  return ghr;
};

const historyRefresher2AfterChallenge = () => {
  // {"history":{"turns":[{"events":[{"nickname":"mina","rack":"?AEELRX","cumulative":92,"row":7,"column":7,
  // "position": "8H", "played_tiles": "RELAXEs", "score": 92
  // }, { "nickname":"mina", "type":3, "cumulative":97, "bonus":5}]}], "players": [{ "nickname": "césar", "real_name": "césar" }, { "nickname": "mina", "real_name": "mina" }], "id_auth": "org.aerolith", "uid": "kqVFQ7PXG3Es3gn9jNX5p9", "description": "Created with Macondo", "last_known_racks": ["EFMPRST", "EEJNNOQ"]

  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setUserId('cesar123');
  player1.setNickname('césar');
  player2.setUserId('mina123');
  player2.setNickname('mina');
  his.setPlayersList([player1, player2]);
  his.setLastKnownRacksList(['EFMPRST', 'EEJNNOQ']);
  his.setUid('game63');
  his.setSecondWentFirst(true);

  const evt1 = new GameEvent();
  evt1.setNickname('mina');
  evt1.setRack('?AEELRX');
  evt1.setCumulative(92);
  evt1.setRow(7);
  evt1.setColumn(7);
  evt1.setPosition('8H');
  evt1.setPlayedTiles('RELAXEs');
  evt1.setScore(92);

  const evt2 = new GameEvent();
  evt2.setNickname('mina');
  evt2.setType(GameEvent.Type.CHALLENGE_BONUS);
  evt2.setCumulative(97);
  evt2.setBonus(5);

  his.addEvents(evt1);
  his.addEvents(evt2);

  ghr.setHistory(his);
  return ghr;
};

/*

{"players":[{"nickname":"césar","real_name":"césar"},{"nickname":"mina","real_name":"mina"}],"id_auth":"org.aerolith","uid":"kqVFQ7PXG3Es3gn9jNX5p9","description":"Created with
Macondo","last_known_racks":["EFMPRST","AEELRX?"],"flip_players":true,"challenge_rule":3}

*/

it('tests flip players', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher2(),
  });
  expect(newState.players[0].currentRack).toBe('AEELRX?');
  expect(newState.players[0].userID).toBe('mina123');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('cesar123');
  expect(newState.onturn).toBe(0);
  expect(newState.turns.length).toBe(0);
});

it('tests challenge with refresher event afterwards', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher2(),
  });

  const sge = new ServerGameplayEvent();
  const evt = new GameEvent();
  evt.setNickname('mina');
  evt.setRack('?AEELRX');
  evt.setCumulative(92);
  evt.setRow(7);
  evt.setColumn(7);
  evt.setPosition('8H');
  evt.setPlayedTiles('RELAXEs');
  evt.setScore(92);
  sge.setNewRack('EEJNNOQ');
  sge.setEvent(evt);
  sge.setGameId('game63');

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
    payload: historyRefresher2AfterChallenge(),
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
    payload: historyRefresher2(),
  });

  const sge = new ServerGameplayEvent();
  const evt = new GameEvent();
  evt.setNickname('mina');
  evt.setRack('?AEELRX');
  evt.setCumulative(92);
  evt.setRow(7);
  evt.setColumn(7);
  evt.setPosition('8H');
  evt.setPlayedTiles('RELAXEs');
  evt.setScore(92);
  sge.setNewRack('EEJNNOQ');
  sge.setEvent(evt);
  sge.setGameId('game63');

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });

  // Now add a challenge event.
  const sge2 = new ServerGameplayEvent();
  const evt2 = new GameEvent();
  evt2.setNickname('mina');
  evt2.setType(GameEvent.Type.CHALLENGE_BONUS);
  evt2.setCumulative(97);
  evt2.setBonus(5);
  sge2.setEvent(evt2);
  sge2.setGameId('game63');

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
  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setNickname('césar');
  player2.setNickname('mina');
  player1.setUserId('cesar123');
  player2.setUserId('mina123');
  his.setPlayersList([player1, player2]);
  his.setLastKnownRacksList(['', 'DEIMNRU']);
  const gevent = new GameEvent();
  gevent.setColumn(6);
  gevent.setRow(7);
  gevent.setScore(12);
  gevent.setPosition('8G');
  gevent.setPlayedTiles('WIT');
  gevent.setNickname('césar');
  gevent.setCumulative(12);
  his.setEventsList([gevent]);
  his.setUid('game42');
  ghr.setHistory(his);

  return ghr;
};

it('tests deduplication of event', () => {
  const state = startingGameState(StandardEnglishAlphabet, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresherWithPlay(),
  });

  const sge = new ServerGameplayEvent();
  const evt = new GameEvent();
  evt.setNickname('césar');
  evt.setCumulative(12);
  evt.setRow(7);
  evt.setColumn(6);
  evt.setPosition('8G');
  evt.setPlayedTiles('WIT');
  evt.setScore(12);
  sge.setNewRack('');
  sge.setEvent(evt);
  sge.setGameId('game42');

  newState = GameReducer(newState, {
    actionType: ActionType.AddGameEvent,
    payload: sge,
  });
  expect(newState.pool['W']).toBe(1);
  expect(newState.pool['I']).toBe(8);
  expect(newState.pool['T']).toBe(5);
  expect(newState.onturn).toBe(1);
});
