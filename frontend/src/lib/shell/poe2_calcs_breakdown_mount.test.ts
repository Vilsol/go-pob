// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, screen, within } from '@testing-library/svelte';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

globalThis.matchMedia = ((q: string) => ({
  matches: false,
  media: q,
  addEventListener: () => {},
  removeEventListener: () => {}
})) as unknown as typeof matchMedia;

const poe1 = vi.hoisted(() => ({}) as Record<string, ReturnType<typeof vi.fn>>);
const syncWrap = new Proxy(poe1, { get: (t, k: string) => t[k] ?? (() => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/calcs') } }));

// PoB-PoE2's main calculation of the sorceress fixture (oracle/poe2_calcs.lua) and the sections its
// CalcBreakdownControl builds for each breakdown (oracle/poe2_breakdowns.lua)
const oracle = (name: string) => JSON.parse(readFileSync(resolve(__dirname, `../../../../testdata/oracle/${name}.json`), 'utf8')) as Record<string, never>;
type Calcs = {
  output: Record<string, number | boolean | string>;
  skillFlags: Record<string, boolean>;
  skillData: Record<string, boolean>;
  mods: Record<string, number>;
};
type Section = { type: string; lines?: string[]; radius?: number; label?: string; columns?: { label: string; key: string }[]; rows?: Record<string, string>[] };
const calcs = oracle('poe2_calcs_sorceress_spark') as unknown as Calcs;
const sections = oracle('poe2_breakdowns_sorceress_spark') as unknown as Record<string, Section[]>;

// the worker's CalcsBreakdowns, as PoB-PoE2 gives them
const breakdowns = Object.fromEntries(
  Object.entries(sections).map(([k, ss]) => [
    k,
    {
      Lines: ss.find((s) => s.type === 'TEXT')?.lines ?? [],
      Radius: ss.find((s) => s.type === 'RADIUS')?.radius ?? 0,
      Tables: ss
        .filter((s) => s.type === 'TABLE')
        .map((s) => ({ Label: s.label ?? '', Footer: '', Columns: s.columns!.map((c) => ({ Label: c.label, Key: c.key })), Rows: s.rows }))
    }
  ])
);

const numbers = Object.fromEntries(Object.entries(calcs.output).filter((e): e is [string, number] => typeof e[1] === 'number'));
const w2 = vi.hoisted(() => ({
  CalcsPage: vi.fn(),
  CalcsModSection: vi.fn(),
  CalcsBreakdownModList: vi.fn(),
  SidebarBreakdownKey: vi.fn(),
  SidebarBreakdownSections: vi.fn(),
  TreeData: vi.fn(() => new Promise(() => {}))
}));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: () => Promise.resolve(w2) }));

const { poe2Build } = await import('$lib/global');
const { calcsBreakdowns2, calcsView2 } = await import('$lib/display/calcs2');
const { calcSections2 } = await import('$lib/calcs/calc_sections2');
const { rowKey } = await import('$lib/design/calcs');
const { breakdownStep } = await import('$lib/design/breakdown_lines');

// the row whose first breakdown cell is bd
const rowOf = (bd: string) => {
  const view = calcsView2(calcSections2, { ...calcs, breakdowns: calcsBreakdowns2(breakdowns as never) });
  for (const s of view.sections)
    for (const [j, b] of (s.blocks ?? []).entries()) for (const [k, r] of b.rows.entries()) if (r.cells.some((c) => c.bd === bd)) return rowKey(s.id, j, k);
  throw new Error(`no cell shows breakdown ${bd}`);
};
// the row labelled label whose cell names a mod table
const modRowOf = (label: string) => {
  const view = calcsView2(calcSections2, { ...calcs, breakdowns: calcsBreakdowns2(breakdowns as never) });
  for (const s of view.sections)
    for (const [j, b] of (s.blocks ?? []).entries())
      for (const [k, r] of b.rows.entries()) if (r.label === label && r.cells.some((c) => c.sections?.length)) return rowKey(s.id, j, k);
  throw new Error(`no ${label} row with a mod table`);
};
// PoB-PoE2's AddModSection tables over the fixture (oracle/poe2_mod_sections.lua)
type ModRows = { rows?: { value: string; name: string; flags: string; tags: string; source: string; sourceName: string; totals?: string[] }[] };
const modSections = oracle('poe2_mod_sections_sorceress_spark') as unknown as Record<string, ModRows>;
const AppCalcs = (await import('./AppCalcs.svelte')).default;

describe('a PoE2 Calcs cell with a breakdown', () => {
  it("shows PoB-PoE2's breakdown lines and sources table in the breakdown panel", async () => {
    w2.CalcsPage.mockResolvedValue({
      mods: calcs.mods,
      flags: Object.fromEntries(Object.entries(calcs.output).filter((e) => e[1] === true)),
      strings: Object.fromEntries(Object.entries(calcs.output).filter((e) => typeof e[1] === 'string')),
      breakdowns: calcsBreakdowns2(breakdowns as never)
    });
    poe2Build.set({
      className: 'Sorceress',
      ascendClassName: 'None',
      level: 1,
      output: numbers,
      skillFlags: calcs.skillFlags,
      skillData: calcs.skillData,
      socketGroups: [],
      mainSocketGroup: 1,
      mainActiveSkill: 1
    } as never);
    const { container } = render(AppCalcs);

    // CalcSections.lua: the Life section's Total row names breakdown "Life"
    const life = sections.Life.find((s) => s.type === 'TEXT')!.lines!;
    expect(life.length, 'PoB-PoE2 writes a Life breakdown for the fixture').toBeGreaterThan(1);
    const lifeKey = rowOf('Life');
    const row = await vi.waitFor(
      () => {
        const r = container.querySelector<HTMLElement>(`[data-key="${lifeKey}"]`);
        if (!r) throw new Error('no Life total row');
        return r;
      },
      { timeout: 10_000 }
    );
    await fireEvent.mouseEnter(row);
    const panel = await vi.waitFor(() => screen.getByRole('complementary', { name: 'Breakdown' }));
    const steps = await vi.waitFor(() => within(panel).getByRole('group', { name: 'Calculation' }));
    const shown = [...steps.querySelectorAll('[data-step]')].map((el) => el.children[1].textContent);
    expect(shown).toEqual(life.map((l) => (breakdownStep(l) as { value: string }).value));

    // the Evasion total's slots table
    const evasion = sections.Evasion.find((s) => s.type === 'TABLE')!;
    await fireEvent.mouseLeave(row);
    const evasionRow = container.querySelector<HTMLElement>(`[data-key="${rowOf('Evasion')}"]`)!;
    expect(evasionRow, 'no Evasion total row').toBeTruthy();
    await fireEvent.mouseEnter(evasionRow);
    const table = await vi.waitFor(() => within(screen.getByRole('complementary', { name: 'Breakdown' })).getByRole('table', { name: 'Sources' }));
    expect(
      within(table)
        .getAllByRole('columnheader')
        .map((c) => c.textContent)
    ).toEqual(evasion.columns!.map((c) => c.label));
    expect(within(table).getAllByRole('row')).toHaveLength(evasion.rows!.length + 1);
  }, 30_000);

  it("shows PoB-PoE2's mod table for a cell's mod name entry", async () => {
    w2.CalcsPage.mockResolvedValue({
      mods: calcs.mods,
      flags: Object.fromEntries(Object.entries(calcs.output).filter((e) => e[1] === true)),
      strings: Object.fromEntries(Object.entries(calcs.output).filter((e) => typeof e[1] === 'string')),
      breakdowns: calcsBreakdowns2(breakdowns as never)
    });
    const want = modSections['Life|BASE|||false|'].rows!;
    expect(want.length, 'PoB-PoE2 lists the fixture Life BASE mods').toBeGreaterThan(1);
    w2.CalcsModSection.mockImplementation((sd: { ModName: string[]; ModType: string; ModSource: string }) =>
      Promise.resolve(
        (modSections[`${sd.ModName.join(',')}|${sd.ModType}||${sd.ModSource}|false|`]?.rows ?? []).map((r) => ({
          Value: r.value,
          Name: r.name,
          Flags: r.flags,
          Tags: r.tags,
          Source: r.source,
          SourceName: r.sourceName,
          Totals: r.totals
        }))
      )
    );
    poe2Build.set({
      className: 'Sorceress',
      ascendClassName: 'None',
      level: 1,
      output: numbers,
      skillFlags: calcs.skillFlags,
      skillData: calcs.skillData,
      socketGroups: [],
      mainSocketGroup: 1,
      mainActiveSkill: 1
    } as never);
    const { container } = render(AppCalcs);

    // CalcSections.lua:1602 (PoB-PoE2): Life's Total Base row is { modName = "Life", modType = "BASE" }
    const key = modRowOf('Total Base');
    const row = await vi.waitFor(
      () => {
        const r = container.querySelector<HTMLElement>(`[data-key="${key}"]`);
        if (!r) throw new Error('no Total Base row');
        return r;
      },
      { timeout: 10_000 }
    );
    await fireEvent.mouseEnter(row);
    const table = await vi.waitFor(() => within(screen.getByRole('complementary', { name: 'Breakdown' })).getByRole('table', { name: 'Modifiers' }));
    expect(w2.CalcsModSection).toHaveBeenCalledWith({ ModName: ['Life'], Multi: false, ModType: 'BASE', Cfg: '', ModSource: '', Enemy: false, Actor: '' });
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows.map((r) => within(r).getAllByRole('cell')[0].textContent)).toEqual(want.map((r) => r.value));
  }, 30_000);

  it("lists the buffs, combat buffs and curses PoB-PoE2's Calcs tab shows", async () => {
    // oracle/poe2_calcs_buff_lists.lua: the lists PoB-PoE2's CALCS calculation writes
    const want = readFileSync(resolve(__dirname, '../../../../testdata/oracle/poe2_calcs_buff_lists.jsonl'), 'utf8')
      .split('\n')
      .filter(Boolean)
      .map(
        (l) =>
          JSON.parse(l) as {
            build: string;
            BuffList: string;
            CombatList: string;
            CurseList: string;
            SkillBuffs?: { value: string; name: string; flags: string; tags: string; source: string; sourceName: string }[];
          }
      )
      .find((r) => r.build === 'skill_buffs.xml')!;
    expect(want.BuffList && want.CurseList, 'PoB-PoE2 lists buffs and curses for the fixture').toBeTruthy();
    w2.CalcsPage.mockResolvedValue({
      mods: calcs.mods,
      flags: {},
      strings: { BuffList: want.BuffList, CombatList: want.CombatList, CurseList: want.CurseList },
      breakdowns: calcsBreakdowns2(breakdowns as never)
    });
    poe2Build.set({
      className: 'Sorceress',
      ascendClassName: 'None',
      level: 1,
      output: numbers,
      skillFlags: calcs.skillFlags,
      skillData: calcs.skillData,
      socketGroups: [],
      mainSocketGroup: 1,
      mainActiveSkill: 1
    } as never);
    render(AppCalcs);

    const list = await vi.waitFor(
      () => {
        const l = document.querySelector('dl[aria-label="Buffs and debuffs"]');
        if (!l) throw new Error('no buff lists');
        return l;
      },
      { timeout: 10_000 }
    );
    const terms = [...list.querySelectorAll('dt')].map((d) => d.textContent);
    const texts = [...list.querySelectorAll('dd')].map((d) => d.textContent);
    expect(terms).toEqual(['Aura and Buff Skills', 'Combat Buffs', 'Curses and Debuffs']);
    expect(texts).toEqual([want.BuffList, want.CombatList || '—', want.CurseList]);

    // CalcBreakdownControl.lua:293-296 (PoB-PoE2): the SkillBuffs breakdown's mod table
    const buffs = want.SkillBuffs ?? [];
    expect(buffs.length, 'PoB-PoE2 lists SkillBuffs mods for the fixture').toBeGreaterThan(1);
    w2.CalcsBreakdownModList.mockResolvedValue(
      buffs.map((r) => ({ Value: r.value, Name: r.name, Flags: r.flags, Tags: r.tags, Source: r.source, SourceName: r.sourceName }))
    );
    await fireEvent.mouseEnter(within(list as HTMLElement).getByRole('button', { name: 'Aura and Buff Skills' }));
    const table = await vi.waitFor(() => within(screen.getByRole('complementary', { name: 'Breakdown' })).getByRole('table', { name: 'Modifiers' }));
    expect(w2.CalcsBreakdownModList).toHaveBeenCalledWith('SkillBuffs');
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows.map((r) => within(r).getAllByRole('cell')[0].textContent)).toEqual(buffs.map((r) => r.value));
    expect(rows.map((r) => within(r).getAllByRole('cell')[1].textContent)).toEqual(buffs.map((r) => r.name));
  }, 30_000);

  it("hands a sidebar line's breakdown the cells PoB-PoE2's GetSidebarBreakdown shows", async () => {
    // oracle/poe2_sidebar_breakdowns.lua: the Life line's key and the cells it shows
    type Cell = { breakdown?: string; modName?: string[]; modType?: string; label?: string };
    const lines = oracle('poe2_sidebar_breakdowns_sorceress_spark') as unknown as { stat: string; key: string; cells: Cell[] }[];
    const life = lines.find((l) => l.stat === 'Life')!;
    expect(life.cells.length, 'PoB-PoE2 shows more than the Life breakdown').toBeGreaterThan(1);
    poe2Build.set({
      className: 'Sorceress',
      ascendClassName: 'None',
      level: 1,
      output: numbers,
      skillFlags: calcs.skillFlags,
      skillData: calcs.skillData,
      socketGroups: [],
      mainSocketGroup: 1,
      mainActiveSkill: 1
    } as never);
    const { poe2Engine } = await import('$lib/go/engine_poe2.svelte');
    const { get } = await import('svelte/store');
    const line = get(poe2Engine.sidebar)!
      .sections.flatMap((s) => s.groups.flat())
      .find((l) => l.stat === 'Life')!;
    expect(line.bd).toEqual({ stat: 'Life', modNames: ['Life'], ignoredSections: ['Base from Gear', 'Inc. from Tree'] });
    w2.SidebarBreakdownKey.mockResolvedValue({ key: life.key, source: life.key });
    const shown = [{ label: 'x', mods: [] }];
    w2.SidebarBreakdownSections.mockResolvedValue(shown);
    expect(await poe2Engine.breakdownSteps!(line)).toBe(shown);
    expect(w2.SidebarBreakdownKey).toHaveBeenCalledWith(line.bd);
    const cells = (w2.SidebarBreakdownSections.mock.calls[0][0] as (Cell & { modName?: string | string[] })[]).map((c) => {
      const cell: Cell = {};
      if (c.breakdown !== undefined) cell.breakdown = c.breakdown;
      if (c.modName !== undefined) cell.modName = [c.modName].flat();
      if (c.modType !== undefined) cell.modType = c.modType;
      if (c.label !== undefined) cell.label = c.label;
      return cell;
    });
    expect(cells).toEqual(life.cells);
  });
});
