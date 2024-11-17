import React from 'react';

import { Select, Form } from 'antd';
import { AllLexica } from './lexica';
import { DisplayFlag } from './display_flag';

type Props = {
  excludedLexica?: Set<string>;
  disabled?: boolean;
  hideRequired?: boolean;
  additionalLexica?: Array<string>;
};

export const MatchLexiconDisplay = (props: {
  lexiconCode: string;
  useShortDescription?: boolean;
}) => {
  const lex = AllLexica[props.lexiconCode];
  if (!lex) {
    // For unsupported lexica just return the old lexicon code.
    return <>{props.lexiconCode}</>;
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

export const lexiconOrder = [
  'CSW21',
  'CSW24',
  'NWL23',
  'ECWL',
  'RD28',
  'FRA24',
  'FILE2017',
  'NSF23',
  'NSWL20',
  'DISC2',
  'OSPS49',
];

export const puzzleLexica = ['CSW21', 'NWL23', 'RD28', 'FRA24'];

export const LexiconFormItem = React.memo((props: Props) => {
  const options = lexiconOrder
    .filter((k) => !props.excludedLexica?.has(k))
    .map((k) => (
      <Select.Option key={k} value={k}>
        <MatchLexiconDisplay lexiconCode={k} useShortDescription />
      </Select.Option>
    ));

  props.additionalLexica?.forEach((lex) => {
    options.push(
      <Select.Option key={lex} value={lex}>
        <MatchLexiconDisplay lexiconCode={lex} useShortDescription />
      </Select.Option>
    );
  });

  return (
    <Form.Item
      label="Dictionary"
      name="lexicon"
      rules={[
        {
          required: props.hideRequired ? false : true,
        },
      ]}
    >
      {/* i don't know why this z-index is 1100 (see app.scss). reset here, figure out later */}
      <Select disabled={props.disabled} listHeight={300} style={{ zIndex: 0 }}>
        {options}
      </Select>
    </Form.Item>
  );
});

export const excludedLexica = (
  enableAllLexicons: boolean,
  enableCSW19X: boolean
): Set<string> => {
  if (!enableAllLexicons) {
    return new Set<string>(['NSWL20', 'ECWL', 'CSW19X']);
  } else if (!enableCSW19X) {
    return new Set<string>(['CSW19X']);
  }
  return new Set<string>();
};
