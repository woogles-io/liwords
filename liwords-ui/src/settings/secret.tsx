import React, { useCallback } from 'react';
import { Switch } from 'antd';
import { useMountedState } from '../utils/mounted';
type Props = {};

export const Secret = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [telestrator, setTelestrator] = useState(
    localStorage?.getItem('enableScreenDrawing') === 'true'
  );
  const toggleTelestrator = useCallback(() => {
    const useTelestrator =
      localStorage?.getItem('enableScreenDrawing') !== 'true';
    localStorage.setItem(
      'enableScreenDrawing',
      useTelestrator ? 'true' : 'false'
    );
    setTelestrator((x) => !x);
  }, []);

  const [enableAllLexicons, setEnableAllLexicons] = useState(
    localStorage?.getItem('enableAllLexicons') === 'true'
  );
  const toggleEnableAllLexicons = useCallback(() => {
    const wantEnableAllLexicons =
      localStorage?.getItem('enableAllLexicons') !== 'true';
    localStorage.setItem(
      'enableAllLexicons',
      wantEnableAllLexicons ? 'true' : 'false'
    );
    setEnableAllLexicons((x) => !x);
  }, []);

  const [blindfold, setBlindfold] = useState(
    localStorage?.getItem('enableBlindfoldMode') === 'true'
  );
  const toggleBlindfold = useCallback(() => {
    const useBlindfold =
      localStorage?.getItem('enableBlindfoldMode') !== 'true';
    localStorage.setItem(
      'enableBlindfoldMode',
      useBlindfold ? 'true' : 'false'
    );
    setBlindfold((x) => !x);
  }, []);
  return (
    <div className="preferences secret">
      <h3>Secret Features</h3>
      <div className="secret-warning">
        Please use these secret, experimental features at your own discretion.
        They may be limited in functionality and/or impact your Woogles user
        experience.{' '}
        <a
          href="https://github.com/domino14/liwords/wiki/Secret-features"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn More.
        </a>
      </div>
      <div>
        <div className="toggle-section">
          <div className="title">Telestrator</div>
          <div>Draw on the board while you’re playing</div>
          <Switch
            defaultChecked={telestrator}
            onChange={toggleTelestrator}
            className="telestrator-toggle"
          />
        </div>
        <div className="toggle-section">
          <div className="title">Blindfold</div>
          <div>Enable text-to-speech keyboard commands</div>
          <Switch
            defaultChecked={blindfold}
            onChange={toggleBlindfold}
            className="blindfold-toggle"
          />
        </div>
        <div className="toggle-section">
          <div className="title">Lexicons</div>
          <div>Enable all lexicons</div>
          <Switch
            defaultChecked={enableAllLexicons}
            onChange={toggleEnableAllLexicons}
            className="dark-toggle"
          />
        </div>
      </div>
    </div>
  );
});
