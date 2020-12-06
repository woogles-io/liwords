// Note: this is a TEMPORARY file. Once we add this ability to the tournament
// backend, we can remove this.

import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';

type settings = { [key: string]: string | number | boolean };

const phillyvirtual = {
  lexicon: 'NWL18',
  challengerule: ChallengeRule.VOID,
  initialtime: 22, // Slider position is equivalent to 20 minutes.
  rated: true,
  extratime: 2,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const hcnj = phillyvirtual;

export const fixedSettings: { [key: string]: settings } = {
  phillyvirtual,
  hcnj,
};
