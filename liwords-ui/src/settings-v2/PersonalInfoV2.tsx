/**
 * Personal Info V2 - Mantine-based Personal Info Panel
 *
 * Placeholder for personal info section
 */

import React from 'react';
import { Stack, Title, Text } from '@mantine/core';

export const PersonalInfoV2: React.FC = () => {
  return (
    <Stack gap="lg">
      <div>
        <Title order={3} mb="md">
          Personal Information
        </Title>
        <Text c="dimmed" size="sm">
          Manage your account details and preferences
        </Text>
      </div>

      <Text c="dimmed">
        Personal info features coming soon...
      </Text>
    </Stack>
  );
};
