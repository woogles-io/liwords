import { Division } from "../gen/api/proto/ipc/league_pb";

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
