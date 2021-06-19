import { notification } from 'antd';
import React from 'react';
import { TimedNotif } from './timed_notif';

type Props = {
  maxDuration: number;
  onExpire: () => void;
  onAccept?: () => void;
  onDecline?: () => void;
  introText: string;
  acceptText: string;
  declineText: string;
};

export const ShowNotif = (props: Props) => {
  const myId = React.useMemo(
    () => `notif/${Math.random()}/${performance.now()}`,
    []
  );
  React.useEffect(() => {
    notification.info({
      // other params, TODO
      closeIcon: <></>,
      key: myId,
      message: '',
      description: <TimedNotif {...props} />,
      placement: 'bottomRight',
      duration: 0,
    });
    return () => {
      notification.close(myId);
    };
  }, [props, myId]);
  return null;
};
