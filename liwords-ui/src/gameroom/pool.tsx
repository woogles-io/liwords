import React from 'react'
import { Card } from 'antd'
import { PoolFormatType, PoolFormats } from '../constants/pool_formats';


type poolType = { [rune: string]: number };

function poolMinusRack(pool: poolType, rack: string) {
  const poolCopy = { ...pool };
  for (let i = 0; i < rack.length; i += 1) {
    poolCopy[rack[i]] -= 1;
  }
  return poolCopy;
}


/* function poolGenerator(possibleLetters: string, pool: poolType) {
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
 */
function renderLetters(pool: poolType, possibleLetters: string, maxConsecutive: number = 6) {
  let output = '';
  for (let possibility = 0; possibility < possibleLetters.length; possibility += 1) {
    const letter = possibleLetters[possibility];
    if (pool[letter]) {
      for (let i = 0; i < pool[letter]; i ++ ) {
        if (i % maxConsecutive) {
          output += letter;
        } else {
          output += ` ${letter}`;
        }
      }
    }
  }
  return <section>{output}</section>;
}

function getPoolCount(pool: poolType) {
  return Object.keys(pool).reduce((acc, cur) => acc + pool[cur], 0);
}

type Props = {
  pool: poolType;
  poolFormat: PoolFormatType,
  setPoolFormat: (format: PoolFormatType) => void;
  currentRack: string;
};

const Pool = (props: Props) => {
  const letterOrder = PoolFormats.find(f => f.poolFormatType === props.poolFormat)?.format || 'ABCDEFGHIJKLMNOPQRSTUVWXYZ?';
  const pool = poolMinusRack(props.pool, props.currentRack);
  const letterSections = letterOrder.split(',')
    .map(letterSection => renderLetters(pool, letterSection));

  return (
    <Card className="pool">
      <header>
       {getPoolCount(pool)} tiles in the bag
      </header>
      <div className="tiles-remaining">
        {letterSections}
      </div>
    </Card>
  );
};

export default Pool;
