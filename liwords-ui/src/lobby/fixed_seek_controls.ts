// Note: this is a TEMPORARY file. Once we add this ability to the tournament
// backend, we can remove this.

import { ChallengeRule } from '../gen/macondo/api/proto/macondo/macondo_pb';

type settings = { [key: string]: string | number | boolean };

const phillyvirtual = {
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
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const cocoblitz = {
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
  lexicon: 'NWL20',
  challengerule: ChallengeRule.DOUBLE,
  initialtime: 19, // 17 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const wandertypo = {
  challengerule: ChallengeRule.VOID,
  initialtime: 7, // 5 minutes
  rated: true,
  extratime: 5,
  friend: '',
  incOrOT: 'increment',
  vsBot: false,
};

const nasscChampionship = {
  lexicon: 'OSPD6',
  challengerule: ChallengeRule.FIVE_POINT,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nasscNovice = {
  lexicon: 'OSPD6',
  challengerule: ChallengeRule.VOID,
  initialtime: 22, // 20 minutes
  rated: true,
  extratime: 1,
  friend: '',
  incOrOT: 'overtime',
  vsBot: false,
};

const nasscHighSchool = {
  lexicon: 'OSPD6',
  challengerule: ChallengeRule.DOUBLE,
  initialtime: 22, // 20 minutes
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
  '26VtG4JCfeD6qvSGJEwRLm': laclub,
  cocoblitz,
  channel275,
  GqgfauAMzorWxGGrCqhV5J: phillyasap,
  cbCrE5EAnfTpacaZkxc4SZ: nssg, // /tournament
  CSLUwqH4rHUTKzNcp7cPRP: nssg, // /club
  K4MwE8nesdmPAQJkHbVpbi: nssg16,
  nEc9xsgU6h78eeKRA3MeCT: nssg19,
  CL9GW5sDfNqeX2yiPRg9YF: nyc,
  aGaSXfc4XDwUFBpkhZ9Jz3: wandertypo,
  vhM3xsCFtxvM794dCK2mE6: nasscChampionship,
  Uzfx4iW2kLhyzWUz6MWQxY: nasscNovice,
  XZDoU8Z6fMk7WrVitthkeU: nasscHighSchool,
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
