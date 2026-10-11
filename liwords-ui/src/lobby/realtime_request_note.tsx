import { Button } from "antd";
import React from "react";
import { SoughtGame } from "../store/reducers/lobby_reducer";

type Props = {
  request: SoughtGame;
  onShow?: () => void;
};

// Tells the user that their real-time seek or match request is still open,
// on the correspondence tab, which does not list it.
export const RealtimeRequestNote = (props: Props) => (
  <p className="realtime-request-note">
    You also have a real-time{" "}
    {props.request.receiverIsPermanent ? "match request" : "seek"} open.{" "}
    <Button type="link" onClick={props.onShow}>
      Show it
    </Button>
  </p>
);
