// Note: this is a TEMPORARY file. Once we add this ability to the tournament
// backend, we can remove this.

import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';

type settings = { [key: string]: string | number | boolean };

const phillyvirtual = {
  variant: 'classic',
  lexicon: 'NWL20',
  challengerule: ChallengeRule.VOID,
  initialtime: 22, // Slider position is equivalent to 20 minutes.
  rated: true,
  extratime: 2,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const cococlub = {
  variant: 'classic',
  lexicon: 'CSW19',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 17, // 15 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const laclub = {
  variant: 'classic',
  lexicon: 'NWL20',
  challengerule: ChallengeRule.DOUBLE,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 3,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const madisonclub = {
  variant: 'classic',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const cocoblitz = {
  variant: 'classic',
  lexicon: 'CSW19',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 5, // 3 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const channel275 = {
  variant: 'classic',
  lexicon: 'CSW19',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nssg = {
  variant: 'classic',
  lexicon: 'CSW19X',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 15, // 13 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nssg16 = {
  ...nssg,
  initialtime: 18, // 16 mins
};

const nssg19 = {
  ...nssg,
  initialtime: 21, // 19 mins
};

const phillyasap = {
  variant: 'classic',
  lexicon: 'NWL20',
  challengerule: ChallengeRule.VOID,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 2,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nyc = {
  variant: 'classic',
  lexicon: 'NWL20',
  challengerule: ChallengeRule.DOUBLE,
  initialtime: 19, // 17 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const learners = {
  variant: 'classic',
  initialtime: 12, // 10 minutes
  rated: true,
  extratime: 0,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nasscChampionship = {
  variant: 'classic',
  lexicon: 'NSWL20',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nasscNovice = {
  variant: 'classic',
  lexicon: 'NSWL20',
  challengerule: ChallengeRule.VOID,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nasscHighSchool = {
  variant: 'classic',
  lexicon: 'NSWL20',
  challengerule: ChallengeRule.DOUBLE,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const premiumswerve = {
  variant: 'wordsmog',
  lexicon: 'CSW19',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 24, // 22 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

export const fixedSettings: { [key: string]: settings } = {
  phillyvirtual,
  cococlub,
  madisonclub,
  eqRyXi3cBrUrDduuKDGuB9: cococlub,
  '26VtG4JCfeD6qvSGJEwRLm': laclub,
  cocoblitz,
  channel275,
  GqgfauAMzorWxGGrCqhV5J: phillyasap,
  cbCrE5EAnfTpacaZkxc4SZ: nssg, // /tournament
  CSLUwqH4rHUTKzNcp7cPRP: nssg, // /club
  K4MwE8nesdmPAQJkHbVpbi: nssg16,
  nEc9xsgU6h78eeKRA3MeCT: nssg19,
  CL9GW5sDfNqeX2yiPRg9YF: nyc,
  vhM3xsCFtxvM794dCK2mE6: nasscChampionship,
  Uzfx4iW2kLhyzWUz6MWQxY: nasscNovice,
  XZDoU8Z6fMk7WrVitthkeU: nasscHighSchool,
  KU8PSEu8p4Pni4qvmhn9x3: learners,
  rmuXRr9CPzpcwjF4vnYdaB: premiumswerve,
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

// Temporary teams for some tournaments. We should add support for this natively
// in the backend.

type teamSettings = {
  odds: string;
  evens: string;
};

export const teamTourneys: { [key: string]: teamSettings } = {
  '/tournament/vcanam': {
    odds: '🇨🇦',
    evens: '🇺🇸',
  },
};
