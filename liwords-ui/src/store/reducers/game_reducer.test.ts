import { GameReducer, startingGameState } from './game_reducer';
import { EnglishCrosswordGameDistribution } from '../../constants/tile_distributions';
import { ActionType } from '../../actions/actions';
import {
  GameHistoryRefresher,
  ServerGameplayEvent,
} from '../../gen/api/proto/realtime/realtime_pb';
import {
  GameHistory,
  PlayerInfo,
  GameEvent,
  GameTurn,
} from '../../gen/macondo/api/proto/macondo/macondo_pb';

const historyRefresher = () => {
  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setUserId('césar');
  player2.setUserId('mina');
  his.setPlayersList([player1, player2]);
  his.setLastKnownRacksList(['CDEIPTV', 'FIMRSUU']);
  his.setUid('game42');
  ghr.setHistory(his);

  return ghr;
};

it('tests refresher', () => {
  const state = startingGameState(EnglishCrosswordGameDistribution, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher(),
  });
  expect(newState.players[0].currentRack).toBe('CDEIPTV');
  expect(newState.players[0].userID).toBe('césar');
  expect(newState.players[1].currentRack).toBe('FIMRSUU');
  expect(newState.players[1].userID).toBe('mina');
  expect(newState.onturn).toBe(0);
  expect(newState.turns.length).toBe(0);
});

it('tests addevent', () => {
  const state = startingGameState(EnglishCrosswordGameDistribution, [], '');
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
  expect(newState2.players[0].userID).toBe('césar');
  expect(newState2.players[1].currentRack).toBe('FIMRSUU');
  expect(newState2.players[1].userID).toBe('mina');
  expect(newState2.onturn).toBe(1);
  // The reducer doesn't add the turn yet, until another turn comes in with
  // a different name.
  expect(newState2.turns.length).toBe(0);
});

it('tests addevent with different id', () => {
  const state = startingGameState(EnglishCrosswordGameDistribution, [], '');
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
  expect(newState2.players[0].userID).toBe('césar');
  expect(newState2.players[1].currentRack).toBe('FIMRSUU');
  expect(newState2.players[1].userID).toBe('mina');
  expect(newState2.onturn).toBe(0);
  expect(newState2.turns.length).toBe(0);
});

const historyRefresher2 = () => {
  const ghr = new GameHistoryRefresher();
  const his = new GameHistory();
  const player1 = new PlayerInfo();
  const player2 = new PlayerInfo();
  player1.setUserId('césar');
  player2.setUserId('mina');
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
  player1.setUserId('césar');
  player1.setNickname('césar');
  player2.setUserId('mina');
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

  const turn = new GameTurn();
  turn.setEventsList([evt1, evt2]);
  his.addTurns(turn);

  ghr.setHistory(his);
  return ghr;
};

/*

{"players":[{"nickname":"césar","real_name":"césar"},{"nickname":"mina","real_name":"mina"}],"id_auth":"org.aerolith","uid":"kqVFQ7PXG3Es3gn9jNX5p9","description":"Created with
Macondo","last_known_racks":["EFMPRST","AEELRX?"],"flip_players":true,"challenge_rule":3}

*/

it('tests flip players', () => {
  const state = startingGameState(EnglishCrosswordGameDistribution, [], '');
  const newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher2(),
  });
  expect(newState.players[0].currentRack).toBe('AEELRX?');
  expect(newState.players[0].userID).toBe('mina');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('césar');
  expect(newState.onturn).toBe(0);
  expect(newState.turns.length).toBe(0);
});

it('tests challenge', () => {
  const state = startingGameState(EnglishCrosswordGameDistribution, [], '');
  let newState = GameReducer(state, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher2(),
  });
  /*

 "Event":{"event":{"nickname":"mina","rack":"?AEELRX","cumulative":92,"row":7,"column":7,"position":"8H","played_tiles":"RELAXEs","score":92},"game_id":"kqVFQ7PXG3Es3gn9jNX5p9","new_rack":"EEJNN
OQ","time_remaining":472033}}
*/

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
  expect(newState.players[0].userID).toBe('mina');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('césar');
  expect(newState.onturn).toBe(1);
  expect(newState.turns.length).toBe(0);

  // Now césar challenges RELAXEs (who knows why, it looks phony)
  newState = GameReducer(newState, {
    actionType: ActionType.RefreshHistory,
    payload: historyRefresher2AfterChallenge(),
  });
  expect(newState.players[0].currentRack).toBe('EEJNNOQ');
  expect(newState.players[0].userID).toBe('mina');
  expect(newState.players[1].currentRack).toBe('EFMPRST');
  expect(newState.players[1].userID).toBe('césar');
  expect(newState.players[0].score).toBe(97);
  expect(newState.players[1].score).toBe(0);
  // It is still César's turn
  expect(newState.onturn).toBe(1);
  expect(newState.turns.length).toBe(1);
});
