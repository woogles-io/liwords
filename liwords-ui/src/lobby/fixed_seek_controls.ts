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

const cococlub = {
  lexicon: 'CSW19',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 17, // 15 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

export const fixedSettings: { [key: string]: settings } = {
  phillyvirtual,
  hcnj,
  cococlub,
};

// A temporary map of club redirects. Map internal tournament ID to slug:
export const clubRedirects: { [key: string]: string } = {
  channel275: '/club/channel275',
  phillyvirtual: '/club/phillyvirtual',
  madisonclub: '/club/madison',
  toucanet: '/club/toucanet',
  dallasbedford: '/club/dallasbedford',
  seattleclub: '/club/seattle',
  sfclub: '/club/sf',
  montrealscrabbleclub: '/club/montreal',
  vvsc: '/club/vvsc',
  OttawaClub: '/club/Ottawa',
  BrookfieldClub: '/club/Brookfield',
  RidgefieldClub: '/club/Ridgefield',
  HuaxiaScrabbleClub: '/club/Huaxia',
  WorkspaceScrabbleClub: '/club/Workspace',
  houstonclub: '/club/houston',
  pghscrabble: '/club/pgh',
  orlandoscrabble: '/club/orlando',
  CambridgeON: '/club/CambridgeON',
  delawareclub599: '/club/delawareclub599',
  scpnepal: '/club/scpnepal',
  uiscrabbleclub: '/club/ui',
  coloradosprings: '/club/coloradosprings',
  bridgewaterclub: '/club/bridgewater',
  cococlub: '/club/coco',
};
