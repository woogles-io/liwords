import { GameEvent } from '../../gen/macondo/api/proto/macondo/macondo_pb';
import { gameEventsToTurns } from './turns';

it('test turns simple', () => {
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

  const turns = gameEventsToTurns([evt1, evt2]);
  expect(turns).toStrictEqual([[evt1, evt2]]);
});

it('test turns simple 2', () => {
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

  const evt3 = new GameEvent();
  evt3.setNickname('césar');
  evt3.setRack('ABCDEFG');
  evt3.setCumulative(38);
  evt3.setRow(6);
  evt3.setColumn(12);
  evt3.setPosition('M7');
  evt3.setPlayedTiles('F.EDBAG');
  evt3.setScore(38);

  const turns = gameEventsToTurns([evt1, evt2, evt3]);
  expect(turns.length).toBe(2);
  expect(turns).toStrictEqual([[evt1, evt2], [evt3]]);
});

it('test turns simple 3', () => {
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

  const evt3 = new GameEvent();
  evt3.setNickname('césar');
  evt3.setRack('ABCDEFG');
  evt3.setCumulative(40);
  evt3.setRow(6);
  evt3.setColumn(12);
  evt3.setPosition('M7');
  evt3.setPlayedTiles('F.EDBAC');
  evt3.setScore(40);

  const evt4 = new GameEvent();
  evt4.setNickname('césar');
  evt4.setType(GameEvent.Type.PHONY_TILES_RETURNED);
  evt4.setCumulative(0);

  const turns = gameEventsToTurns([evt1, evt2, evt3, evt4]);
  expect(turns.length).toBe(2);
  expect(turns).toStrictEqual([
    [evt1, evt2],
    [evt3, evt4],
  ]);
});
