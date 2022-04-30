import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import * as serviceWorker from './serviceWorker';
import { Store } from './store/store';
import { BriefProfiles } from './utils/brief_profiles';
import { postJsonObj } from './api/api';

declare global {
  interface Window {
    RUNTIME_CONFIGURATION: { [key: string]: string };
    webkitAudioContext: typeof AudioContext;
  }
}

window.console.info(
  'Woogles.io is open source! https://github.com/domino14/liwords'
);

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

// Some magic code here to force everyone to use the naked domain before
// using Cloudfront to redirect:
{
  type jwtresp = {
    jwt: string;
  };
  const cookieHandoverFunc = async () => {
    await postJsonObj(
      'user_service.AuthenticationService',
      'GetSignedCookie',
      {},
      (response) => {
        const r = response as jwtresp;
        console.log('got jwt', r.jwt);
        const newPath = `/handover-signed-cookie?${new URLSearchParams({
          jwt: r.jwt,
          ls: JSON.stringify(localStorage),
          path: window.location.pathname,
        })}`;
        const loc = window.location;
        const protocol = loc.protocol;
        const hostname = loc.hostname;
        const nakedHost = hostname.replace(/www\./, '');
        window.location.replace(`${protocol}//${nakedHost}${newPath}`);
      },
      (err) => {} // ignore errors.
    );
  };
  if (window.location.hostname.startsWith('www.')) {
    cookieHandoverFunc();
  }
}

ReactDOM.render(
  <React.StrictMode>
    <BrowserRouter>
      <Store>
        <BriefProfiles>
          <App />
        </BriefProfiles>
      </Store>
    </BrowserRouter>
  </React.StrictMode>,
  document.getElementById('root')
);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://bit.ly/CRA-PWA
serviceWorker.unregister();
