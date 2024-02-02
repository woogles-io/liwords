import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { Store as LegacyStore } from './store/store';
import { BriefProfiles } from './utils/brief_profiles';

import { Provider } from 'react-redux';

import 'antd/dist/reset.css';
import './index.css';
import { store } from './store/redux_store';

declare global {
  interface Window {
    RUNTIME_CONFIGURATION: { [key: string]: string };
    webkitAudioContext: typeof AudioContext;
  }
}

window.console.info(
  'Woogles.io is open source! https://github.com/woogles-io/liwords'
);

// Scope the variables declared here.
{
  const oldValue = localStorage.getItem('enableWordSmog');
  if (oldValue) {
    if (!localStorage.getItem('enableVariants')) {
      localStorage.setItem('enableVariants', oldValue);
    }
    localStorage.removeItem('enableWordSmog');
  }
}

// Scope the variables declared here.
{
  // Adjust this constant accordingly.
  const minimumViableWidth = 558;
  const metaViewport = document.querySelector("meta[name='viewport']");
  if (!metaViewport) {
    // Should not happen because this is in public/index.html.
    throw new Error('missing meta');
  }
  const resizeFunc = () => {
    let desiredViewport = 'width=device-width, initial-scale=1';
    const deviceWidth = window.outerWidth;
    if (deviceWidth < minimumViableWidth) {
      desiredViewport = `width=${minimumViableWidth}, initial-scale=${
        deviceWidth / minimumViableWidth
      }`;
    }
    metaViewport.setAttribute('content', desiredViewport);
  };
  window.addEventListener('resize', resizeFunc);
  resizeFunc();
}

const container = document.getElementById('root');
const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <BrowserRouter>
      <Provider store={store}>
        {/* legacy store will be slowly decommissioned */}
        <LegacyStore>
          <BriefProfiles>
            <App />
          </BriefProfiles>
        </LegacyStore>
      </Provider>
    </BrowserRouter>
  </React.StrictMode>
);
