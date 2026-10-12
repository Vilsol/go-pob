import { describe, expect, it } from 'vitest';
import { compareMetrics, configDiffs, metricsOf, skillsFixture, summaryStats } from './compare';
import type { CompareSideData } from './compare_types';
import type { exposition } from '$lib/types';
import { configurations } from '$lib/display/configurations';

describe('summaryStats (CompareTab:DrawStatList)', () => {
  const side = (
    Output: Record<string, number>,
    SkillFlags: Record<string, boolean> = {},
    Minion?: { Output: Record<string, number>; SkillFlags: Record<string, boolean> }
  ) => ({
    Output,
    SkillFlags,
    Minion
  });

  it('shows a stat either build has, the other side 0, and both sides carry the same rows', () => {
    const { a, b } = summaryStats(side({ Life: 1000 }), side({ Life: 0, EnergyShield: 500 }));
    const keys = (g: { key: string }[][]) => g.flat().map((s) => s.key);
    expect(keys(a)).toEqual(keys(b));
    const es = b.flat().find((s) => s.label === 'Energy Shield')!;
    expect(es.text).toBe('500');
    expect(a.flat().find((s) => s.key === es.key)!.text).toBe('0');
    expect(a.flat().some((s) => s.label === 'Total Life')).toBe(true);
  });

  // CompareTab.lua:3328-3341
  it("carries PoB's absolute difference, formatted as the stat is", () => {
    const { a, b } = summaryStats(side({ Life: 1000, EnergyShield: 0 }), side({ Life: 1250, EnergyShield: 500 }));
    const of = (g: { label: string; diff?: string }[][], label: string) => g.flat().find((s) => s.label === label)?.diff;
    expect(of(a, 'Total Life')).toBe('+250 (+25.0%)');
    expect(of(b, 'Total Life')).toBe('+250 (+25.0%)');
    expect(of(a, 'Energy Shield')).toBe('+500');
    const regen = summaryStats(side({ TotalNetRegen: 10 }), side({ TotalNetRegen: 12.5 }));
    expect(of(regen.a, 'Total Net Recovery')).toBe('+2.5 (+25.0%)');
    const same = summaryStats(side({ Life: 1000 }), side({ Life: 1000 }));
    expect(of(same.a, 'Total Life')).toBe('');
  });

  it('admits a flag-gated stat when either build has the flag', () => {
    const gated = (flags: Record<string, boolean>) => summaryStats(side({ AverageDamage: 10 }, flags), side({ AverageDamage: 20 })).a.flat();
    expect(gated({ attack: true }).some((s) => s.key.includes('AverageDamage'))).toBe(true);
    expect(gated({}).some((s) => s.key.includes('AverageDamage'))).toBe(false);
  });

  it('switches to the minion list when either main skill is a minion skill', () => {
    const { a } = summaryStats(side({ Life: 100 }), side({ Life: 50 }, {}, { Output: { Life: 3000 }, SkillFlags: {} }));
    expect(a.flat().every((s) => s.key.startsWith('Minion:'))).toBe(true);
  });
});

describe('configDiffs (NormalizeConfigVals, the compare sections)', () => {
  const check = configurations.flatMap((s) => s.variables).find((v) => v.type === 'check')!;
  const count = configurations.flatMap((s) => s.variables).find((v) => v.type === 'count')!;

  it('an unset check is false and an unset count 0', () => {
    expect(configDiffs({ [check.var]: false }, {})).toEqual([]);
    expect(configDiffs({ [count.var]: 0 }, {})).toEqual([]);
    expect(configDiffs({ [check.var]: true }, {}).map((d) => [d.key, d.a, d.b])).toEqual([[check.var, 'Yes', '(not set)']]);
  });

  it('skips text options and the custom / map modifier sections', () => {
    const skipped = configurations
      .filter((s) => s.name === 'Map Modifiers and Player Debuffs')
      .flatMap((s) => s.variables)
      .find((v) => v.type === 'check');
    expect(skipped).toBeDefined();
    expect(configDiffs({ [skipped!.var]: true }, {})).toEqual([]);
  });
});

describe('metricsOf (powerStatList.GetFromOutput)', () => {
  it('Full DPS falls back to the player and minion Combined DPS when the build has none', () => {
    expect(metricsOf({ Output: { CombinedDPS: 10 }, Minion: { Output: { CombinedDPS: 5 } } }).dps).toBe(15);
    expect(metricsOf({ Output: { FullDPS: 7, CombinedDPS: 10 } }).dps).toBe(7);
  });
});

// CompareTab.lua:968-974
describe('compareMetrics', () => {
  it('leads with dps / ehp / life and adds the other power stats by their output key', () => {
    const list = compareMetrics([
      { Stat: 'FullDPS', Label: 'Full DPS' },
      { Stat: 'Mana', Label: 'Mana' },
      { Stat: 'MinionLife', Label: 'Minion Life' },
      { Stat: 'Life', Label: 'Life' }
    ]);
    expect(list.map((m) => m.id)).toEqual(['dps', 'ehp', 'life', 'Mana', 'MinionLife']);
    expect(metricsOf({ Output: { Mana: 40, Life: 9 }, Minion: { Output: { Life: 7 } } }, list)).toMatchObject({ life: 9, Mana: 40, MinionLife: 7 });
  });
});

// CompareTab.lua:4032-4071, 4186-4198
describe('skillsFixture', () => {
  it("appends the group's imbued support as a gem", () => {
    const gem = (ID: string, name: string, Support: boolean) => ({ ID, Support, GemType: 'INT', Base: { Name: name } }) as unknown as exposition.SkillGem;
    const gems = new Map([gem('a', 'Arc', false), gem('s', 'More Duration Support', true)].map((g) => [g.ID, g]));
    const group = { label: 'Arc', slot: '', enabled: true, inFull: true, count: 1, imbued: 'More Duration' };
    const side = {
      skills: {
        activeSet: 0,
        sets: [{ title: 'x', groups: [{ ...group, gems: [{ gemId: 'a', nameSpec: 'Arc', skillId: 'Arc', level: 20, quality: 0, enabled: true, count: 1 }] }] }]
      }
    };
    const fx = skillsFixture(side as unknown as CompareSideData, gems);
    expect(fx.sets[0].groups[0].gems.map((m) => [m.name, m.level, m.quality, m.support])).toEqual([
      ['Arc', 20, 0, false],
      ['More Duration Support', 1, 0, true]
    ]);
  });
});
