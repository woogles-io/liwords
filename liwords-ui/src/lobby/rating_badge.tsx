import React from 'react';
import { Badge, Tag } from 'antd';
import { ratingToColor } from '../store/constants';

type Props = {
  rating: string;
  player: string;
};

export const RatingBadge = (props: Props) => {
  const [rating, color] = ratingToColor(props.rating);
  return (
    <span>
      {props.player} <Tag color={color}>{props.rating}</Tag>
    </span>
  );
};
