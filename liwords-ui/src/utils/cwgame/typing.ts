import { Alphabet } from "../../constants/alphabets";
import { Blank } from "./common";

// Whether a character typed is one of the alphabet's tiles, in either case:
// in a Norwegian game "å" and "Å" are, in an English game neither is. The
// blank's "?" is not (it is typed for what it is elsewhere), nor is any key
// name ("Dead", "Enter") or digit.
export const isTileKey = (key: string, alphabet: Alphabet) => {
  if (!key || key === Blank) return false;
  const upper = key.toUpperCase();
  return alphabet.letterMap[upper] !== undefined && upper !== Blank;
};

// Whether a key pressed with Alt/Opt made a tile of its own: a letter beyond
// a-z that is one of the alphabet's tiles (Opt+a's "å", AltGr+a's "ą"). Where
// Alt makes no character (Windows, Linux), Alt+a is still "a": that stays a
// shortcut's, as it always was, not a tile's.
export const isAltTileKey = (key: string, alphabet: Alphabet) =>
  !/^[a-z]$/i.test(key) && isTileKey(key, alphabet);
