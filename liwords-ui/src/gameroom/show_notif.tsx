import React from "react";
import { Millis } from "../store/timer_controller";
import { TimedNotif } from "./timed_notif";
import { NotificationInstance } from "antd/lib/notification/interface";

type Props = {
  maxDuration: Millis;
  onExpire: () => void;
  onAccept?: () => void;
  onDecline?: () => void;
  introText: string;
  countdownText: string;
  acceptText: string;
  declineText: string;
  notification: NotificationInstance;
};

export const ShowNotif = (props: Props) => {
  const myId = React.useMemo(
    () => `notif/${Math.random()}/${performance.now()}`,
    [],
  );
  React.useEffect(() => {
    props.notification.open({
      // other params, TODO
      className: "cancel-notification",
      closeIcon: <div></div>,
      key: myId,
      message: "",
      description: <TimedNotif {...props} />,
      placement: "topRight",
      duration: 0,
    });
    return () => {
      props.notification.destroy(myId);
    };
  }, [props, myId]);
  return null;
};
