// @vitest-environment jsdom
// Build.lua:355-414
// CompareTab.lua:290-340

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import type { CompareInfo } from './compare_types';

afterEach(cleanup);
const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted bar threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const A = '0027-saboteur-spark-of-the-nova.xml';
const B = '0045-hierophant-spark-of-the-nova.xml';
const CORPUS = `${import.meta.dirname}/../../../../testdata/corpus-3.29`;

type Build = NonNullable<typeof workerInstance.currentBuild>;
const parts = (b: Build) => {
  const calc = types.calculator.NewCalculator(b)!;
  try {
    const p = calc.PoB!;
    return { tree: p.ActiveSpecIndex(), skill: p.Skills.ActiveSkillSet, item: p.Items.ActiveItemSet, config: p.Config.ActiveConfigSet };
  } finally {
    calc.release();
  }
};

let info: CompareInfo;
let bBuild: Build;

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(A);
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.spyOn(workerInstance as unknown as { compareSide: () => Promise<unknown> }, 'compareSide').mockResolvedValue({ outputs: {} });
  const code = types.pob.CompressEncode(await readFile(`${CORPUS}/${B}`, 'utf8')).unwrap();
  info = await workerInstance.LoadCompareCode(code, 'B');
  bBuild = (workerInstance as unknown as { compareB: { build: Build } }).compareB.build;
}, 120_000);

const mount = async () => {
  const Bar = (await import('./AppCompareSets.svelte')).default;
  const view = render(Bar, { info });
  const group = within(await view.findByRole('group', { name: 'Comparison build sets' }, { timeout: 10_000 }));
  return { view, group };
};
const options = (sel: HTMLSelectElement) => [...sel.options].filter((o) => !o.disabled).map((o) => o.text);
const other = async (sel: HTMLSelectElement) => {
  await waitFor(() => expect(options(sel).length).toBeGreaterThan(1));
  return sel.value === '0' ? 1 : 0;
};

describe('Compare top bar', () => {
  it('shows each build’s Loadout select first; the sets are behind a toggle', async () => {
    const { group } = await mount();
    const a = within(group.getByRole('region', { name: 'This build' }));
    const b = within(group.getByRole('region', { name: 'Comparison' }));
    await waitFor(() => expect(options(b.getByRole('combobox', { name: 'Comparison loadout' }))).toEqual(['Final', 'Swap on Spark Nova (70 lvl)']));
    await waitFor(() => expect(options(a.getByRole('combobox', { name: 'Primary loadout' }))).toEqual(['Final', 'Swap on Spark Nova (around 70 lvl)']));
    expect(b.queryByRole('combobox', { name: 'Comparison item set' })).toBeNull();
    const toggle = b.getByRole('button', { name: /^Comparison sets/ });
    expect(toggle.getAttribute('aria-expanded')).toBe('false');
    await fireEvent.click(toggle);
    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    for (const n of ['tree', 'skill', 'item', 'config']) expect(b.getByRole('combobox', { name: `Comparison ${n} set` })).toBeTruthy();
  });

  it('picking B’s loadout sets all four of B’s parts; overriding one part keeps the others', async () => {
    const { group } = await mount();
    const b = within(group.getByRole('region', { name: 'Comparison' }));
    const before = parts(bBuild);
    const sel = b.getByRole<HTMLSelectElement>('combobox', { name: 'Comparison loadout' });
    const idx = await other(sel);
    await fireEvent.change(sel, { target: { value: String(idx) } });
    await waitFor(() => expect(parts(bBuild).tree).not.toBe(before.tree));
    const second = parts(bBuild);
    expect([second.skill, second.item, second.config]).not.toContain(undefined);
    expect(second.skill).not.toBe(before.skill);
    expect(second.item).not.toBe(before.item);
    expect(second.config).not.toBe(before.config);
    await waitFor(() => expect(sel.value).toBe(String(idx)));

    await fireEvent.click(b.getByRole('button', { name: /^Comparison sets/ }));
    const tree = b.getByRole<HTMLSelectElement>('combobox', { name: 'Comparison tree set' });
    await waitFor(() => expect(tree.value).toBe(String(second.tree)));
    await fireEvent.change(tree, { target: { value: String(before.tree) } });
    await waitFor(() => expect(parts(bBuild).tree).toBe(before.tree));
    expect(parts(bBuild)).toEqual({ ...second, tree: before.tree });
    await waitFor(() => expect(sel.value).toBe('-1'));
    expect(sel.selectedOptions[0].text).toBe('Custom sets');
  });

  it('picking this build’s loadout sets all four of its parts', async () => {
    const { group } = await mount();
    const a = within(group.getByRole('region', { name: 'This build' }));
    const build = workerInstance.currentBuild!;
    const before = parts(build);
    const sel = a.getByRole<HTMLSelectElement>('combobox', { name: 'Primary loadout' });
    await fireEvent.change(sel, { target: { value: String(await other(sel)) } });
    await waitFor(() => expect(parts(build).tree).not.toBe(before.tree));
    const after = parts(build);
    expect(after.skill).not.toBe(before.skill);
    expect(after.item).not.toBe(before.item);
    expect(after.config).not.toBe(before.config);
  });
});
