import React, { useCallback } from 'react';
import { useMountedState } from '../utils/mounted';
import { Col, Row, Select, Switch } from 'antd';
import { preferredSortOrder, setPreferredSortOrder } from '../store/constants';

type Props = {};

const KNOWN_TILE_ORDERS = [
  {
    name: 'Alphabetical',
    value: '',
  },
  {
    name: 'Vowels first',
    value: 'AEIOU',
  },
  {
    name: 'Consonants first',
    value: 'BCDFGHJKLMNPQRSTVWXYZ',
  },
  {
    name: 'Descending points',
    value: 'QZJXKFHVWYBCMPDG',
  },
  {
    name: 'Blanks first',
    value: '?',
  },
];

function toggleLocalSetting(key: string) {
  const value = localStorage?.getItem(key) !== 'true';
  localStorage.setItem(key, value ? 'true' : 'false');
  return value;
}

function localSetting(key: string) {
  return localStorage?.getItem(key) === 'true';
}

export const Preferences = React.memo((props: Props) => {
  const { useState } = useMountedState();

  const [darkMode, setDarkMode] = useState(
    localStorage?.getItem('darkMode') === 'true'
  );
  const toggleDarkMode = useCallback(() => {
    const useDarkMode = localStorage?.getItem('darkMode') !== 'true';
    localStorage.setItem('darkMode', useDarkMode ? 'true' : 'false');
    if (useDarkMode) {
      document?.body?.classList?.add('mode--dark');
      document?.body?.classList?.remove('mode--default');
    } else {
      document?.body?.classList?.add('mode--default');
      document?.body?.classList?.remove('mode--dark');
    }
    setDarkMode((x) => !x);
  }, []);

  const showRemainingKey = 'showRemainingTiles';
  const [showRemaining, setShowRemaining] = useState(
    localSetting(showRemainingKey)
  );
  const toggleShowRemaining = useCallback(() => {
    setShowRemaining(toggleLocalSetting(showRemainingKey));
  }, []);

  const seeCSWKey = 'enableLexiconCSW';
  const seeNWLKey = 'enableLexiconNWL';
  const seePolishKey = 'enableLexiconPolish';
  const seeNorwegianKey = 'enableLexiconNorwegian';

  const [seeCSW, setSeeCSW] = useState(localSetting(seeCSWKey));
  const toggleSeeCSW = useCallback(() => {
    setSeeCSW(toggleLocalSetting(seeCSWKey));
  }, []);

  const [seeNWL, setSeeNWL] = useState(localSetting(seeNWLKey));
  const toggleSeeNWL = useCallback(() => {
    setSeeNWL(toggleLocalSetting(seeNWLKey));
  }, []);

  const [seePolish, setSeePolish] = useState(localSetting(seePolishKey));
  const toggleSeePolish = useCallback(() => {
    setSeePolish(toggleLocalSetting(seePolishKey));
  }, []);

  const [seeNorwegian, setSeeNorwegian] = useState(
    localSetting(seeNorwegianKey)
  );
  const toggleSeeNorwegian = useCallback(() => {
    setSeeNorwegian(toggleLocalSetting(seeNorwegianKey));
  }, []);

  const [tileOrder, setTileOrder] = useState(preferredSortOrder ?? '');
  const handleTileOrderChange = useCallback((value) => {
    setTileOrder(value);
    setPreferredSortOrder(value);
  }, []);

  return (
    <div className="preferences">
      <h3>Preferences</h3>
      <div className="section-header">Display</div>
      <div className="toggle-section">
        <div className="title">Dark mode</div>
        <div>Use the dark version of the Woogles UI on Woogles.io</div>
        <Switch
          defaultChecked={darkMode}
          onChange={toggleDarkMode}
          className="dark-toggle"
        />
      </div>
      <div className="section-header">OMGWords settings</div>
      <Row>
        <Col span={12}>
          <div className="tile-order">Default tile order</div>
          <Select
            className="tile-order-select"
            size="large"
            defaultValue={tileOrder}
            onChange={handleTileOrderChange}
          >
            {KNOWN_TILE_ORDERS.map(({ name, value }) => (
              <Select.Option value={value} key={value}>
                {name}
              </Select.Option>
            ))}
            {KNOWN_TILE_ORDERS.some(({ value }) => value === tileOrder) || (
              <Select.Option value={tileOrder}>Custom</Select.Option>
            )}
          </Select>
        </Col>
      </Row>
      <div className="toggle-section">
        <div className="title">Show letters remaining</div>
        <div>See what letters are left in the tile bag during the game</div>
        <Switch
          defaultChecked={showRemaining}
          onChange={toggleShowRemaining}
          className="dark-toggle"
        />
      </div>
      <div className="section-header">Languages</div>
      <div className="toggle-section">
        <div className="title">English</div>
        <div>See English game requests (CSW)</div>
        <Switch
          defaultChecked={seeCSW}
          onChange={toggleSeeCSW}
          className="dark-toggle"
        />
      </div>
      <div className="toggle-section">
        <div className="title">English</div>
        <div>See American English game requests (NWL)</div>
        <Switch
          defaultChecked={seeNWL}
          onChange={toggleSeeNWL}
          className="dark-toggle"
        />
      </div>
      <div className="toggle-section">
        <div className="title">Polish</div>
        <div>See American Polish game requests</div>
        <Switch
          defaultChecked={seePolish}
          onChange={toggleSeePolish}
          className="dark-toggle"
        />
      </div>
      <div className="toggle-section">
        <div className="title">Norwegian</div>
        <div>See American Norwegian game requests</div>
        <Switch
          defaultChecked={seeNorwegian}
          onChange={toggleSeeNorwegian}
          className="dark-toggle"
        />
      </div>
    </div>
  );
});
