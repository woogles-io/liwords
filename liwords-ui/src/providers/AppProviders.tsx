/**
 * AppProviders - Dual provider system for gradual migration
 *
 * This component wraps the entire app with both:
 * 1. New Mantine + Zustand system
 * 2. Legacy Ant Design + Context store system
 *
 * During the migration, both systems coexist. As we migrate components,
 * they will use the new Mantine/Zustand system. Legacy components continue
 * to use the Ant Design/Context system.
 */

import React, { useEffect } from 'react';
import { MantineProvider, ColorSchemeScript } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { ConfigProvider, App as AntDApp } from 'antd';
import { mantineTheme } from '../theme/mantine-theme';
import { liwordsDefaultTheme, liwordsDarkTheme } from '../themes';
import { useUIStore, selectThemeMode } from '../stores/ui-store';

// Import Mantine styles
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';

interface AppProvidersProps {
  children: React.ReactNode;
}

/**
 * Inner component that accesses Zustand store
 * (must be inside Zustand provider to work)
 */
const ThemeSync: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const themeMode = useUIStore(selectThemeMode);

  // Sync theme mode to body class for legacy SCSS
  useEffect(() => {
    document.body.classList.remove('mode--default', 'mode--dark');
    document.body.classList.add(`mode--${themeMode === 'dark' ? 'dark' : 'default'}`);
  }, [themeMode]);

  // Select appropriate Ant Design theme
  const antdTheme = themeMode === 'dark' ? liwordsDarkTheme : liwordsDefaultTheme;

  return (
    <MantineProvider theme={mantineTheme} defaultColorScheme={themeMode}>
      <Notifications position="top-right" zIndex={2500} />

      {/* Legacy Ant Design provider */}
      <ConfigProvider theme={antdTheme}>
        <AntDApp>
          {children}
        </AntDApp>
      </ConfigProvider>
    </MantineProvider>
  );
};

/**
 * Main AppProviders component
 *
 * Wraps the app with all necessary providers for both old and new systems.
 * The QueryClientProvider and TransportProvider are expected to be set up
 * in index.tsx, so we don't duplicate them here.
 */
export const AppProviders: React.FC<AppProvidersProps> = ({ children }) => {
  return (
    <>
      {/* ColorSchemeScript must be in document head for SSR support */}
      <ColorSchemeScript defaultColorScheme="light" />

      {/* ThemeSync component accesses Zustand and sets up providers */}
      <ThemeSync>
        {children}
      </ThemeSync>
    </>
  );
};
