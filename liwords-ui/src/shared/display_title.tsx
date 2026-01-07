import React from "react";
import { useBriefProfile } from "../utils/brief_profiles";
import { Tooltip } from "antd";

interface DisplayUserTitleProps {
  uuid?: string;
}

// Map organization codes to display names
const organizationNames: Record<string, string> = {
  naspa: "NASPA",
  wespa: "WESPA",
  absp: "ABSP",
};

// Map abbreviations to their display colors
const titleStyles: Record<
  string,
  { background: string; border: string; color: string }
> = {
  GM: {
    // NASPA & WESPA
    background: "#C9FFCB", // Green/Light
    border: "#449E2D", // Green/Dark
    color: "#449E2D",
  },
  IM: {
    // WESPA
    background: "#E1BEE7", // Purple/Light
    border: "#8E24AA", // Purple/Dark
    color: "#8E24AA",
  },
  SM: {
    // NASPA
    background: "#FFC9C9",
    border: "#A92E2D",
    color: "#A92E2D",
  },
  M: {
    // WESPA
    background: "#FFFDC9", // Yellow/Light
    border: "#F4B000", // Yellow/Dark
    color: "#F4B000",
  },
  EX: {
    // NASPA
    background: "#FFFDC9", // Red/Light
    border: "#F4B000", // Red/Dark
    color: "#F4B000",
  },
  EXP: {
    // ABSP
    background: "#FFC9C9", // Red/Light
    border: "#A92E2D", // Red/Dark
    color: "#A92E2D",
  },
};

export const DisplayUserTitle: React.FC<DisplayUserTitleProps> = ({ uuid }) => {
  const briefProfile = useBriefProfile(uuid);

  if (!briefProfile || !briefProfile.titleAbbreviation) {
    return null;
  }

  const abbreviation = briefProfile.titleAbbreviation;
  const styles = titleStyles[abbreviation] || {
    background: "#e6f7ff",
    border: "#1890ff",
    color: "#1890ff",
  };

  const orgName = briefProfile.titleOrganizationCode
    ? organizationNames[briefProfile.titleOrganizationCode] ||
      briefProfile.titleOrganizationCode
    : "";
  const tooltipText = orgName
    ? `${briefProfile.title} (${orgName})`
    : briefProfile.title;

  return (
    <Tooltip title={tooltipText} mouseEnterDelay={0.3}>
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          justifyContent: "center",
          width: "20px",
          height: "20px",
          borderRadius: "50%",
          backgroundColor: styles.background,
          border: `1px solid ${styles.border}`,
          color: styles.color,
          fontSize: "9px",
          fontWeight: "bold",
          marginLeft: "8px",
          cursor: "default",
          verticalAlign: "middle",
        }}
      >
        {abbreviation}
      </span>
    </Tooltip>
  );
};
