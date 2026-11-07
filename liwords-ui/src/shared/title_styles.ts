// Map abbreviations to their display colors, shared by the circle by a
// username (display_title.tsx) and the tags on a profile
// (profile/organizations.tsx)
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
  NM: {
    // SG
    background: "#E1BEE7", // Purple/Light
    border: "#8E24AA", // Purple/Dark
    color: "#8E24AA",
  },
  SM: {
    // NASPA & SG
    background: "#FFC9C9",
    border: "#A92E2D",
    color: "#A92E2D",
  },
  M: {
    // WESPA & SG
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

const defaultTitleStyle = {
  background: "#e6f7ff",
  border: "#1890ff",
  color: "#1890ff",
};

export const titleStyle = (abbreviation: string) =>
  titleStyles[abbreviation] || defaultTitleStyle;
