import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone";
import { Timestamp, timestampFromMs } from "@bufbuild/protobuf/wkt";

dayjs.extend(utc);
dayjs.extend(timezone);

export const doesCurrentUserUse24HourTime = () => {
  const formatter = new Intl.DateTimeFormat(navigator.language, {
    hour: "numeric",
    hour12: undefined,
  });
  return !formatter.resolvedOptions().hour12;
};

export const protobufTimestampToDayjsIgnoringNanos = (timestamp: Timestamp) => {
  console.log("timestamp", timestamp.seconds);
  const date = dayjs.unix(Number(timestamp.seconds));
  return date;
};

export const dayjsToProtobufTimestampIgnoringNanos = (date: dayjs.Dayjs) => {
  return timestampFromMs(date.unix() * 1000);
};

export { dayjs };
