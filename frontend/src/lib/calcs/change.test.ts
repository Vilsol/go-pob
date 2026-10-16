import { describe, expect, it } from 'vitest';
import { changeOf } from './change';

describe('a Calcs cell between two calculations', () => {
  it('is better or worse where its stat says which way is better (Build.lua displayStats)', () => {
    expect(changeOf('5,123', '5,400', 'Life')).toBe('better');
    expect(changeOf('5,400', '5,123', 'Life')).toBe('worse');
    expect(changeOf('1.200s', '1.500s', 'Cooldown')).toBe('worse');
    expect(changeOf('1.500s', '1.200s', 'Cooldown')).toBe('better');
  });

  it('reads the number past PoB colour codes', () => {
    expect(changeOf('^x33FF7712%', '^x33FF7715%', 'Life')).toBe('better');
  });

  it('says only that it changed where no direction is known', () => {
    expect(changeOf('10', '12')).toBe('changed');
    expect(changeOf('10', '12', 'NotADisplayedStat')).toBe('changed');
    expect(changeOf('Yes', 'No', 'Life')).toBe('changed');
    expect(changeOf('10 (x2)', '10 (x3)', 'Life')).toBe('changed');
  });

  it('is nothing when the text is the same', () => {
    expect(changeOf('5,123', '5,123', 'Life')).toBeUndefined();
  });
});
