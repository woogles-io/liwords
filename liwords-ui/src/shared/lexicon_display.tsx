import React from 'react';

import { Select, Form } from 'antd';
import { AllLexica } from './lexica';
import { DisplayFlag } from './display_flag';

type Props = {
  excludedLexica?: Set<string>;
  disabled?: boolean;
};

export const MatchLexiconDisplay = (props: {
  lexiconCode: string;
  useShortDescription?: boolean;
}) => {
  const lex = AllLexica[props.lexiconCode];
  if (!lex) {
    return null;
  }
  const desc = (
    <>
      {props.useShortDescription ? lex.shortDescription : lex.matchName}
      {lex.flagCode && (
        <>
          {' '}
          <DisplayFlag countryCode={lex.flagCode} />
        </>
      )}
    </>
  );

  return desc;
};

export const LexiconFormItem = React.memo((props: Props) => {
  const order = [
    'CSW19',
    'NWL20',
    'ECWL',
    'RD28',
    'NSF21',
    'NWL18',
    'NSWL20',
    'CSW19X',
  ];

  const options = order
    .filter((k) => !props.excludedLexica?.has(k))
    .map((k) => (
      <Select.Option key={k} value={k}>
        <MatchLexiconDisplay lexiconCode={k} useShortDescription />
      </Select.Option>
    ));
  return (
    <Form.Item
      label="Dictionary"
      name="lexicon"
      rules={[
        {
          required: true,
        },
      ]}
    >
      <Select disabled={props.disabled}>{options}</Select>
    </Form.Item>
  );
});

export const excludedLexica = (
  enableAllLexicons: boolean,
  enableCSW19X: boolean
): Set<string> => {
  if (!enableAllLexicons) {
    return new Set<string>(['NWL18', 'NSWL20', 'ECWL', 'CSW19X']);
  } else if (!enableCSW19X) {
    return new Set<string>(['CSW19X']);
  }
  return new Set<string>();
};
