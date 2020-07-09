import { millisToTimeStr } from './timer_controller';

it('tests millis to time', () => {
  expect(millisToTimeStr(479900)).toEqual('08:00');
  expect(millisToTimeStr(479000)).toEqual('07:59');
  expect(millisToTimeStr(60000)).toEqual('01:00');
  expect(millisToTimeStr(59580)).toEqual('01:00');
  expect(millisToTimeStr(8900)).toEqual('00:08.9');
  expect(millisToTimeStr(890)).toEqual('00:00.9');
});
