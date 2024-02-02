import { defaultTheme, darkTheme } from '@ant-design/compatible';
import { theme } from 'antd';

export const liwordsDefaultTheme = {
  ...defaultTheme,
  token: {
    ...defaultTheme.token,
    fontFamily: 'Mulish',
  },
};

export const liwordsDarkTheme = {
  ...darkTheme,
  algorithm: theme.darkAlgorithm,
  token: {
    ...darkTheme.token,
    fontFamily: 'Mulish',
    // See color_modes.scss
    // colorBgBase: '#3A3A3A',
  },
};
