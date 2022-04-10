import React from 'react';

export const reloadAction = (
  <>
    Woogles has been updated. Please{' '}
    <span
      className="message-action"
      onClick={() => window.location.reload()}
      role="button"
    >
      refresh
    </span>{' '}
    at your convenience.
  </>
);
