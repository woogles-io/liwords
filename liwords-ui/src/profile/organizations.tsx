import React, { useEffect, useState } from "react";
import { Tag, Spin } from "antd";
import { useClient, flashError } from "../utils/hooks/connect";
import {
  OrganizationService,
  GetPublicOrganizationsRequestSchema,
  OrganizationTitle,
} from "../gen/api/proto/user_service/user_service_pb";
import { create } from "@bufbuild/protobuf";

interface DisplayUserOrganizationsProps {
  username: string;
}

const organizationNames: Record<string, string> = {
  naspa: "NASPA",
  wespa: "WESPA",
  absp: "ABSP",
};

// Title colors based on abbreviation
// Colors: gold (grandmaster), purple (international master), blue (master), green (expert)
const titleColors: Record<string, string> = {
  GM: "gold",
  IM: "purple",
  SM: "blue",
  M: "blue",
  EX: "green",
  EXP: "green",
};

// Full title names for display
const titleFullNames: Record<string, string> = {
  GM: "Grandmaster",
  IM: "International Master",
  SM: "NASPA Master",
  M: "Master",
  EX: "Expert",
  EXP: "Expert",
};

export const DisplayUserOrganizations: React.FC<
  DisplayUserOrganizationsProps
> = ({ username }) => {
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

  // Filter to only show organizations with titles (normalizedTitle contains the abbreviation)
  const orgsWithTitles = organizations.filter((org) => org.normalizedTitle);

  if (orgsWithTitles.length === 0) {
    return null;
  }

  return (
    <>
      <h2>Titles</h2>
      <div style={{ marginBottom: 16, marginLeft: 16 }}>
        {orgsWithTitles.map((org) => {
          const abbreviation = org.normalizedTitle;
          const fullName = titleFullNames[abbreviation] || abbreviation;
          const color = titleColors[abbreviation] || "default";

          return (
            <div key={org.organizationCode} style={{ marginBottom: 8 }}>
              <strong>
                {organizationNames[org.organizationCode] || org.organizationCode}:
              </strong>{" "}
              <Tag color={color} style={{ fontWeight: "bold" }}>
                {abbreviation}
              </Tag>
              <span style={{ color: "#666", fontSize: "0.9em" }}>
                ({fullName})
              </span>
            </div>
          );
        })}
      </div>
    </>
  );
};
