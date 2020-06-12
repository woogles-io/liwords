import React from 'react';
import { Col, Row, Card } from 'antd';

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
  currentRack: string;
};

const Pool = (props: Props) => {
  const pool = poolMinusRack(props.pool, props.currentRack);
  const letters = poolGenerator(letterOrder, pool);

  return (
    <Card>
      <Row>
        <Col span={24}>{letters.length} unseen tiles:</Col>
      </Row>

      <Row>
        <Col span={24}>
          <span style={{ fontFamily: 'monospace' }}>
            <big>{letters.join(' ')}</big>
          </span>
        </Col>
      </Row>
    </Card>
  );
};

export default Pool;
