import React from "react";
import { Select } from "antd";
import { Division } from "../gen/api/proto/ipc/league_pb";

const { Option } = Select;

type DivisionSelectorProps = {
  divisions: Division[];
  selectedDivisionId: string;
  onDivisionChange: (divisionId: string) => void;
  currentUserId?: string;
};

export const DivisionSelector: React.FC<DivisionSelectorProps> = ({
  divisions,
  selectedDivisionId,
  onDivisionChange,
}) => {
  if (divisions.length === 0) {
    return null;
  }

  return (
    <div style={{ marginBottom: 16 }}>
      <Select
        value={selectedDivisionId}
        onChange={onDivisionChange}
        style={{ width: 200 }}
        placeholder="Select Division"
      >
        {divisions.map((division) => (
          <Option key={division.uuid} value={division.uuid}>
            {division.divisionName || `Division ${division.divisionNumber}`}
          </Option>
        ))}
      </Select>
    </div>
  );
};

// Helper function to find the user's division
export const findUserDivision = (
  divisions: Division[],
  userId: string,
): Division | undefined => {
  return divisions.find((division) =>
    division.standings.some(
      (standing: { userId: string }) => standing.userId === userId,
    ),
  );
};

// Helper function to get default division
export const getDefaultDivisionId = (
  divisions: Division[],
  userId?: string,
): string => {
  if (divisions.length === 0) return "";

  if (userId) {
    const userDivision = findUserDivision(divisions, userId);
    if (userDivision) return userDivision.uuid;
  }

  // Default to Division 1 (typically the first division)
  return divisions[0].uuid;
};
