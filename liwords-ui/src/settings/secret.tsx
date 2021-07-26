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

  const [wordSmog, setWordSmog] = useState(
    localStorage?.getItem('enableWordSmog') === 'true'
  );
  const toggleWordSmog = useCallback(() => {
    const useWordSmog = localStorage?.getItem('enableWordSmog') !== 'true';
    localStorage.setItem('enableWordSmog', useWordSmog ? 'true' : 'false');
    setWordSmog((x) => !x);
  }, []);
  const [showEquityLoss, setShowEquityLoss] = useState(
    localStorage?.getItem('enableShowEquityLoss') === 'true'
  );
  const toggleShowEquityLoss = useCallback(() => {
    const useShowEquityLoss =
      localStorage?.getItem('enableShowEquityLoss') !== 'true';
    localStorage.setItem(
      'enableShowEquityLoss',
      useShowEquityLoss ? 'true' : 'false'
    );
    setShowEquityLoss((x) => !x);
  }, []);

  const [enableSilentSite, setEnableSilentSite] = useState(
    localStorage?.getItem('enableSilentSite') === 'true'
  );
  const toggleEnableSilentSite = useCallback(() => {
    const wantEnableSilentSite =
      localStorage?.getItem('enableSilentSite') !== 'true';
    localStorage.setItem(
      'enableSilentSite',
      wantEnableSilentSite ? 'true' : 'false'
    );
    setEnableSilentSite((x) => !x);
  }, []);

  return (
    <div className="preferences secret">
      <h3>Secret features</h3>
      <div className="secret-warning">
        Please use these secret, experimental features at your own discretion.
        They may be limited in functionality and/or impact your Woogles user
        experience.{' '}
        <a
          href="https://github.com/domino14/liwords/wiki/Secret-features"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn more.
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
        <div className="toggle-section">
          <div className="title">WordSmog</div>
          <div>Enable WordSmog</div>
          <Switch
            defaultChecked={wordSmog}
            onChange={toggleWordSmog}
            className="wordsmog-toggle"
          />
        </div>
        <div className="toggle-section">
          <div className="title">Show equity loss</div>
          <div>Show equity loss in analyzer</div>
          <Switch
            defaultChecked={showEquityLoss}
            onChange={toggleShowEquityLoss}
            className="show-equity-loss-toggle"
          />
        </div>
        <div className="toggle-section">
          <div className="title">Enable silent site</div>
          <div>Mute all sounds</div>
          <Switch
            defaultChecked={enableSilentSite}
            onChange={toggleEnableSilentSite}
            className="sounds-toggle"
          />
        </div>
      </div>
    </div>
  );
});
