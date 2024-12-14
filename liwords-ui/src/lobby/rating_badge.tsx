import React from "react";
import { Tag } from "antd";
import { ratingToColor } from "../store/constants";

type Props = {
  rating: string;
};

export const RatingBadge = (props: Props) => {
  const [, color] = ratingToColor(props.rating);
  return <Tag color={color}>{props.rating}</Tag>;
};
