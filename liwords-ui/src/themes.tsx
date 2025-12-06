import { theme } from "antd";

// We should migrate to using antd component tokens as much as possible,
// and make the custom SCSS as short as possible. This will be a long process,
// likely, but all new styles should end up here.
// We should probably also make these variables!

const tokenoverrides = {
  fontFamily: "Mulish",
};

// componentOverrides are for default mode
const componentOverrides = {
  Input: {
    colorBorder: "#b9b9b9",
  },
  Dropdown: {
    paddingBlock: 10,
  },
  Checkbox: {
    colorBgContainer: "#ffffff",
  },
  Button: {
    borderRadius: 0,
  },
};

const lightComponentOverrides = {
  // Our light-mode notifications and message have a dark background,
  // and vice-versa
  Notification: {
    colorText: "#ffffff",
    colorBgElevated: "#11659e",
    colorTextHeading: "#ffffff",
    colorInfo: "#e2f8ff",
    colorIcon: "#ffffff",
  },
  Message: {
    colorText: "#ffffff",
    contentBg: "#11659e",
    colorInfo: "#e2f8ff",
  },
  Modal: {
    colorText: "#282828",
  },
  Table: {
    rowHoverBg: "unset",
    headerSortHoverBg: "unset",
    headerFilterHoverBg: "unset",
    rowSelectedHoverBg: "unset",
    colorBgContainer: "#ffffff",
  },
};

const darkComponentOverrides = {
  Notification: {
    colorText: "#282828",
    colorBgElevated: "#C9F0FF",
    colorTextHeading: "#282828",
    colorInfo: "#135380",
    colorIcon: "#282828",
    // Correct token names for antd 5.x
    notificationBg: "#C9F0FF",
    colorInfoBg: "#C9F0FF",
    colorSuccessBg: "#C9F0FF",
    colorWarningBg: "#C9F0FF",
    colorErrorBg: "#C9F0FF",
  },
  Message: {
    colorText: "#282828",
    contentBg: "#C9F0FF",
    colorInfo: "#135380",
  },
  Modal: {
    colorText: "#000000",
  },
  Table: {
    rowHoverBg: "unset",
    headerSortHoverBg: "unset",
    headerFilterHoverBg: "unset",
    rowSelectedHoverBg: "unset",
    colorBgContainer: "#3a3a3a",
  },
};

export const liwordsDefaultTheme = {
  algorithm: theme.defaultAlgorithm,
  token: {
    ...tokenoverrides,
  },
  components: { ...componentOverrides, ...lightComponentOverrides },
};

export const liwordsDarkTheme = {
  algorithm: theme.darkAlgorithm,
  token: {
    ...tokenoverrides,
  },
  components: { ...componentOverrides, ...darkComponentOverrides },
};
