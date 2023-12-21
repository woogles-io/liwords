// eslint-disable-next-line @typescript-eslint/no-var-requires
const screenSizes = require('../../base.scss').default;

export type MachineLetter = number;
export type MachineWord = Array<MachineLetter>;

export type EphemeralTile = {
  // ephemeron <3 you are missed
  row: number;
  col: number;
  letter: MachineLetter;
};

// An EmptyRackSpaceMachineLetter is an empty space on a rack. We don't make it
// a 0 because that is a blank. 0x80 would be a "lowercase blank", which
// doesn't exist. This is a good "gap" letter.
export const EmptyRackSpaceMachineLetter: MachineLetter = 0x80;
export const EmptySpace = ' ';
export const EmptyBoardSpaceMachineLetter: MachineLetter = 0;
export const BlankMachineLetter: MachineLetter = 0;
export const Blank = '?';
// PlayedTiles is made for quick indexing of a recently placed tile.
export type PlayedTiles = { [tilecoords: string]: boolean };

// PlayerOfTiles maps to onturn. May continue to map challenged-off squares.
export type PlayerOfTiles = { [tilecoords: string]: number };

export enum Direction {
  Horizontal,
  Vertical,
}

export const isTouchDevice = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  if (/android/i.test(userAgent) || /iPad|iPhone|iPod/.test(userAgent)) {
    return true;
  }
  return !!('ontouchstart' in window);
};

export const isMac = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  return /Mac/i.test(userAgent);
};

export const isWindows = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  return /Win/i.test(userAgent);
};

export const uniqueTileIdx = (row: number, col: number): number => {
  // Just a unique number to identify a row,col coordinate.
  return row * 100 + col;
};

export const isBlank = (letter: string): boolean => {
  return letter.toLowerCase() === letter;
};

export const isDesignatedBlankMachineLetter = (ml: MachineLetter): boolean => {
  return (ml & 0x80) !== 0;
};

export const makeBlank = (ml: MachineLetter): MachineLetter => {
  return ml | 0x80;
};

export const getVW = () =>
  Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0);

export const isMobile = () =>
  getVW() < parseInt(screenSizes.screenSizeTablet, 10);

export const isTablet = () =>
  getVW() >= parseInt(screenSizes.screenSizeTablet, 10) &&
  getVW() < parseInt(screenSizes.screenSizeLaptop, 10);

export const isDesignatedBlank = (letter: string): boolean => {
  return letter.toLowerCase() === letter && letter.toUpperCase() !== letter;
};

// String.charAt implementation that handles surrogate pairs
// modified from:
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/charAt#Fixing_charAt()_to_support_non-Basic-Multilingual-Plane_(BMP)_characters
export const fixedCharAt = (
  string: string,
  startIndex: number,
  length: number
) => {
  const surrogatePairs = /[\uD800-\uDBFF][\uDC00-\uDFFF]/g;
  const end = string.length;
  let currentIndex = startIndex;
  let remainingChars = length;
  let ret = '';

  while (remainingChars > 0) {
    while (surrogatePairs.exec(string) != null) {
      const { lastIndex } = surrogatePairs;

      if (lastIndex - 2 < currentIndex) {
        currentIndex++;
      } else {
        break;
      }
    }

    if (currentIndex >= end || currentIndex < 0) {
      return ret;
    }

    ret += string.charAt(currentIndex);

    if (
      /[\uD800-\uDBFF]/.test(ret) &&
      /[\uDC00-\uDFFF]/.test(string.charAt(currentIndex + 1))
    ) {
      // Go one further, since one of the "characters" is part of a surrogate pair
      ret += string.charAt(++currentIndex);
    }

    currentIndex++;
    remainingChars--;
  }

  return ret;
};

export default fixedCharAt;
