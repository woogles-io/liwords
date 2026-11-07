import React, { useCallback, useState } from "react";
import { create } from "@bufbuild/protobuf";
import {
  GetPublicOrganizationsRequestSchema,
  OrganizationService,
  OrganizationTitle,
} from "../gen/api/proto/user_service/user_service_pb";
import { useBriefProfile } from "../utils/brief_profiles";
import { useClient } from "../utils/hooks/connect";
import { titleStyle } from "./title_styles";
import { Tooltip } from "antd";

import "./display_title.scss";

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

const organizationName = (code: string) => organizationNames[code] || code;

// WESPA, the world body: the circle of one of its titles is ringed
// (display_title.scss)
const worldOrganization = "wespa";

// Every verified title of a player, by username, fetched once, as the
// circle's tooltip first opens; a failed fetch is tried again the next time.
const titlesByUsername = new Map<string, Promise<Array<OrganizationTitle>>>();

// The tooltip's lines: the title the circle shows first, then the player's
// other titles in the order the server lists them. Only verified titles, as
// the circle's: an admin is sent the unverified links too.
export const titleLines = (
  titles: Array<OrganizationTitle>,
  shownOrganization: string,
): Array<string> => {
  const held = titles.filter((t) => t.verified && t.rawTitle);
  return [
    ...held.filter((t) => t.organizationCode === shownOrganization),
    ...held.filter((t) => t.organizationCode !== shownOrganization),
  ].map(
    (t) =>
      `${t.titleFullName || t.rawTitle} (${organizationName(t.organizationCode)})`,
  );
};

export const DisplayUserTitle: React.FC<DisplayUserTitleProps> = ({ uuid }) => {
  const briefProfile = useBriefProfile(uuid);
  const orgClient = useClient(OrganizationService);
  const [lines, setLines] = useState<Array<string> | null>(null);
  const username = briefProfile?.username;
  const shownOrganization = briefProfile?.titleOrganizationCode ?? "";

  const listTitles = useCallback(
    (open: boolean) => {
      if (!open || lines || !username) {
        return;
      }
      let titles = titlesByUsername.get(username);
      if (!titles) {
        titles = orgClient
          .getPublicOrganizations(
            create(GetPublicOrganizationsRequestSchema, { username }),
          )
          .then((response) => response.titles);
        titlesByUsername.set(username, titles);
        titles.catch(() => titlesByUsername.delete(username));
      }
      titles.then(
        (held) => setLines(titleLines(held, shownOrganization)),
        () => {},
      );
    },
    [lines, username, orgClient, shownOrganization],
  );

  if (!briefProfile || !briefProfile.titleAbbreviation) {
    return null;
  }

  const abbreviation = briefProfile.titleAbbreviation;
  const styles = titleStyle(abbreviation);

  const orgName = briefProfile.titleOrganizationCode
    ? organizationName(briefProfile.titleOrganizationCode)
    : "";
  const tooltipText = orgName
    ? `${briefProfile.title} (${orgName})`
    : briefProfile.title;

  return (
    <Tooltip
      title={(lines?.length ? lines : [tooltipText]).map((line) => (
        <div className="title-line" key={line}>
          {line}
        </div>
      ))}
      rootClassName="title-tooltip"
      mouseEnterDelay={0.3}
      onOpenChange={listTitles}
    >
      <span
        className={
          briefProfile.titleOrganizationCode === worldOrganization
            ? "title-circle world"
            : "title-circle"
        }
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
          cursor: "default",
          verticalAlign: "middle",
        }}
      >
        {abbreviation}
      </span>
    </Tooltip>
  );
};
