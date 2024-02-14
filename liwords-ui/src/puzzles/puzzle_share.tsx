import React, { useCallback, useEffect, useMemo } from 'react';
import { Button, Tooltip } from 'antd';
import { ShareAltOutlined } from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';
import { calculatePuzzleScore, renderStars } from './puzzle_info';
import { singularCount } from '../utils/plural';
import { PuzzleStatus } from '../gen/api/proto/puzzle_service/puzzle_service_pb';

import variables from '../base.module.scss';
const { colorPrimary } = variables;

type Props = {
  puzzleID?: string;
  attempts?: number;
  solved: number;
};

export const PuzzleShareButton = (props: Props) => {
  const { puzzleID, attempts, solved } = props;
  const { useState } = useMountedState();
  const [showTooltip, setShowTooltip] = useState(false);
  const message = useMemo(() => {
    let messageBuilder = '';
    if (solved === PuzzleStatus.CORRECT && attempts !== undefined) {
      messageBuilder += `${renderStars(
        calculatePuzzleScore(true, attempts),
        true
      )}`;
      messageBuilder += ` I solved this puzzle at Woogles.io in ${singularCount(
        attempts,
        'try',
        'tries'
      )}!`;
    } else {
      messageBuilder += 'Check out this puzzle at Woogles.io.';
    }
    messageBuilder += ` https://woogles.io/puzzle/${puzzleID} Can you solve it?`;
    return messageBuilder;
  }, [puzzleID, attempts, solved]);

  const writeToClipboard = useCallback(() => {
    navigator.clipboard.writeText(message).then(() => {
      setShowTooltip(true);
    });
  }, [message]);

  useEffect(() => {
    if (!showTooltip) {
      return;
    }
    const timeout = setTimeout(() => {
      setShowTooltip(false);
    }, 1000);
    return () => clearTimeout(timeout);
  }, [showTooltip]);

  return (
    <Tooltip
      title="Copied to clipboard"
      trigger="click"
      open={showTooltip}
      color={colorPrimary}
    >
      <Button
        type="default"
        onClick={writeToClipboard}
        className="puzzle-share-button"
      >
        <ShareAltOutlined />
        Share
      </Button>
    </Tooltip>
  );
};
