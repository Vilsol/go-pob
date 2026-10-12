// CalcBreakdownControl.lua:174-209
import { describe, expect, it } from 'vitest';
import { breakdownOf } from './calcs';
import type { calculator } from '$lib/types';

describe('breakdown tables', () => {
  // CalcBreakdown.lua:107
  it("carries breakdown.area's radius", () => {
    expect(breakdownOf({ Label: '', Lines: ['= 1.9m'], Radius: 19 } as calculator.EntryPlain, Infinity)?.radius).toBe(19);
    expect(breakdownOf({ Label: '', Lines: ['x'] } as calculator.EntryPlain, Infinity)?.radius).toBeUndefined();
  });

  it('lays out reservations with PoB columns', () => {
    const bd = breakdownOf(
      {
        Label: '',
        Lines: [],
        Reservations: [{ SkillName: 'Hatred', Base: '50%', More: 'x 0.9', Efficiency: 'x 0.8333', Total: '37.5%' } as calculator.BReservationPlain]
      } as unknown as calculator.EntryPlain,
      Infinity
    );
    const t = bd?.tables?.[0];
    expect(t?.label).toBe('Reservations');
    expect(t?.columns.map((c) => c.label)).toEqual(['Skill', 'Base', 'MCM', 'More/less', 'Inc/red', 'Efficiency', 'Efficiency More/less', 'Reservation']);
    expect(t?.rows).toEqual([
      { skillName: 'Hatred', base: '50%', mult: '', more: 'x 0.9', inc: '', efficiency: 'x 0.8333', efficiencyMore: '', total: '37.5%' }
    ]);
  });

  it('lays out damage types with PoB columns', () => {
    const bd = breakdownOf(
      {
        Label: '',
        Lines: [],
        DamageTypes: [
          { Source: 'Physical', Base: '10 to 20', Total: '10 to 20', ConvDst: '50% to Fire', GainDst: '10% gained as Fire' } as calculator.BDamageTypePlain
        ]
      } as unknown as calculator.EntryPlain,
      Infinity
    );
    const t = bd?.tables?.[0];
    expect(t?.label).toBe('Damage types');
    expect(t?.columns.map((c) => c.label)).toEqual(['From', 'Base', 'Inc/red', 'More/less', 'Converted Damage', 'Total', 'Conversion', 'Gain']);
    expect(t?.rows?.[0]).toEqual({
      source: 'Physical',
      base: '10 to 20',
      inc: '',
      more: '',
      convSrc: '',
      total: '10 to 20',
      convDst: '50% to Fire',
      gainDst: '10% gained as Fire'
    });
  });
});

// CalcBreakdownControl.lua:37-56
describe('cell modifier sections', () => {
  it("carries every modName entry's fields, a list of names marked Multi", async () => {
    const { calcsView } = await import('./calcs');
    const { calcSections } = await import('$lib/calcs/calc_sections');
    const view = calcsView(calcSections, { Output: { Life: 100 }, OutputTable: {}, SkillFlags: {}, Breakdown: {} } as never);
    const cells = view.sections.flatMap((s) => s.blocks ?? []).flatMap((b) => b.rows.flatMap((r) => r.cells));
    const life = cells.flatMap((c) => c.sections ?? []).find((s) => s.sd.ModName.includes('Life') && s.sd.ModType === 'BASE');
    expect(life?.sd).toMatchObject({ Multi: false, Cfg: '', Enemy: false });
    const multi = cells.flatMap((c) => c.sections ?? []).find((s) => s.sd.Multi);
    expect(multi?.sd.ModName.length).toBeGreaterThan(1);
  });
});

// CalcFormat.lua:34-60
// CalcSections.lua:1732-1773
describe('the mod-sum props Go evaluates', () => {
  it('include entries without a cfg, and carry their actor', async () => {
    const { calcElements, cellId } = await import('./calcs');
    const { calcSections } = await import('$lib/calcs/calc_sections');
    const props = calcElements(calcSections);
    expect(props[`${cellId('Life', 0, 0, 0)}:0`]).toMatchObject({ modName: 'Life', modType: 'BASE', modSource: 'Item' });
    expect(props[`${cellId('Flasks', 0, 0, 0)}:0`]).toMatchObject({ modName: 'FlaskEffect', modType: 'INC', actor: 'player' });
    expect(Object.values(props).filter((p) => p.actor === 'player')).toHaveLength(7);
  });

  // CalcSections.lua:1729-1816
  it("carry the pin's Flasks, Rage and Charges sections", async () => {
    const { calcSections } = await import('$lib/calcs/calc_sections');
    const ids = calcSections.map((s) => s[1]);
    expect(ids.slice(ids.indexOf('Flasks'), ids.indexOf('MiscDefences') + 1)).toEqual(['Flasks', 'Rage', 'Charges', 'MiscDefences']);
    expect(calcSections.find((s) => s[1] === 'Charges')![4].map((b) => b.label)).toEqual(['Charges', 'Endurance', 'Frenzy', 'Power']);
  });

  // CalcSections.lua:431-464
  it("carry the pin's Pacts section", async () => {
    const { calcsView } = await import('./calcs');
    const { calcSections } = await import('$lib/calcs/calc_sections');
    const ids = calcSections.map((s) => s[1]);
    expect(ids.slice(ids.indexOf('Warcries'), ids.indexOf('Dot') + 1)).toEqual(['Warcries', 'Pacts', 'Dot']);
    const out = { CreatePactOffensiveCalcSection: 1, LyciaUpTimeRatio: 6.96, LyciaEmpoweredCount: 4 };
    const view = calcsView(calcSections, { Output: out, OutputTable: {}, SkillFlags: {}, Breakdown: {} } as never);
    const pacts = view.sections.find((s) => s.id === 'Pacts')!;
    expect(pacts.title).toBe('Empowered Spells');
    const rows = pacts.blocks![0].rows;
    expect(rows.map((r) => r.label)).toEqual(['', 'Pact of Lycia']);
    expect(rows[1].cells.slice(0, 2).map((c) => c.text)).toEqual(['6.96%', '4']);
  });
});
