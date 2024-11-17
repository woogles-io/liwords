import React from 'react';
import { Link } from 'react-router-dom';
import { Card, Dropdown } from 'antd';
import { PoolFormatType, PoolFormats } from '../constants/pool_formats';
import { singularCount } from '../utils/plural';
import { Alphabet, machineLetterToRune } from '../constants/alphabets';
import { Blank, MachineLetter, MachineWord } from '../utils/cwgame/common';

type poolType = { [ml: MachineLetter]: number };

function poolMinusRack(pool: poolType, rack: MachineWord) {
  const poolCopy = { ...pool };
  for (let i = 0; i < rack.length; i += 1) {
    poolCopy[rack[i]] -= 1;
  }
  return poolCopy;
}

function renderLetters(
  pool: poolType,
  alphabet: Alphabet,
  possibleLetters: Array<MachineLetter>,
  sectionIndex: number,
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
      const rune = machineLetterToRune(letter, alphabet, false, true);
      for (let i = 0; i < pool[letter]; i++) {
        if (i % maxConsecutive) {
          letterGroup += rune;
        } else {
          letterGroup += ` ${rune}`;
        }
      }
      output.push(
        <React.Fragment key={`lg-${letter}-${possibility}`}>
          <span className="letter-group" data-multichar={rune.length > 1}>
            {letterGroup.trim()}
          </span>{' '}
        </React.Fragment>
      );
    }
  }
  return (
    <section className="pool-section" key={sectionIndex}>
      {output}
    </section>
  );
}

function getPoolCount(
  pool: poolType,
  alphabet: Alphabet,
  filterfn: (a: Alphabet) => Array<MachineLetter>
) {
  const letters = filterfn(alphabet);
  return letters.reduce((acc, cur) => {
    return acc + pool[cur];
  }, 0);
}

type Props = {
  omitCard?: boolean;
  pool: poolType;
  poolFormat: PoolFormatType;
  setPoolFormat: (format: PoolFormatType) => void;
  currentRack: MachineWord;
  alphabet: Alphabet;
};

const Pool = React.memo((props: Props) => {
  const readHidePool = React.useCallback(
    () => localStorage.getItem('hidePool') === 'true',
    []
  );
  const [hidePool, setHidePool] = React.useState(readHidePool);
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
  const letterOrder = PoolFormats.find(
    (f) => f.poolFormatType === props.poolFormat
  )?.format(props.alphabet) || [props.alphabet.letters.map((_, idx) => idx)];
  const pool = poolMinusRack(props.pool, props.currentRack);
  const letterSections = letterOrder.map((section, idx) => {
    return renderLetters(pool, props.alphabet, section, idx);
  });
  const poolMenuItems = PoolFormats.map((pf) => ({
    label: pf.displayName,
    key: pf.poolFormatType.toString(),
  }));

  const dropDown = !props.hidePool && (
    <Dropdown
      menu={{
        items: poolMenuItems,
        onClick: ({ key }) => {
          localStorage?.setItem('poolFormat', key);
          props.setPoolFormat(
            PoolFormats.find((p) => p.poolFormatType.toString() === key)
              ?.poolFormatType || PoolFormatType.Alphabet
          );
        },
      }}
      trigger={['click']}
      placement="bottomRight"
      overlayClassName="format-dropdown"
    >
      <Link to="/" onClick={(e) => e.preventDefault()}>
        Rearrange
      </Link>
    </Dropdown>
  );

  const vowels = (a: Alphabet): Array<MachineLetter> => {
    return a.letters.reduce((a, letter, idx) => {
      if (letter.vowel) {
        a.push(idx);
      }
      return a;
    }, new Array<MachineLetter>());
  };

  const consonants = (a: Alphabet): Array<MachineLetter> => {
    return a.letters.reduce((a, letter, idx) => {
      if (!letter.vowel && letter.rune !== Blank) {
        a.push(idx);
      }
      return a;
    }, new Array<MachineLetter>());
  };

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
    a.letters.map((l, idx) => idx)
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
