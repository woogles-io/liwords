import React from 'react';
import { Card, Dropdown, Menu } from 'antd';
import { PoolFormatType, PoolFormats } from '../constants/pool_formats';

type poolType = { [rune: string]: number };

// TODO: Store these elsewhere -- they're language specific
const VOWELS = 'AEIOU';
const CONSONANTS = 'BCDFGHJKLMNPQRSTVWXYZ';

function poolMinusRack(pool: poolType, rack: string) {
  const poolCopy = { ...pool };
  for (let i = 0; i < rack.length; i += 1) {
    poolCopy[rack[i]] -= 1;
  }
  return poolCopy;
}

function renderLetters(
  pool: poolType,
  possibleLetters: string,
  maxConsecutive: number = 6
) {
  const output = [];
  for (
    let possibility = 0;
    possibility < possibleLetters.length;
    possibility += 1
  ) {
    const letter = possibleLetters[possibility];
    let letterGroup = '';
    if (pool[letter]) {
      for (let i = 0; i < pool[letter]; i++) {
        if (i % maxConsecutive) {
          letterGroup += letter;
        } else {
          letterGroup += ` ${letter}`;
        }
      }
      output.push(
        <React.Fragment key={`lg-${letter}-${possibility}`}>
          <span className="letter-group" data-rune={letter}>
            {letterGroup.trim()}
          </span>{' '}
        </React.Fragment>
      );
    }
  }
  return (
    <section className="pool-section" key={possibleLetters}>
      {output}
    </section>
  );
}

function getPoolCount(
  pool: poolType,
  includeRunes = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ?'
) {
  return Object.keys(pool).reduce((acc, cur) => {
    if (includeRunes.lastIndexOf(cur) > -1) {
      return acc + pool[cur];
    }
    return acc;
  }, 0);
}

type Props = {
  omitCard?: boolean;
  pool: poolType;
  poolFormat: PoolFormatType;
  setPoolFormat: (format: PoolFormatType) => void;
  currentRack: string;
};

const Pool = React.memo((props: Props) => {
  const letterOrder =
    PoolFormats.find((f) => f.poolFormatType === props.poolFormat)?.format ||
    'ABCDEFGHIJKLMNOPQRSTUVWXYZ?';
  const pool = poolMinusRack(props.pool, props.currentRack);
  const letterSections = letterOrder
    .split(',')
    .map((letterSection) => renderLetters(pool, letterSection));
  const poolMenu = (
    <Menu>
      {PoolFormats.map((pf) => (
        <Menu.Item
          key={pf.poolFormatType}
          onClick={() => {
            localStorage?.setItem('poolFormat', pf.poolFormatType.toString());
            props.setPoolFormat(pf.poolFormatType);
          }}
        >
          {pf.displayName}
        </Menu.Item>
      ))}
    </Menu>
  );
  const dropDown = (
    <Dropdown
      overlay={poolMenu}
      trigger={['click']}
      placement="bottomRight"
      overlayClassName="format-dropdown"
    >
      <a href="/" onClick={(e) => e.preventDefault()}>
        Rearrange
      </a>
    </Dropdown>
  );

  const renderContents = (title?: string) => (
    <div className="pool">
      {title ? <p className="label">{title}</p> : null}
      <div className="tiles-remaining">{letterSections}</div>
      <div className="vc-distribution">
        <div>{getPoolCount(pool, VOWELS)} vowels</div>
        <div>{getPoolCount(pool, CONSONANTS)} consonants</div>
      </div>
    </div>
  );

  const unseen = getPoolCount(pool);
  const inbag = Math.max(unseen - 7, 0);

  let title: string;
  if (inbag === 0) {
    title = `Opponent has ${unseen} ${unseen === 1 ? 'tile' : 'tiles'}`;
  } else {
    title = `${inbag} ${inbag === 1 ? 'tile' : 'tiles'} in bag`;
  }

  if (props.omitCard) {
    return <>{renderContents(title)}</>;
  }
  return (
    <Card className="pool" title={title} extra={dropDown}>
      {renderContents()}
    </Card>
  );
});

export default Pool;
