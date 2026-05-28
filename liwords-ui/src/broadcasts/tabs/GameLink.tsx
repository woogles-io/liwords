import React from "react";
import { Link } from "react-router";
import { LinkOutlined } from "@ant-design/icons";

export const GameLink: React.FC<{ gameUuid: string; label?: string }> = ({
  gameUuid,
  label = "Watch",
}) => (
  <Link to={`/anno/${gameUuid}`}>
    <LinkOutlined /> {label}
  </Link>
);
