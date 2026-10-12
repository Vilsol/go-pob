import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { displayStats, minionDisplayStats, type Stat } from './stats';

type Entry = Record<string, string | boolean>;
const expected = JSON.parse(
  readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/display_stats_expected.json', import.meta.url)), 'utf8')
) as Entry[];

const STRINGS = ['stat', 'childStat', 'label', 'fmt', 'flag', 'notFlag', 'color', 'overCapStat'] as const;
const BOOLS = ['pc', 'mod', 'hideStat', 'warnColor', 'chaosInoc'] as const;
const FUNCS = ['condFunc', 'warnFunc', 'suffixCondFunc', 'overCapStatCondFunc'] as const;

const project = (s: Stat): Entry => {
  const out: Entry = {};
  for (const k of STRINGS) {
    if (s[k] !== undefined) out[k] = k === 'color' ? s[k].toUpperCase() : s[k];
  }
  if (s.suffix !== undefined) out.suffix = typeof s.suffix === 'function' ? '<function>' : s.suffix;
  for (const k of BOOLS) {
    if (s[k]) out[k] = true;
  }
  for (const k of FUNCS) {
    if (s[k]) out[k] = true;
  }
  return out;
};

describe('displayStats against PoB BuildDisplayStats.lua', () => {
  it('has the same entries, in order, with the same fields', () => {
    expect(expected.length).toBeGreaterThan(100);
    expect(displayStats.map(project)).toEqual(expected);
  });

  it('has the same minion entries, in order, with the same fields', () => {
    const expectedMinion = JSON.parse(
      readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/minion_display_stats_expected.json', import.meta.url)), 'utf8')
    ) as Entry[];
    expect(expectedMinion.length).toBeGreaterThan(20);
    expect(minionDisplayStats.map(project)).toEqual(expectedMinion);
  });
});
