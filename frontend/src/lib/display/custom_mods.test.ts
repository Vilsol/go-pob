import { expect, describe, it } from 'vitest';
import { cellsToText, textToCells } from './custom_mods';
import {
  configRelevanceQuery,
  configurations,
  defaultValue,
  isModified,
  isRelevant,
  isShowAllExcluded,
  relevanceSets,
  reverseConfigOptions
} from './configurations';

const noRelevance = relevanceSets(Object.fromEntries(Object.keys(configRelevanceQuery).map((k) => [k, []])) as never)!;

describe('the option list is the pin', () => {
  it('has PoB’s sections in PoB’s order, Custom Modifiers last', () => {
    expect(configurations.map((s) => s.name)).toEqual([
      'General',
      'Skill Options',
      'Map Modifiers and Player Debuffs',
      'When In Combat',
      'For Effective DPS',
      'Enemy Stats',
      'Custom Modifiers'
    ]);
  });

  it('holds custom mods as groups, not as an option (ConfigTab.lua:872-884 migrates customMods away)', () => {
    expect(reverseConfigOptions.customMods).toBeUndefined();
    expect(configurations.at(-1)?.variables).toHaveLength(0);
  });
});

describe('defaults (ConfigTab.lua:650-660)', () => {
  it('uses defaultIndex for a list, not its first entry', () => {
    expect(defaultValue(reverseConfigOptions.resistancePenalty)).toBe(-60);
    expect(defaultValue(reverseConfigOptions.enemyIsBoss)).toBe('Pinnacle');
    expect(defaultValue(reverseConfigOptions.bandit)).toBe('None');
  });

  it('marks a value modified only when set and different from the default', () => {
    const v = reverseConfigOptions.resistancePenalty;
    expect(isModified(v, {}, {})).toBe(false);
    expect(isModified(v, { resistancePenalty: -60 }, {})).toBe(false);
    expect(isModified(v, { resistancePenalty: 0 }, {})).toBe(true);
  });

  it('lets a placeholder be the default (GetDefaultState)', () => {
    const v = reverseConfigOptions.enemyLightningResist;
    expect(isModified(v, { enemyLightningResist: 30 }, { enemyLightningResist: 30 })).toBe(false);
    expect(isModified(v, { enemyLightningResist: 30 }, { enemyLightningResist: 0 })).toBe(true);
  });
});

describe('visibility (ConfigTab.lua:350-630)', () => {
  const conditionMoving = reverseConfigOptions.conditionMoving;

  it('hides a condition option the build does not use, shows it when it does', () => {
    expect(isRelevant(conditionMoving, {}, noRelevance)).toBe(false);
    expect(isRelevant(conditionMoving, {}, { ...noRelevance, Conds: new Set(['Moving']) })).toBe(true);
  });

  it('shows ungated options always', () => {
    expect(isRelevant(reverseConfigOptions.enemyIsBoss, {}, noRelevance)).toBe(true);
  });

  it('gates ifOption on the other option’s value', () => {
    const gated = configurations.flatMap((s) => s.variables).find((v) => typeof v.ifOption === 'string' && !v.ifCond && !v.ifSkill && !v.ifFlag && !v.ifMult)!;
    expect(gated).toBeDefined();
    const on = { [gated.ifOption as string]: true };
    expect(isRelevant(gated, {}, noRelevance)).toBe(false);
    expect(isRelevant(gated, on, noRelevance)).toBe(true);
  });

  it('asks Go about every name a predicate tests', () => {
    expect(configRelevanceQuery.Conds).toContain('Moving');
    expect(configRelevanceQuery.TransfiguredSkills?.length).toBeGreaterThan(0);
    expect(configRelevanceQuery.Flags).toContain('totem');
  });

  it('keeps ifSkill and recently options out of Show All (ConfigVisibility.lua:144-155)', () => {
    const skillOption = configurations.flatMap((s) => s.variables).find((v) => v.ifSkill)!;
    expect(isShowAllExcluded(skillOption)).toBe(true);
    expect(isShowAllExcluded(conditionMoving)).toBe(false);
  });
});

describe('custom mod text as cells', () => {
  const saved = '\n\t\t\t\t#Kingmaker\nNearby Allies have Culling Strike\n\n+10 to Strength\n\t\t\t';

  it('round-trips PoB-saved text byte for byte', () => {
    for (const text of [saved, '', '\n\t\t', 'one line', 'a\nb']) {
      expect(cellsToText(textToCells(text))).toBe(text);
    }
  });

  it('keeps the XML indentation out of the cells and interior blank lines in', () => {
    expect(textToCells(saved).cells).toEqual(['\t\t\t\t#Kingmaker', 'Nearby Allies have Culling Strike', '', '+10 to Strength']);
  });

  it('changes only the edited line', () => {
    const t = textToCells(saved);
    t.cells[1] = 'Nearby Allies have Onslaught';
    expect(cellsToText(t)).toBe(saved.replace('Culling Strike', 'Onslaught'));
  });

  it('writes typed lines into an empty group plainly', () => {
    const t = textToCells('');
    t.cells = ['+1 to Strength', '+2 to Dexterity'];
    expect(cellsToText(t)).toBe('+1 to Strength\n+2 to Dexterity');
  });
});
