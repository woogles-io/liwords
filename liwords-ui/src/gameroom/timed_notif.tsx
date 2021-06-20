import { Button } from 'antd';
import React from 'react';
import { Millis } from '../store/timer_controller';
import { SimpleTimer } from './simple_timer';

type Props = {
  maxDuration: Millis;
  onExpire: () => void;
  onAccept?: () => void;
  onDecline?: () => void;
  introText: string;
  countdownText: string;
  acceptText: string;
  declineText: string;
};

export const TimedNotif = (props: Props) => {
  const {
    maxDuration,
    onExpire,
    onAccept,
    onDecline,
    introText,
    countdownText,
    acceptText,
    declineText,
  } = props;
  const whenLoaded = React.useMemo(() => performance.now() as Millis, []);
  React.useEffect(() => {
    // andy: i prefer not to write setTimeout(onExpire, ...)
    const t = setTimeout(() => onExpire(), maxDuration);
    return () => clearTimeout(t);
  }, [maxDuration, onExpire]);

  // in antd, "primary" button is blue. this is perfect for the non-primary button.
  // (please facepalm.)

  return (
    <div className="timed-notification">
      <p>{introText}</p>
      <p>
        {countdownText}
        <SimpleTimer
          lastRefreshedPerformanceNow={whenLoaded}
          millisAtLastRefresh={maxDuration}
          isRunning
        />
      </p>
      <p className="row-of-buttons">
        {onDecline && (
          <Button
            type={onDecline && onAccept ? 'primary' : undefined}
            onClick={onDecline}
          >
            {declineText}
          </Button>
        )}
        {onAccept && <Button onClick={onAccept}>{acceptText}</Button>}
      </p>
    </div>
  );
};
