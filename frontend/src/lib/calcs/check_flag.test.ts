import { expect, describe, it } from 'vitest';
import type { Outputs } from '$lib/custom_types';
import { calcSections } from './calc_sections';
import { checkFlag, rowsOf, sectionEnabled } from './check_flag';

const out = (o: Partial<Outputs>): Outputs => ({ SkillFlags: {}, Output: {}, OutputTable: {}, ...o });

describe('checkFlag', () => {
  it('passes an object with no gates', () => {
    expect(checkFlag({}, out({}))).toBe(true);
  });

  it('does not filter at all when there are no outputs yet', () => {
    expect(checkFlag({ flag: 'attack' }, undefined)).toBe(true);
  });

  // CalcsTab.lua:364-366
  it('requires flag to be set on the skill', () => {
    expect(checkFlag({ flag: 'attack' }, out({ SkillFlags: { attack: true } }))).toBe(true);
    expect(checkFlag({ flag: 'attack' }, out({ SkillFlags: {} }))).toBe(false);
    expect(checkFlag({ flag: 'attack' }, out({ SkillFlags: { attack: false } }))).toBe(false);
  });

  // CalcsTab.lua:367-373
  it('requires every member of flagList', () => {
    const flagList = ['weapon1Attack', 'effective'];
    expect(checkFlag({ flagList }, out({ SkillFlags: { weapon1Attack: true, effective: true } }))).toBe(true);
    expect(checkFlag({ flagList }, out({ SkillFlags: { weapon1Attack: true } }))).toBe(false);
    expect(checkFlag({ flagList }, out({ SkillFlags: {} }))).toBe(false);
  });

  // CalcsTab.lua:374-376
  it('requires playerFlag to be set', () => {
    expect(checkFlag({ playerFlag: 'multiPart' }, out({ SkillFlags: { multiPart: true } }))).toBe(true);
    expect(checkFlag({ playerFlag: 'multiPart' }, out({ SkillFlags: {} }))).toBe(false);
  });

  // CalcsTab.lua:377-379
  it('rejects when notFlag is set', () => {
    expect(checkFlag({ notFlag: 'attack' }, out({ SkillFlags: { attack: true } }))).toBe(false);
    expect(checkFlag({ notFlag: 'attack' }, out({ SkillFlags: {} }))).toBe(true);
  });

  // CalcsTab.lua:380-386
  it('rejects when ANY member of notFlagList is set', () => {
    const notFlagList = ['focused', 'hasOverride'];
    expect(checkFlag({ notFlagList }, out({ SkillFlags: {} }))).toBe(true);
    expect(checkFlag({ notFlagList }, out({ SkillFlags: { focused: true } }))).toBe(false);
    expect(checkFlag({ notFlagList }, out({ SkillFlags: { hasOverride: true } }))).toBe(false);
  });

  // CalcsTab.lua:387-396
  describe('haveOutput', () => {
    it('requires the output to be present and nonzero', () => {
      expect(checkFlag({ haveOutput: 'ImpaleChance' }, out({ Output: { ImpaleChance: 25 } }))).toBe(true);
      expect(checkFlag({ haveOutput: 'ImpaleChance' }, out({ Output: { ImpaleChance: 0 } }))).toBe(false);
      expect(checkFlag({ haveOutput: 'ImpaleChance' }, out({ Output: {} }))).toBe(false);
    });

    it('treats a false-valued output as absent, the way Lua `not x` does', () => {
      expect(checkFlag({ haveOutput: 'AnyAegis' }, out({ Output: { AnyAegis: false as unknown as number } }))).toBe(false);
      expect(checkFlag({ haveOutput: 'AnyAegis' }, out({ Output: { AnyAegis: true as unknown as number } }))).toBe(true);
    });

    it('resolves a namespaced output against the pass-local table', () => {
      const table = { OutputTable: { MainHand: { ImpaleChance: 40 } } };
      expect(checkFlag({ haveOutput: 'MainHand.ImpaleChance' }, out(table))).toBe(true);
      expect(checkFlag({ haveOutput: 'MainHand.Missing' }, out(table))).toBe(false);
      expect(checkFlag({ haveOutput: 'OffHand.ImpaleChance' }, out(table))).toBe(false);
      expect(checkFlag({ haveOutput: 'MainHand.ImpaleChance' }, out({ OutputTable: { MainHand: { ImpaleChance: 0 } } }))).toBe(false);
    });

    it('does not read a namespaced name out of the flat output table', () => {
      expect(checkFlag({ haveOutput: 'MainHand.ImpaleChance' }, out({ Output: { 'MainHand.ImpaleChance': 40 } }))).toBe(false);
    });
  });

  it('applies every gate on one object conjunctively', () => {
    const row = { flag: 'triggered', notFlag: 'focused', haveOutput: 'Cooldown' };
    expect(checkFlag(row, out({ SkillFlags: { triggered: true }, Output: { Cooldown: 2 } }))).toBe(true);
    expect(checkFlag(row, out({ SkillFlags: { triggered: true, focused: true }, Output: { Cooldown: 2 } }))).toBe(false);
    expect(checkFlag(row, out({ SkillFlags: { triggered: true }, Output: { Cooldown: 0 } }))).toBe(false);
  });
});

describe('sectionEnabled', () => {
  // CalcSectionControl.lua:17-18
  it('gates the section on the FIRST subsection data flag only', () => {
    const section = [3, 'X', 1, '#fff', [{ defaultCollapsed: false, label: 'a', data: { flag: 'hit', 0: { label: 'r' } } }]] as never;
    expect(sectionEnabled(section, out({ SkillFlags: { hit: true } }))).toBe(true);
    expect(sectionEnabled(section, out({ SkillFlags: {} }))).toBe(false);
  });

  it('ignores a flag on a LATER subsection, as PoB does', () => {
    const section = [
      3,
      'X',
      1,
      '#fff',
      [
        { defaultCollapsed: false, label: 'a', data: { 0: { label: 'r' } } },
        { defaultCollapsed: false, label: 'b', data: { flag: 'nonesuch', 0: { label: 's' } } }
      ]
    ] as never;
    expect(sectionEnabled(section, out({ SkillFlags: {} }))).toBe(true);
  });

  // CalcSectionControl.lua:94-103
  it('hides a section whose every row is gated out', () => {
    const section = [
      3,
      'X',
      1,
      '#fff',
      [{ defaultCollapsed: false, label: 'a', data: { 0: { label: 'r', flag: 'attack' }, 1: { label: 's', flag: 'attack' } } }]
    ] as never;
    expect(sectionEnabled(section, out({ SkillFlags: {} }))).toBe(false);
    expect(sectionEnabled(section, out({ SkillFlags: { attack: true } }))).toBe(true);
  });

  it('keeps a section alive on a row in a later subsection', () => {
    const section = [
      3,
      'X',
      1,
      '#fff',
      [
        { defaultCollapsed: false, label: 'a', data: { 0: { label: 'r', flag: 'attack' } } },
        { defaultCollapsed: false, label: 'b', data: { 0: { label: 's' } } }
      ]
    ] as never;
    expect(sectionEnabled(section, out({ SkillFlags: {} }))).toBe(true);
  });
});

describe('against the real transcribed section data', () => {
  it('rowsOf skips the non-numeric keys of a data block', () => {
    const rows = rowsOf({ extra: '{output:X}', flag: 'hit', colWidth: 95, 0: { label: 'a' }, 1: { label: 'b' } });
    expect(rows).toHaveLength(2);
    expect(rows.map((r) => r.label)).toEqual(['a', 'b']);
  });

  it('hides the weapon rows of Skill Hit Damage on a spell build', () => {
    const hit = calcSections.find((s) => s[1] === 'HitDamage');
    expect(hit).toBeDefined();
    const spell = out({ SkillFlags: { hit: true, spell: true, effective: true } });
    const labels = rowsOf(hit![4][0].data)
      .filter((r) => checkFlag(r, spell))
      .map((r) => r.label);

    expect(labels).toContain('Total Increased');
    expect(labels).not.toContain('MH Total Increased');
    expect(labels).not.toContain('OH Total Increased');
    expect(labels).not.toContain('MH Average Hit');
  });

  it('shows the main-hand rows and hides the off-hand rows on a one-handed attack build', () => {
    const hit = calcSections.find((s) => s[1] === 'HitDamage');
    const attack = out({ SkillFlags: { hit: true, attack: true, weapon1Attack: true, effective: true } });
    const labels = rowsOf(hit![4][0].data)
      .filter((r) => checkFlag(r, attack))
      .map((r) => r.label);

    expect(labels).toContain('MH Total Increased');
    expect(labels).not.toContain('OH Total Increased');
    expect(labels).not.toContain('Total Increased');
  });

  it('gates out whole sections that a spell build cannot have', () => {
    const spell = out({ SkillFlags: { hit: true, spell: true }, Output: {}, OutputTable: {} });
    const shown = calcSections.filter((s) => sectionEnabled(s, spell)).map((s) => s[1]);
    expect(shown.length).toBeGreaterThan(0);
    expect(shown.length).toBeLessThan(calcSections.length);
  });

  it('shows every section when there are no outputs, so a loading page is not blank', () => {
    const shown = calcSections.filter((s) => sectionEnabled(s, undefined));
    expect(shown).toHaveLength(calcSections.length);
  });

  // CalcSectionControl.lua:86
  it('leaves descriptor-level flags inert', () => {
    const impale = calcSections
      .flatMap((s) => s[4])
      .flatMap((b) => rowsOf(b.data))
      .find((r) => r.label === 'Impale Chance');
    expect(impale).toBeDefined();

    const descriptor = (impale![0] as Record<number, { flag?: string }>)[0];
    expect(descriptor.flag).toBe('impale');

    expect(checkFlag(impale!, out({ SkillFlags: { spell: true }, Output: { ImpaleChance: 10 } }))).toBe(true);
  });
});
