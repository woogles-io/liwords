export type PlayerOrder = 'p0' | 'p1';

// See cutoffs in variants.go. XXX: Try to tie these together better.
export const timeCtrlToDisplayName = (secs: number) => {
  if (secs <= 2 * 60) {
    return ['Ultra-Blitz!', 'magenta'];
  }
  if (secs <= 6 * 60) {
    return ['Blitz', 'volcano'];
  }
  if (secs <= 14 * 60) {
    return ['Rapid', 'gold'];
  }
  return ['Regular', 'blue'];
};
