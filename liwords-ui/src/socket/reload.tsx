/* eslint-disable jsx-a11y/interactive-supports-focus */
/* eslint-disable jsx-a11y/click-events-have-key-events */
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
