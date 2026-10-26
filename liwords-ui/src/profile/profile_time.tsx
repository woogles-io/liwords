import { Tooltip } from "antd";
import moment from "moment";
import React from "react";
import { useLocalStorageBool } from "../utils/use_local_storage";
import { ABSOLUTE_TIMES_KEY } from "./history_prefs";

// When something on a profile happened (a game played, an annotated game
// made, a collection updated), as "13 days ago" or, after a click on any of
// them, as dates (a page of games all "13 days ago" is otherwise hard to tell
// apart). Remembered for the next visit, and followed by other open tabs,
// like the chat display settings. The other form shows on hover.
export const ProfileTime = (props: { at?: Date }) => {
  const [absoluteTimes, setAbsoluteTimes] =
    useLocalStorageBool(ABSOLUTE_TIMES_KEY);
  if (!props.at) return null;
  const whenMoment = moment(props.at);
  return (
    <Tooltip
      title={
        absoluteTimes
          ? `${whenMoment.fromNow()}; click to show how long ago`
          : `${whenMoment.format("LLL")}; click to show dates`
      }
    >
      <span
        className="when-toggle"
        onClick={() => setAbsoluteTimes(!absoluteTimes)}
      >
        {absoluteTimes
          ? whenMoment.format("YYYY-MM-DD HH:mm")
          : whenMoment.fromNow()}
      </span>
    </Tooltip>
  );
};
