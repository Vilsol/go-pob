import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { calcSections2 } from '$lib/calcs/calc_sections2';
import type { CalcSectionColumn2, CalcSectionRow2 } from '$lib/calcs/calc_section_types2';
import { calcsModProps2, calcsView2, checkFlag2, formatCalcStr2, modSectionData2, type CalcsActor2 } from './calcs2';
import { cellId, elementsOf } from './calcs';

// PoB-PoE2's Calcs tab over the wasm2 fixture builds' CALCS calculation (oracle/poe2_calcs.lua, `mise run calc-sections-poe2`)
type Oracle = CalcsActor2 & {
  entries: Record<string, unknown>;
  cells: Record<string, string>;
  extras: Record<string, string>;
  enabled: Record<string, boolean>;
};
const load = (name: string) =>
  JSON.parse(readFileSync(fileURLToPath(new URL(`../../../../testdata/oracle/poe2_calcs_${name}.json`, import.meta.url)), 'utf8')) as Oracle;

const byId = (a: string, b: string) => {
  const pa = a.split(':');
  const pb = b.split(':');
  if (pa[0] !== pb[0]) return 0;
  for (let i = 1; i < pa.length; i++) if (pa[i] !== pb[i]) return Number(pa[i]) - Number(pb[i]);
  return 0;
};

describe.each(['sorceress_spark', 'warrior_dual_mace'])('the PoE2 Calcs page: %s', (name) => {
  const oracle = load(name);
  const actor: CalcsActor2 = { output: oracle.output, skillFlags: oracle.skillFlags, skillData: oracle.skillData, mods: oracle.mods };

  it("reads PoB-PoE2's mod entries", () => {
    expect(Object.keys(calcsModProps2(calcSections2)).sort()).toEqual(Object.keys(oracle.entries).sort());
  });

  it("shows PoB-PoE2's cells with PoB-PoE2's text", () => {
    const cells: Record<string, string> = {};
    for (const [, id, , , subs] of calcSections2) {
      const first = subs[0];
      if (!checkFlag2({ flag: first.data.flag, notFlag: first.data.notFlag }, actor)) continue;
      subs.forEach((sub, j) =>
        elementsOf<CalcSectionRow2>(sub.data).forEach((row, k) => {
          if (!checkFlag2(row, actor)) return;
          elementsOf<CalcSectionColumn2>(row).forEach((col, l) => {
            if (col.format !== undefined && checkFlag2(col, actor)) cells[cellId(id, j, k, l)] = formatCalcStr2(col.format, actor, col, cellId(id, j, k, l));
          });
        })
      );
    }
    expect(Object.keys(cells).length).toBeGreaterThan(100);
    expect(cells).toEqual(oracle.cells);
  });

  it("lays out PoB-PoE2's sections, extras and cells in order", () => {
    const view = calcsView2(calcSections2, actor);
    expect(Object.fromEntries(view.sections.map((s) => [s.id, s.applicable]))).toEqual(oracle.enabled);
    for (const s of view.sections.filter((v) => v.applicable)) {
      const want = Object.keys(oracle.cells)
        .filter((k) => k.startsWith(`${s.id}:`))
        .sort(byId)
        .map((k) => oracle.cells[k])
        .filter((t) => t !== '');
      const got = (s.blocks ?? []).flatMap((b) => b.rows.flatMap((r) => r.cells.map((c) => c.text))).filter((t) => t !== '');
      expect(got, s.id).toEqual(want);
      const extras = Object.keys(oracle.extras)
        .filter((k) => k.startsWith(`${s.id}:`))
        .sort(byId)
        .map((k) => oracle.extras[k]);
      expect(
        (s.blocks ?? []).map((b) => b.extra).filter((e) => e !== undefined),
        s.id
      ).toEqual(extras);
    }
  });

  it('hides a flag-gated section when the main skill lacks the flag', () => {
    const view = calcsView2(calcSections2, { ...actor, skillFlags: { ...actor.skillFlags, hit: false } });
    expect(view.sections.find((s) => s.id === 'HitDamage')?.applicable).toBe(false);
  });
});

// CalcSections.lua:776-777, 1575 (PoB-PoE2): a modName table holding modType or cfg among its names; AddModSection
// unpacks its sequence and reads modType and cfg from the entry, which has neither. The keys are the mod tables
// oracle/poe2_mod_sections.lua records for the same entries.
describe('a Calcs entry whose modName table holds keys', () => {
  const recorded = Object.keys(
    JSON.parse(readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/poe2_mod_sections_sorceress_spark.json', import.meta.url)), 'utf8')) as object
  );
  const entries = calcSections2.flatMap(([, , , , subs]) =>
    subs.flatMap((sub) =>
      elementsOf<CalcSectionRow2>(sub.data).flatMap((row) =>
        elementsOf<CalcSectionColumn2>(row).flatMap((col) =>
          elementsOf<import('$lib/calcs/calc_section_types2').CalcSectionEntry2>(col).filter(
            (e) => e.modName && typeof e.modName === 'object' && !Array.isArray(e.modName)
          )
        )
      )
    )
  );

  it('sends its names and the entry own modType and cfg, as PoB-PoE2 shows them', () => {
    const sds = entries.map(modSectionData2);
    expect(sds.map((sd) => sd.ModName)).toEqual([['CrossbowBoltCount'], ['ChanceToNotConsumeAmmo'], ['CullPercent', 'CriticalCullPercent']]);
    for (const sd of sds) {
      expect(sd.Multi).toBe(true);
      expect(recorded).toContain(`${sd.ModName.join(',')}|${sd.ModType}|${sd.Cfg}|${sd.ModSource}|${sd.Enemy}|${sd.Actor}`);
    }
  });
});
