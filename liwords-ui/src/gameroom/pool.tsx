import React from 'react';
import { Link } from 'react-router-dom';
import { Card, Dropdown, Menu } from 'antd';
import { PoolFormatType, PoolFormats } from '../constants/pool_formats';
import { singularCount } from '../utils/plural';
import { Alphabet } from '../constants/alphabets';

type poolType = { [rune: string]: number };

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
  maxConsecutive = 6
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
  alphabet: Alphabet,
  filterfn: (a: Alphabet) => string
) {
  return Object.keys(pool).reduce((acc, cur) => {
    if (filterfn(alphabet).lastIndexOf(cur) > -1) {
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
  currentRack: Uint8Array;
  alphabet: Alphabet;
};

const Pool = React.memo((props: Props) => {
  const readHidePool = React.useCallback(
    () => localStorage.getItem('hidePool') === 'true',
    []
  );
  const [hidePool, setHidePool] = React.useState(readHidePool); // ok to not use useMountedState
  React.useEffect(() => {
    const interval = setInterval(() => {
      setHidePool(readHidePool);
    }, 1000); // how long should it be before it picks up changes from other tabs?
    return () => {
      clearInterval(interval);
    };
  }, [readHidePool]);

  return (
    <React.Fragment>
      <ActualPool {...props} hidePool={hidePool} />
    </React.Fragment>
  );
});

const ActualPool = React.memo((props: Props & { hidePool: boolean }) => {
  const letterOrder =
    PoolFormats.find((f) => f.poolFormatType === props.poolFormat)?.format(
      props.alphabet
    ) || 'ABCDEFGHIJKLMNOPQRSTUVWXYZ?';
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
  const dropDown = !props.hidePool && (
    <Dropdown
      overlay={poolMenu}
      trigger={['click']}
      placement="bottomRight"
      overlayClassName="format-dropdown"
    >
      <Link to="/" onClick={(e) => e.preventDefault()}>
        Rearrange
      </Link>
    </Dropdown>
  );

  const vowels = (a: Alphabet): string => {
    return a.letters.map((l) => (l.vowel ? l.rune : '')).join('');
  };

  const consonants = (a: Alphabet): string => {
    return a.letters.map((l) => (!l.vowel ? l.rune : '')).join('');
  };

  console.log('vowels', vowels(props.alphabet));

  const renderContents = (title?: string) =>
    (!props.hidePool || title) && (
      <div className="pool">
        {title ? <p className="label">{title}</p> : null}
        {!props.hidePool && (
          <React.Fragment>
            <div className="tiles-remaining">{letterSections}</div>
            <div className="vc-distribution">
              <div>
                {singularCount(
                  getPoolCount(pool, props.alphabet, vowels),
                  'vowel',
                  'vowels'
                )}
              </div>
              <div>
                {singularCount(
                  getPoolCount(pool, props.alphabet, consonants),
                  'consonant',
                  'consonants'
                )}
              </div>
            </div>
          </React.Fragment>
        )}
      </div>
    );

  const unseen = getPoolCount(pool, props.alphabet, (a: Alphabet) =>
    a.letters.map((l) => l.rune).join('')
  );
  const inbag = Math.max(unseen - 7, 0);

  let title: string;
  if (inbag === 0) {
    title = `Opponent has ${singularCount(unseen, 'tile', 'tiles')}`;
  } else {
    title = `${singularCount(inbag, 'tile', 'tiles')} in bag`;
  }

  if (props.omitCard) {
    return <>{renderContents(title)}</>;
  }
  return (
    <Card className="pool" id="pool" title={title} extra={dropDown}>
      {renderContents()}
    </Card>
  );
});

export default Pool;
