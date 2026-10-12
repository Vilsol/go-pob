import { afterEach, describe, expect, it } from 'vitest';
import { buildStatList } from './stat_list';
import { displayStats } from './stats';
import { DEFAULT_NUMBER_FORMAT, numberFormat } from './number_format';

// Build.lua:1723
describe('Full DPS skill lines', () => {
  afterEach(() => numberFormat.set(DEFAULT_NUMBER_FORMAT));

  it('follow the compact-values option', () => {
    numberFormat.set({ ...DEFAULT_NUMBER_FORMAT, useCompactValues: true });
    const skill = { Name: 'Fireball', DPS: 1234567, Count: 1, Trigger: '', SkillPart: '', Source: '' };
    const stat = displayStats.find((s) => s.stat === 'SkillDPS');
    expect(stat).toBeDefined();
    const lines = buildStatList({ SkillDPS: [skill] }, [stat!]).flat();
    expect(lines[0]?.value).not.toContain('1,234,567');
  });
});
