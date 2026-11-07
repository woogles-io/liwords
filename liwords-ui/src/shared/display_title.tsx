import React from "react";
import { useBriefProfile } from "../utils/brief_profiles";
import { titleStyle } from "./title_styles";
import { Tooltip } from "antd";

interface DisplayUserTitleProps {
  uuid?: string;
}

// Map organization codes to display names
const organizationNames: Record<string, string> = {
  naspa: "NASPA",
  wespa: "WESPA",
  absp: "ABSP",
  sg: "Singapore Scrabble Association",
};

export const DisplayUserTitle: React.FC<DisplayUserTitleProps> = ({ uuid }) => {
  const briefProfile = useBriefProfile(uuid);

  if (!briefProfile || !briefProfile.titleAbbreviation) {
    return null;
  }

  const abbreviation = briefProfile.titleAbbreviation;
  const styles = titleStyle(abbreviation);

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
