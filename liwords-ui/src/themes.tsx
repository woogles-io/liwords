import { defaultTheme, darkTheme } from '@ant-design/compatible';
import { theme } from 'antd';

const tokenoverrides = {
  fontFamily: 'Mulish',
};

const componentOverrides = {
  // Button: {
  //   controlHeight: 36,
  // },
};

export const liwordsDefaultTheme = {
  ...defaultTheme,
  token: {
    ...defaultTheme.token,
    ...tokenoverrides,
  },
  components: componentOverrides,
};

export const liwordsDarkTheme = {
  ...darkTheme,
  algorithm: theme.darkAlgorithm,
  token: {
    ...darkTheme.token,
    ...tokenoverrides,
    // See color_modes.scss
    // colorBgBase: '#3A3A3A',
  },
  components: componentOverrides,
};
