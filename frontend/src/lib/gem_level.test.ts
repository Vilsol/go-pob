import { describe, expect, it } from 'vitest';

import { formatGemLevel, GEM_LEVEL_MODES, isGemLevelMode, parseGemLevel, type GemLevelSetting } from './gem_level';

describe('gem level is an enum-or-number, not a number', () => {
  it('round-trips every symbolic mode the pin defines', () => {
    for (const { value } of GEM_LEVEL_MODES) {
      const parsed = parseGemLevel(value);
      expect(parsed).toEqual({ kind: 'mode', mode: value });
      expect(formatGemLevel(parsed)).toBe(value);
    }
  });

  it('covers exactly the five modes the corpus contains', () => {
    expect(GEM_LEVEL_MODES.map((m) => m.value)).toEqual(['normalMaximum', 'corruptedMaximum', 'awakenedMaximum', 'characterLevel', 'levelOne']);
  });

  it('round-trips an explicit numeric level', () => {
    const parsed = parseGemLevel('20');
    expect(parsed).toEqual({ kind: 'level', level: 20 });
    expect(formatGemLevel(parsed)).toBe('20');
  });

  // SkillsTab.lua:91
  it('treats an absent or empty value as normalMaximum', () => {
    for (const raw of [undefined, null, '']) {
      expect(parseGemLevel(raw)).toEqual({ kind: 'mode', mode: 'normalMaximum' });
    }
  });

  it('never yields NaN, even for an unrecognised string', () => {
    const parsed = parseGemLevel('someFutureMode');
    expect(parsed.kind).toBe('mode');
    expect(formatGemLevel(parsed)).toBe('normalMaximum');
    expect(formatGemLevel(parsed)).not.toBe('NaN');
  });

  it('distinguishes a mode from a level in the type, not by convention', () => {
    const mode: GemLevelSetting = { kind: 'mode', mode: 'characterLevel' };
    const level: GemLevelSetting = { kind: 'level', level: 1 };
    expect(formatGemLevel(mode)).toBe('characterLevel');
    expect(formatGemLevel(level)).toBe('1');
    expect(isGemLevelMode('characterLevel')).toBe(true);
    expect(isGemLevelMode('1')).toBe(false);
  });

  it('a read-then-write with no edit preserves the stored value', () => {
    for (const stored of ['normalMaximum', 'characterLevel', 'corruptedMaximum', 'awakenedMaximum', 'levelOne', '20', '1']) {
      expect(formatGemLevel(parseGemLevel(stored))).toBe(stored);
    }
  });
});
