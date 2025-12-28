import {
  millisToTimeStr,
  millisToTimeStrWithoutDays,
} from "./timer_controller";

const milliday = (md: number) => 24 * 3600 * md;

const testCommonBehavior = (millisToTimeStr: (ms: number) => string) => {
  expect(millisToTimeStr(milliday(1000) - 1000)).toEqual("23:59:59");
  expect(millisToTimeStr(3600000)).toEqual("01:00:00");
  expect(millisToTimeStr(3599001)).toEqual("01:00:00");
  expect(millisToTimeStr(3599000)).toEqual("59:59");
  expect(millisToTimeStr(479900)).toEqual("08:00");
  expect(millisToTimeStr(479000)).toEqual("07:59");
  expect(millisToTimeStr(60000)).toEqual("01:00");
  expect(millisToTimeStr(59580)).toEqual("01:00");
  expect(millisToTimeStr(11000)).toEqual("00:11");
  expect(millisToTimeStr(10001)).toEqual("00:11");
  expect(millisToTimeStr(10000)).toEqual("00:10.0");
  expect(millisToTimeStr(9901)).toEqual("00:10.0");
  expect(millisToTimeStr(9900)).toEqual("00:09.9");
  expect(millisToTimeStr(8900)).toEqual("00:08.9");
  expect(millisToTimeStr(890)).toEqual("00:00.9");
  expect(millisToTimeStr(101)).toEqual("00:00.2");
  expect(millisToTimeStr(100)).toEqual("00:00.1");
  expect(millisToTimeStr(1)).toEqual("00:00.1");
  expect(millisToTimeStr(0)).toEqual("00:00.0");
  expect(millisToTimeStr(-1)).toEqual("-00:00.0");
  expect(millisToTimeStr(-40)).toEqual("-00:00.0");
  expect(millisToTimeStr(-89)).toEqual("-00:00.0");
  expect(millisToTimeStr(-99)).toEqual("-00:00.0");
  expect(millisToTimeStr(-100)).toEqual("-00:00.1");
  expect(millisToTimeStr(-890)).toEqual("-00:00.8");
  expect(millisToTimeStr(-990)).toEqual("-00:00.9");
  expect(millisToTimeStr(-999)).toEqual("-00:00.9");
  expect(millisToTimeStr(-1000)).toEqual("-00:01");
  expect(millisToTimeStr(-1999)).toEqual("-00:01");
  expect(millisToTimeStr(-2000)).toEqual("-00:02");
  expect(millisToTimeStr(-8900)).toEqual("-00:08");
  expect(millisToTimeStr(-10000)).toEqual("-00:10");
  expect(millisToTimeStr(-10300)).toEqual("-00:10");
  expect(millisToTimeStr(-10600)).toEqual("-00:10");
  expect(millisToTimeStr(-11000)).toEqual("-00:11");
  expect(millisToTimeStr(-59000)).toEqual("-00:59");
  expect(millisToTimeStr(-59700)).toEqual("-00:59");
  expect(millisToTimeStr(-60000)).toEqual("-01:00");
  expect(millisToTimeStr(-3599000)).toEqual("-59:59");
  expect(millisToTimeStr(-3599999)).toEqual("-59:59");
  expect(millisToTimeStr(-3600000)).toEqual("-01:00:00");
  expect(millisToTimeStr(-milliday(1000) + 1)).toEqual("-23:59:59");
};

it("tests millis to time without days", () => {
  const millisToTimeStr = millisToTimeStrWithoutDays;
  expect(millisToTimeStr(milliday(4000) - 999)).toEqual("4:00:00:00");
  expect(millisToTimeStr(milliday(4000) - 1000)).toEqual("3:23:59:59");
  expect(millisToTimeStr(milliday(3950) - 999)).toEqual("3:22:48:00");
  expect(millisToTimeStr(milliday(3950) - 1000)).toEqual("3:22:47:59");
  expect(millisToTimeStr(milliday(3900) - 999)).toEqual("3:21:36:00");
  expect(millisToTimeStr(milliday(3900) - 1000)).toEqual("3:21:35:59");
  expect(millisToTimeStr(milliday(3850) - 999)).toEqual("3:20:24:00");
  expect(millisToTimeStr(milliday(3850) - 1000)).toEqual("3:20:23:59");
  expect(millisToTimeStr(milliday(1150) - 999)).toEqual("1:03:36:00");
  expect(millisToTimeStr(milliday(1150) - 1000)).toEqual("1:03:35:59");
  expect(millisToTimeStr(milliday(1100) - 999)).toEqual("1:02:24:00");
  expect(millisToTimeStr(milliday(1100) - 1000)).toEqual("1:02:23:59");
  expect(millisToTimeStr(milliday(1050) - 999)).toEqual("1:01:12:00");
  expect(millisToTimeStr(milliday(1050) - 1000)).toEqual("1:01:11:59");
  expect(millisToTimeStr(milliday(1000) - 999)).toEqual("1:00:00:00");
  testCommonBehavior(millisToTimeStr);
  expect(millisToTimeStr(-milliday(1000))).toEqual("-1:00:00:00");
  expect(millisToTimeStr(-milliday(1050) + 1)).toEqual("-1:01:11:59");
  expect(millisToTimeStr(-milliday(1050))).toEqual("-1:01:12:00");
  expect(millisToTimeStr(-milliday(1100) + 1)).toEqual("-1:02:23:59");
  expect(millisToTimeStr(-milliday(1100))).toEqual("-1:02:24:00");
  expect(millisToTimeStr(-milliday(1150) + 1)).toEqual("-1:03:35:59");
  expect(millisToTimeStr(-milliday(1150))).toEqual("-1:03:36:00");
  expect(millisToTimeStr(-milliday(3850) + 1)).toEqual("-3:20:23:59");
  expect(millisToTimeStr(-milliday(3850))).toEqual("-3:20:24:00");
  expect(millisToTimeStr(-milliday(3900) + 1)).toEqual("-3:21:35:59");
  expect(millisToTimeStr(-milliday(3900))).toEqual("-3:21:36:00");
  expect(millisToTimeStr(-milliday(3950) + 1)).toEqual("-3:22:47:59");
  expect(millisToTimeStr(-milliday(3950))).toEqual("-3:22:48:00");
  expect(millisToTimeStr(-milliday(4000) + 1)).toEqual("-3:23:59:59");
  expect(millisToTimeStr(-milliday(4000))).toEqual("-4:00:00:00");
});

it("tests millis to time", () => {
  // for anything <= 4:00:00:00 and > 3:23:59:59 (for example 3:23:59:59.001),
  // millisToTimeStrWithoutDays rounds up to "4:00:00:00".
  // millisToTimeStr likewise returns "4.0 days" for 3:23:59:59.001.
  // this is by design, because this way, millisToTimeStr gets invalidated only
  // when millisToTimeStrWithoutDays does.
  expect(millisToTimeStr(milliday(4000) - 999)).toEqual("4.0 days");
  expect(millisToTimeStr(milliday(4000) - 1000)).toEqual("3.9 days");
  expect(millisToTimeStr(milliday(3950) - 999)).toEqual("3.9 days");
  expect(millisToTimeStr(milliday(3950) - 1000)).toEqual("3.9 days");
  expect(millisToTimeStr(milliday(3900) - 999)).toEqual("3.9 days");
  expect(millisToTimeStr(milliday(3900) - 1000)).toEqual("3.8 days");
  expect(millisToTimeStr(milliday(3850) - 999)).toEqual("3.8 days");
  expect(millisToTimeStr(milliday(3850) - 1000)).toEqual("3.8 days");
  expect(millisToTimeStr(milliday(1150) - 999)).toEqual("1.1 days");
  expect(millisToTimeStr(milliday(1150) - 1000)).toEqual("1.1 days");
  expect(millisToTimeStr(milliday(1100) - 999)).toEqual("1.1 days");
  expect(millisToTimeStr(milliday(1100) - 1000)).toEqual("1.0 day");
  expect(millisToTimeStr(milliday(1050) - 999)).toEqual("1.0 day");
  expect(millisToTimeStr(milliday(1050) - 1000)).toEqual("1.0 day");
  expect(millisToTimeStr(milliday(1000) - 999)).toEqual("1.0 day");
  testCommonBehavior(millisToTimeStr);
  expect(millisToTimeStr(-milliday(1000))).toEqual("-1.0 day");
  expect(millisToTimeStr(-milliday(1050) + 1)).toEqual("-1.0 day");
  expect(millisToTimeStr(-milliday(1050))).toEqual("-1.0 day");
  expect(millisToTimeStr(-milliday(1100) + 1)).toEqual("-1.0 day");
  expect(millisToTimeStr(-milliday(1100))).toEqual("-1.1 days");
  expect(millisToTimeStr(-milliday(1150) + 1)).toEqual("-1.1 days");
  expect(millisToTimeStr(-milliday(1150))).toEqual("-1.1 days");
  expect(millisToTimeStr(-milliday(3850) + 1)).toEqual("-3.8 days");
  expect(millisToTimeStr(-milliday(3850))).toEqual("-3.8 days");
  expect(millisToTimeStr(-milliday(3900) + 1)).toEqual("-3.8 days");
  expect(millisToTimeStr(-milliday(3900))).toEqual("-3.9 days");
  expect(millisToTimeStr(-milliday(3950) + 1)).toEqual("-3.9 days");
  expect(millisToTimeStr(-milliday(3950))).toEqual("-3.9 days");
  expect(millisToTimeStr(-milliday(4000) + 1)).toEqual("-3.9 days");
  expect(millisToTimeStr(-milliday(4000))).toEqual("-4.0 days");
});
