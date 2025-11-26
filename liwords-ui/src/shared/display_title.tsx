import React from "react";
import { useBriefProfile } from "../utils/brief_profiles";
import { Tooltip } from "antd";

interface DisplayUserTitleProps {
  uuid?: string;
}

const titleColors: Record<string, string> = {
  GM: "#faad14", // gold
  Master: "#1890ff", // blue
  Expert: "#52c41a", // green
};

const titleAbbreviations: Record<string, string> = {
  GM: "GM",
  Master: "M",
  Expert: "EX",
};

export const DisplayUserTitle: React.FC<DisplayUserTitleProps> = ({ uuid }) => {
  const briefProfile = useBriefProfile(uuid);

  if (!briefProfile || !briefProfile.title) {
    return null;
  }

  const abbreviation =
    titleAbbreviations[briefProfile.title] || briefProfile.title;
  const color = titleColors[briefProfile.title] || "#1890ff";

  return (
    <Tooltip title={briefProfile.title}>
      <span
        style={{
          marginRight: "6px",
          fontWeight: "bold",
          color: color,
          fontFamily: "monospace",
        }}
      >
        {abbreviation}
      </span>
    </Tooltip>
  );
};
