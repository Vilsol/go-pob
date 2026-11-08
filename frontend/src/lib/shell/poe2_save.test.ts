// Build.lua:1196-1244 (PoB-PoE2)

import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const serialize = vi.hoisted(() => vi.fn((stats?: unknown) => Promise.resolve(stats ? '<with stats/>' : '<without/>')));
const saveForeign = vi.hoisted(() => vi.fn((path: string, xml: string) => Promise.resolve(void [path, xml])));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: () => Promise.resolve({ Serialize: serialize }) }));
vi.mock('$lib/go/worker', () => ({ syncWrap: { SaveForeignBuild: saveForeign } }));

const { savedStats2, savePoe2Build } = await import('./poe2_save');
const { luaToString } = await import('$lib/display/saved_stats');
const { poe2Build } = await import('$lib/global');

// oracle/poe2_save_stats.lua over wasm2/exposition/testdata/sorceress_spark.xml on the PoB-PoE2 pin
const pob = JSON.parse(readFileSync(resolve(__dirname, 'testdata/sorceress_spark_save.json'), 'utf8')) as {
  output: Record<string, number>;
  skillFlags: Record<string, boolean>;
  rows: [string, string, string][];
};

describe('the stat rows a PoE2 save writes', () => {
  it("are PoB-PoE2's rows for its own output, in its order", () => {
    const s = savedStats2({ output: pob.output, skillFlags: pob.skillFlags });
    const ours = [...s.player.map((p) => ['PlayerStat', p.Stat, luaToString(p.Value)]), ...s.minion.map((p) => ['MinionStat', p.Stat, luaToString(p.Value)])];
    expect(pob.rows.length).toBeGreaterThan(100);
    expect(ours).toEqual(pob.rows);
    expect(s.fullDPS).toEqual([]);
  });

  it('list a flagged row only with its flag, and every minion row the minion has', () => {
    const output = { AverageHit: 1, AverageDamage: 2, PowerChargesMax: 3 };
    const without = savedStats2({ output, skillFlags: {} }).player.map((p) => p.Stat);
    expect(without).toEqual(['AverageHit', 'PowerChargesMax']);
    expect(savedStats2({ output, skillFlags: { attack: true } }).player.map((p) => p.Stat)).toEqual(['AverageHit', 'AverageDamage', 'PowerChargesMax']);
    const minion = savedStats2({ output: {}, skillFlags: {}, minion: { Speed: 1.5, Life: 9 } }).minion;
    expect(minion).toEqual([
      { Stat: 'Speed', Value: 1.5 },
      { Stat: 'Speed', Value: 1.5 },
      { Stat: 'Life', Value: 9 }
    ]);
  });

  it('are what Save hands the PoE2 worker for the open build', async () => {
    const view = { className: 'Sorceress', ascendClassName: 'None', level: 40, output: pob.output, skillFlags: pob.skillFlags };
    poe2Build.set(view);
    await savePoe2Build('/Spark');
    expect(serialize).toHaveBeenLastCalledWith(savedStats2(view));
    expect(saveForeign).toHaveBeenLastCalledWith('/Spark', '<with stats/>');
  });

  // oracle/poe2_full_dps.lua over the PoE2 fixture builds with every group included
  const fullDPS = readFileSync(resolve(__dirname, '../../../../testdata/oracle/poe2_full_dps.jsonl'), 'utf8')
    .split('\n')
    .filter(Boolean)
    .map(
      (l) =>
        JSON.parse(l) as {
          build: string;
          skills: { name: string; dps: number; count: number; trigger?: string; skillPart?: string; source?: string }[];
          saved: { stat: string; value: string; skillPart: string; source: string }[];
        }
    )
    .filter((r) => r.saved.length > 0);

  it.each(fullDPS.map((r) => [r.build, r] as const))("write PoB-PoE2's FullDPSSkill rows for %s", (_, r) => {
    const skillDPS = r.skills.map((s) => ({
      Name: s.name,
      DPS: s.dps,
      Count: s.count,
      Trigger: s.trigger ?? '',
      SkillPart: s.skillPart ?? '',
      Source: s.source ?? ''
    }));
    const rows = savedStats2({ output: {}, skillFlags: {}, skillDPS }).fullDPS;
    // the build serializer formats the value; it is graded as a number here
    expect(rows.map((d) => ({ stat: d.Stat, skillPart: d.SkillPart, source: d.Source }))).toEqual(
      r.saved.map((d) => ({ stat: d.stat, skillPart: d.skillPart, source: d.source }))
    );
    rows.forEach((d, i) => expect(Math.abs(d.Value - Number(r.saved[i].value))).toBeLessThanOrEqual(Math.abs(d.Value) * 1e-12));
  });
});
