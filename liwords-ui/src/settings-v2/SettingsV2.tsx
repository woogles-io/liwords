/**
 * Settings V2 - Mantine-based Settings Page
 *
 * Pixel-perfect recreation of the original settings page using Mantine
 */

import React, { useState, useCallback } from 'react';
import { useParams } from 'react-router';
import { Box } from '@mantine/core';
import { IconHeartFilled } from '@tabler/icons-react';
import { TopBar } from '../navigation/topbar';
import { useLoginStateStoreContext } from '../store/store';
import { PreferencesV2 } from './PreferencesV2';
import { PersonalInfoV2 } from './PersonalInfoV2';
import { useUIStore } from '../stores/ui-store';

// Tab values
const TABS = {
  PERSONAL_INFO: 'personal',
  CHANGE_PASSWORD: 'password',
  PREFERENCES: 'preferences',
  INTEGRATIONS: 'integrations',
  BLOCKED_PLAYERS: 'blocked',
  SECRET: 'secret',
  API: 'api',
  ROLES: 'roles',
  LOGOUT: 'logout',
  SUPPORT: 'support',
} as const;

type TabValue = typeof TABS[keyof typeof TABS];

const getInitialTab = (section: string | undefined, loggedIn: boolean): TabValue => {
  if (!loggedIn && section === 'donate') {
    return TABS.SUPPORT;
  }

  switch (section) {
    case 'donate':
    case 'support':
      return TABS.SUPPORT;
    case 'personal':
      return TABS.PERSONAL_INFO;
    case 'password':
      return TABS.CHANGE_PASSWORD;
    case 'preferences':
      return TABS.PREFERENCES;
    case 'integrations':
      return TABS.INTEGRATIONS;
    case 'secret':
      return TABS.SECRET;
    case 'blocked':
      return TABS.BLOCKED_PLAYERS;
    case 'logout':
      return TABS.LOGOUT;
    case 'api':
      return TABS.API;
    case 'roles':
      return TABS.ROLES;
    default:
      return TABS.PREFERENCES;
  }
};

interface CategoryChoiceProps {
  title: string | React.ReactNode;
  value: TabValue;
  active: boolean;
  onClick: () => void;
}

const CategoryChoice: React.FC<CategoryChoiceProps> = ({ title, active, onClick }) => {
  const themeMode = useUIStore((state) => state.themeMode);

  return (
    <Box
      onClick={onClick}
      style={{
        textAlign: 'right',
        height: 32,
        fontWeight: 'bold',
        color: active
          ? (themeMode === 'dark' ? '#c9f0ff' : '#11659e')
          : (themeMode === 'dark' ? '#ccc' : '#999'),
        cursor: 'pointer',
        paddingRight: 14,
        paddingTop: 8,
        borderRight: `2px solid ${active ? (themeMode === 'dark' ? '#c9f0ff' : '#11659e') : 'transparent'}`,
      }}
      onMouseEnter={(e) => {
        if (!active) {
          e.currentTarget.style.color = themeMode === 'dark' ? '#c9f0ff' : '#11659e';
        }
      }}
      onMouseLeave={(e) => {
        if (!active) {
          e.currentTarget.style.color = themeMode === 'dark' ? '#ccc' : '#999';
        }
      }}
    >
      {title}
    </Box>
  );
};

export const SettingsV2: React.FC = () => {
  const { loginState } = useLoginStateStoreContext();
  const { loggedIn } = loginState;
  const { section } = useParams();
  const themeMode = useUIStore((state) => state.themeMode);

  const [activeTab, setActiveTab] = useState<TabValue>(
    getInitialTab(section, loggedIn)
  );

  const handleTabChange = useCallback(
    (value: TabValue) => {
      setActiveTab(value);
      window.history.replaceState({}, 'settings', `/settings-v2/${value}`);
    },
    []
  );

  if (!loggedIn && activeTab !== TABS.SUPPORT) {
    return (
      <>
        <TopBar />
        <Box
          style={{
            padding: '51px 24px 48px',
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          <div style={{ textAlign: 'center' }}>
            Log in to see your settings
          </div>
        </Box>
      </>
    );
  }

  return (
    <>
      <TopBar />
      <Box
        style={{
          padding: '51px 24px 48px',
          lineHeight: '20px',
          display: 'flex',
          justifyContent: 'center',
        }}
      >
        {/* Left sidebar - categories */}
        <Box
          style={{
            padding: '4px 30px 0 0',
            minWidth: 180,
          }}
        >
          <CategoryChoice
            title="Personal info"
            value={TABS.PERSONAL_INFO}
            active={activeTab === TABS.PERSONAL_INFO}
            onClick={() => handleTabChange(TABS.PERSONAL_INFO)}
          />
          <CategoryChoice
            title="Change password"
            value={TABS.CHANGE_PASSWORD}
            active={activeTab === TABS.CHANGE_PASSWORD}
            onClick={() => handleTabChange(TABS.CHANGE_PASSWORD)}
          />
          <CategoryChoice
            title="Integrations"
            value={TABS.INTEGRATIONS}
            active={activeTab === TABS.INTEGRATIONS}
            onClick={() => handleTabChange(TABS.INTEGRATIONS)}
          />
          <CategoryChoice
            title="Preferences"
            value={TABS.PREFERENCES}
            active={activeTab === TABS.PREFERENCES}
            onClick={() => handleTabChange(TABS.PREFERENCES)}
          />
          <CategoryChoice
            title="Blocked players list"
            value={TABS.BLOCKED_PLAYERS}
            active={activeTab === TABS.BLOCKED_PLAYERS}
            onClick={() => handleTabChange(TABS.BLOCKED_PLAYERS)}
          />
          <CategoryChoice
            title="Secret features"
            value={TABS.SECRET}
            active={activeTab === TABS.SECRET}
            onClick={() => handleTabChange(TABS.SECRET)}
          />
          <CategoryChoice
            title="API"
            value={TABS.API}
            active={activeTab === TABS.API}
            onClick={() => handleTabChange(TABS.API)}
          />
          <CategoryChoice
            title="Roles & Permissions"
            value={TABS.ROLES}
            active={activeTab === TABS.ROLES}
            onClick={() => handleTabChange(TABS.ROLES)}
          />
          <CategoryChoice
            title="Log out"
            value={TABS.LOGOUT}
            active={activeTab === TABS.LOGOUT}
            onClick={() => handleTabChange(TABS.LOGOUT)}
          />
          <CategoryChoice
            title={
              <>
                <IconHeartFilled
                  size={14}
                  style={{
                    marginRight: 6,
                    color: themeMode === 'dark' ? '#ce5f66' : '#a92e2e',
                    verticalAlign: 'baseline',
                    position: 'relative',
                    top: 2,
                  }}
                />
                Support Woogles
              </>
            }
            value={TABS.SUPPORT}
            active={activeTab === TABS.SUPPORT}
            onClick={() => handleTabChange(TABS.SUPPORT)}
          />
        </Box>

        {/* Right content area */}
        <Box style={{ maxWidth: 800, flexGrow: 2 }}>
          {activeTab === TABS.PERSONAL_INFO && <PersonalInfoV2 />}
          {activeTab === TABS.CHANGE_PASSWORD && <div>Change Password (Coming soon)</div>}
          {activeTab === TABS.INTEGRATIONS && <div>Integrations (Coming soon)</div>}
          {activeTab === TABS.PREFERENCES && <PreferencesV2 />}
          {activeTab === TABS.BLOCKED_PLAYERS && <div>Blocked Players (Coming soon)</div>}
          {activeTab === TABS.SECRET && <div>Secret Features (Coming soon)</div>}
          {activeTab === TABS.API && <div>API (Coming soon)</div>}
          {activeTab === TABS.ROLES && <div>Roles & Permissions (Coming soon)</div>}
          {activeTab === TABS.LOGOUT && <div>Log Out (Coming soon)</div>}
          {activeTab === TABS.SUPPORT && <div>Support Woogles (Coming soon)</div>}
        </Box>
      </Box>
    </>
  );
};
