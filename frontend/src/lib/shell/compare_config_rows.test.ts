// CompareTab.lua:2099-2124
import { describe, expect, it } from 'vitest';
import { configRows } from './compare';
import { configRelevanceQuery, configurations, isShowAllExcluded } from '$lib/display/configurations';
import type { calculator } from '$lib/types';

const none = Object.fromEntries(Object.keys(configRelevanceQuery).map((k) => [k, []])) as unknown as calculator.ConfigRelevance;
const withCond = (c: string) => ({ ...none, Conds: [c] }) as calculator.ConfigRelevance;

describe('the compare config rows', () => {
  it('lists the differences, then the equal options either build uses', () => {
    const a = { conditionMoving: true, enemyIsBoss: 'Uber' };
    const b = { conditionMoving: true };
    const rows = configRows(a, b, withCond('Moving'), none);
    const diff = rows.filter((r) => r.diff).map((r) => r.key);
    const common = rows.filter((r) => !r.diff).map((r) => r.key);
    expect(diff).toEqual(['enemyIsBoss']);
    expect(common).toContain('conditionMoving');
  });

  it('drops an equal gated option neither build uses', () => {
    const rows = configRows({ conditionMoving: true }, { conditionMoving: true }, none, none);
    expect(rows.map((r) => r.key)).not.toContain('conditionMoving');
  });

  it('keeps an equal option relevant to B only', () => {
    const rows = configRows({ conditionMoving: true }, { conditionMoving: true }, none, withCond('Moving'));
    expect(rows.find((r) => r.key === 'conditionMoving')?.diff).toBe(false);
  });

  // CompareTab.lua:923-933, 2115
  it('Show All adds the equal options neither build uses, except those Show All excludes', () => {
    const rows = configRows({ conditionMoving: true }, { conditionMoving: true }, none, none, true);
    expect(rows.find((r) => r.key === 'conditionMoving')?.diff).toBe(false);
    const excluded = configurations.flatMap((s) => s.variables).filter((v) => v.type !== 'text' && isShowAllExcluded(v));
    expect(excluded.length, 'vacuity: no option is excluded from Show All').toBeGreaterThan(0);
    expect(rows.map((r) => r.key)).not.toContain(excluded[0].var);
  });
});
