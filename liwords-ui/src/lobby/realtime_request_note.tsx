import { Button } from "antd";
import React from "react";
import { SoughtGame } from "../store/reducers/lobby_reducer";

type Props = {
  requests: Array<SoughtGame>;
  onShow?: () => void;
};

// Tells the user that their real-time seeks or match requests are still
// open, on the correspondence tab, which does not list them.
export const RealtimeRequestNote = (props: Props) => {
  const n = props.requests.length;
  const matches = props.requests.filter((r) => r.receiverIsPermanent).length;
  const kind =
    matches === 0
      ? "seek"
      : matches === n
        ? "match request"
        : "seek or match request";
  return (
    <p className="realtime-request-note">
      {n === 1
        ? `You also have a real-time ${kind} open.`
        : `You also have ${n} real-time ${kind === "seek or match request" ? "seeks and match requests" : `${kind}s`} open.`}{" "}
      <Button type="link" onClick={props.onShow}>
        {n === 1 ? "Show it" : "Show them"}
      </Button>
    </p>
  );
};
