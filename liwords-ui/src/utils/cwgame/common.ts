import variables from "../../base.module.scss";
const { screenSizeLaptop, screenSizeTablet } = variables;

// PlayedTiles is made for quick indexing of a recently placed tile.
export type PlayedTiles = { [tilecoords: string]: boolean };

// PlayerOfTiles maps to onturn. May continue to map challenged-off squares.
export type PlayerOfTiles = { [tilecoords: string]: number };

export const isTouchDevice = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  if (/android/i.test(userAgent) || /iPad|iPhone|iPod/.test(userAgent)) {
    return true;
  }
  return !!("ontouchstart" in window);
};

export const isMac = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  return /Mac/i.test(userAgent);
};

export const isWindows = () => {
  const userAgent = navigator.userAgent || navigator.vendor;
  return /Win/i.test(userAgent);
};

export const isBlank = (letter: string): boolean => {
  return letter.toLowerCase() === letter;
};

export const getVW = () =>
  Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0);

export const isMobile = () => getVW() < parseInt(screenSizeTablet, 10);

export const isTablet = () =>
  getVW() >= parseInt(screenSizeTablet, 10) &&
  getVW() < parseInt(screenSizeLaptop, 10);

export const isDesignatedBlank = (letter: string): boolean => {
  return letter.toLowerCase() === letter && letter.toUpperCase() !== letter;
};

// String.charAt implementation that handles surrogate pairs
// modified from:
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/charAt#Fixing_charAt()_to_support_non-Basic-Multilingual-Plane_(BMP)_characters
export const fixedCharAt = (
  string: string,
  startIndex: number,
  length: number,
) => {
  const surrogatePairs = /[\uD800-\uDBFF][\uDC00-\uDFFF]/g;
  const end = string.length;
  let currentIndex = startIndex;
  let remainingChars = length;
  let ret = "";

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

export function typedKeys<T extends object>(obj: T): (keyof T)[] {
  return Object.keys(obj) as (keyof T)[];
}

export default fixedCharAt;
