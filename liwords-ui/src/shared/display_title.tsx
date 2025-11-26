import React from "react";
import { useBriefProfile } from "../utils/brief_profiles";
import { Tooltip } from "antd";

interface DisplayUserTitleProps {
  uuid?: string;
}

const titleStyles: Record<
  string,
  { background: string; border: string; color: string }
> = {
  Grandmaster: {
    background: "#C9FFCB", // UI/Light/Green
    border: "#449E2D", // UI/Dark/Green
    color: "#449E2D",
  },
  Master: {
    background: "#FFFDC9", // UI/Light/Yellow
    border: "#F4B000", // UI/Dark/Yellow
    color: "#F4B000",
  },
  Expert: {
    background: "#FFC9C9", // UI/Light/Red
    border: "#A92E2E", // UI/Dark/Red
    color: "#A92E2E",
  },
};

const titleAbbreviations: Record<string, string> = {
  Grandmaster: "GM",
  Master: "M",
  Expert: "E",
};

export const DisplayUserTitle: React.FC<DisplayUserTitleProps> = ({ uuid }) => {
  const briefProfile = useBriefProfile(uuid);

  if (!briefProfile || !briefProfile.title) {
    return null;
  }

  const abbreviation =
    titleAbbreviations[briefProfile.title] || briefProfile.title;
  const styles = titleStyles[briefProfile.title] || {
    background: "#e6f7ff",
    border: "#1890ff",
    color: "#1890ff",
  };

  return (
    <Tooltip title={`${briefProfile.title} title`}>
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          width: "24px",
          height: "24px",
          borderRadius: "50%",
          backgroundColor: styles.background,
          border: `1px solid ${styles.border}`,
          color: styles.color,
          fontSize: "11px",
          fontWeight: "bold",
          marginLeft: "4px",
          marginRight: "4px",
        }}
      >
        {abbreviation}
      </span>
    </Tooltip>
  );
};
