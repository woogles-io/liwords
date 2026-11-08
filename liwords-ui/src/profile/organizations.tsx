import React, { useEffect, useState } from "react";
import { Spin } from "antd";
import { TitleTag } from "../shared/title_tag";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  OrganizationService,
  GetPublicOrganizationsRequestSchema,
  OrganizationTitle,
} from "../gen/api/proto/user_service/user_service_pb";
import { create } from "@bufbuild/protobuf";

interface DisplayUserOrganizationsProps {
  username: string;
  // the organization of the title in the circle by the username, listed
  // first; the rest come in the tie order the server sends them in
  firstOrganizationCode?: string;
}

const organizationNames: Record<string, string> = {
  naspa: "NASPA",
  wespa: "WESPA",
  absp: "ABSP",
  sg: "Singapore Scrabble Association",
};

export const DisplayUserOrganizations: React.FC<
  DisplayUserOrganizationsProps
> = ({ username, firstOrganizationCode }) => {
  const [organizations, setOrganizations] = useState<OrganizationTitle[]>([]);
  const [loading, setLoading] = useState(true);

  const orgClient = useClient(OrganizationService);

  useEffect(() => {
    const fetchOrganizations = async () => {
      try {
        const response = await orgClient.getPublicOrganizations(
          create(GetPublicOrganizationsRequestSchema, {
            username,
          }),
        );
        setOrganizations(response.titles);
      } catch (e) {
        flashError(e);
      } finally {
        setLoading(false);
      }
    };

    fetchOrganizations();
  }, [username, orgClient]);

  if (loading) {
    return <Spin size="small" />;
  }

  // Filter to only show organizations with titles: the circle's first, then
  // the rest as they came
  const titled = organizations.filter((org) => org.rawTitle);
  const orgsWithTitles = [
    ...titled.filter((org) => org.organizationCode === firstOrganizationCode),
    ...titled.filter((org) => org.organizationCode !== firstOrganizationCode),
  ];

  if (orgsWithTitles.length === 0) {
    return null;
  }

  return (
    <>
      <h2>Titles</h2>
      <div style={{ marginBottom: 16, marginLeft: 16 }}>
        {orgsWithTitles.map((org) => {
          const abbreviation = org.rawTitle; // e.g., "GM", "SM"
          const fullName = org.titleFullName || abbreviation; // e.g., "Grandmaster"

          return (
            <div key={org.organizationCode} style={{ marginBottom: 8 }}>
              <strong>
                {organizationNames[org.organizationCode] ||
                  org.organizationCode}
                :
              </strong>{" "}
              <TitleTag
                abbreviation={abbreviation}
                style={{ fontWeight: "bold" }}
              >
                {fullName}
              </TitleTag>
              <span style={{ color: "#666", fontSize: "0.9em" }}>
                ({abbreviation})
              </span>
            </div>
          );
        })}
      </div>
    </>
  );
};
