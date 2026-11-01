// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import { Blob as NodeBlob } from 'node:buffer';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()) }));

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const VIEW = { className: 'Sorceress', ascendClassName: 'None', level: 40, output: { Life: 683, Mana: 231, EnergyShield: 0, TotalDPS: 24.914285714286 } };
const open = vi.fn((xml: string) => Promise.resolve(xml ? VIEW : VIEW));
const poe2Worker = vi.fn(() => Promise.resolve({ Open: open }));
vi.mock('$lib/go/worker2', () => ({ poe2Worker }));

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, poe2Build, poe2Enabled } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const { POE2_DISABLED, POE2_LEGACY } = await import('./game_route');
const types = await import('$lib/types');

const serialize = (b: unknown) => new TextDecoder().decode(types.builds.SerializeBuild(b as never).unwrap());
let poe2Xml = '';
let tick: MockInstance;

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.stubGlobal('Blob', NodeBlob);
  poe2Xml = await readFile(resolve(process.cwd(), '../wasm2/exposition/testdata/sorceress_spark.xml'), 'utf8');
}, 120_000);

const importAsNew = async (xml: string) => {
  const Page = (await import('$lib/shell/AppImportExport.svelte')).default;
  const view = render(Page);
  const box = within(view.getByRole('region', { name: 'Build import' }));
  const field = box.getByRole('textbox', { name: /^To import a build/ });
  await fireEvent.input(field, { target: { value: types.pob.CompressEncode(xml).unwrap() } });
  await fireEvent.click(box.getByRole<HTMLInputElement>('radio', { name: /Import as a new build/ }));
  const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
  await vi.waitFor(() => expect(go.disabled).toBe(false));
  await fireEvent.click(go);
  return box;
};

describe('a PoE2 build code in the import dialog', () => {
  it('with the flag off is refused, the PoE1 build stays, and no second worker is made', async () => {
    poe2Enabled.set(false);
    const before = serialize(workerInstance.currentBuild);
    tick.mockClear();
    const box = await importAsNew(poe2Xml);
    await vi.waitFor(() => expect(box.getByRole('alert').textContent).toBe(`Import failed: ${POE2_DISABLED}`));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    expect(tick).not.toHaveBeenCalled();
    expect(poe2Worker).not.toHaveBeenCalled();
    expect(get(poe2Build)).toBeUndefined();
  });

  it('saved before PoB-PoE2 renamed its root is refused whatever the flag', async () => {
    poe2Enabled.set(true);
    const legacy = poe2Xml.replaceAll('PathOfBuilding2', 'PathOfBuilding');
    const box = await importAsNew(legacy);
    await vi.waitFor(() => expect(box.getByRole('alert').textContent).toBe(`Import failed: ${POE2_LEGACY}`));
    expect(poe2Worker).not.toHaveBeenCalled();
    expect(workerInstance.currentBuild).toBeDefined();
  });

  it('with the flag on opens in the second worker and closes the PoE1 build', async () => {
    poe2Enabled.set(true);
    tick.mockClear();
    await importAsNew(poe2Xml);
    await vi.waitFor(() => expect(get(poe2Build)).toEqual(VIEW));
    expect(poe2Worker).toHaveBeenCalledTimes(1);
    expect(open).toHaveBeenCalledWith(poe2Xml);
    expect(workerInstance.currentBuild).toBeUndefined();
    expect(get(currentBuild)).toBeUndefined();
    expect(tick).not.toHaveBeenCalledWith('ImportCode');

    // the frame shows it: the tree route's PoE2 body and the sidebar's PoE2 stats
    const Tree = (await import('../../routes/tree/+page.svelte')).default;
    expect(await within(render(Tree).container).findByText('Level 40 Sorceress')).toBeTruthy();
    const Sidebar = (await import('$lib/shell/AppSidebar.svelte')).default;
    const stats = within(render(Sidebar).getByRole('region', { name: 'Build stats' }));
    expect(stats.getByText('683')).toBeTruthy();
  });

  it('is read in the main thread, and a PoE1 worker still loading does not hold it up', async () => {
    poe2Enabled.set(true);
    poe2Build.set(undefined);
    workerInstance.ClearBuild();
    expect(get(currentBuild)).toBeUndefined();
    const importCode = vi.spyOn(workerInstance, 'ImportCode');
    const clear = vi.spyOn(workerInstance, 'ClearBuild').mockImplementation(() => new Promise<never>(() => {}) as never);
    try {
      await importAsNew(poe2Xml);
      await vi.waitFor(() => expect(get(poe2Build)).toEqual(VIEW));
      expect(open).toHaveBeenLastCalledWith(poe2Xml);
      expect(clear).toHaveBeenCalled();
      expect(importCode).not.toHaveBeenCalled();
    } finally {
      importCode.mockRestore();
      clear.mockRestore();
    }
  });

  it('a PoE1 build opening afterwards clears the PoE2 view', async () => {
    poe2Build.set(VIEW);
    await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
    expect(get(poe2Build)).toBeUndefined();
  });
});
