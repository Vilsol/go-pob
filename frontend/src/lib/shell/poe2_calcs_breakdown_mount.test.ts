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
const w2 = vi.hoisted(() => ({ CalcsPage: vi.fn(), TreeData: vi.fn(() => new Promise(() => {})) }));
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
});
