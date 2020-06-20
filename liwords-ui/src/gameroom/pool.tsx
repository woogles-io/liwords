import React from 'react';
import { Card } from 'antd';
import { PoolFormatType } from '../store/store'

const letterOrder = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ?';
type poolType = { [rune: string]: number };

/**
 * Generate a list of letters that are in the pool.
 * @param  {string} possibleLetters A string with the letters to look for
 * @param  {Object} pool A dictionary with pool counts.
 * @return {Array.<string>}
 */
function poolGenerator(possibleLetters: string, pool: poolType) {
  const poolArr = [];
  for (let i = 0; i < possibleLetters.length; i += 1) {
    const letter = possibleLetters[i];
    if (pool[letter]) {
      for (let n = 0; n < pool[letter]; n += 1) {
        poolArr.push(letter);
      }
    }
  }
  return poolArr;
}

function poolMinusRack(pool: poolType, rack: string) {
  const poolCopy = { ...pool };
  for (let i = 0; i < rack.length; i += 1) {
    poolCopy[rack[i]] -= 1;
  }
  return poolCopy;
}

type Props = {
  pool: poolType;
  poolFormat: PoolFormatType,
  currentRack: string;
};

const Pool = (props: Props) => {
  const pool = poolMinusRack(props.pool, props.currentRack);
  const letters = poolGenerator(letterOrder, pool);

  const renderPool = () => {
    return <>{letters.join(' ')}</>
  };

  return (
    <Card className="pool">
      <header>
        <p>
          {letters.length} tiles in the bag
        </p>
      </header>
      <div className="tiles-remaining">
          {renderPool()}
      </div>
    </Card>
  );
};

export default Pool;
