// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within, screen } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import { Blob as NodeBlob } from 'node:buffer';
import type { storage } from '$lib/types';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

// the build list needs a real store: an in-memory one with the calls storage_init.ts makes
vi.mock('@zenfs/core', () => {
  const files = new Map<string, Uint8Array>();
  const dirs = new Set<string>(['/']);
  const norm = (p: string) => p.replace(/\/+/g, '/').replace(/(.)\/$/, '$1');
  const enoent = () => Object.assign(new Error('ENOENT'), { code: 'ENOENT' });
  type Done<T = void> = (err: Error | null, value?: T) => void;
  const fs = {
    readdir: (p: string, cb: Done<string[]>) => {
      const d = norm(p);
      const under = d === '/' ? '/' : `${d}/`;
      const names = new Set<string>();
      for (const k of [...files.keys(), ...dirs]) if (k.startsWith(under) && k !== d) names.add(k.slice(under.length).split('/')[0]);
      cb(null, [...names]);
    },
    statSync: (p: string) => ({ isFile: () => files.has(norm(p)), isDirectory: () => !files.has(norm(p)), mtime: new Date() }),
    readFileSync: (p: string) => Buffer.from(files.get(norm(p)) ?? new Uint8Array()),
    readFile: (p: string, cb: Done<Buffer>) => {
      const f = files.get(norm(p));
      if (f) cb(null, Buffer.from(f));
      else cb(enoent());
    },
    writeFile: (p: string, v: Uint8Array, cb: Done) => {
      files.set(norm(p), v);
      cb(null);
    },
    access: (p: string, cb: Done) => cb(files.has(norm(p)) || dirs.has(norm(p)) ? null : enoent()),
    mkdir: (p: string, _mode: number, cb: Done) => {
      dirs.add(norm(p));
      cb(null);
    },
    unlink: (p: string, cb: Done) => {
      files.delete(norm(p));
      cb(null);
    }
  };
  return { configure: () => Promise.resolve(undefined), fs };
});

vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const VIEW = { className: 'Sorceress', ascendClassName: 'None', level: 40, output: { Life: 683, Mana: 231 } };
let poe2Xml = '';
const open = vi.fn((xml: string) => Promise.resolve(xml ? VIEW : VIEW));
const serializeXml = vi.fn(() => Promise.resolve(poe2Xml));
// PoB-PoE2's TreeData/0_5/tree.json classes, as wasm2/exposition's TestTreeClassesAreTheTreesClasses asserts them
const TREE_CLASSES = ['Witch', 'Ranger', 'Warrior', 'Sorceress', 'Huntress', 'Mercenary', 'Monk', 'Druid'];
const treeClasses = vi.fn(() => Promise.resolve(TREE_CLASSES));
const poe2Worker = vi.fn(() => Promise.resolve({ Open: open, Serialize: serializeXml, TreeClasses: treeClasses }));
vi.mock('$lib/go/worker2', () => ({ poe2Worker }));

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, currentBuildPath, poe2Build, poe2Enabled } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const { toasts } = await import('$lib/design/toast');
const { POE2_DISABLED } = await import('$lib/go/game_route');
const COMPARE_MIXED = 'A Path of Exile 2 build cannot be compared with a Path of Exile 1 build.';
const NAME = 'PoE2 spark';
let tick: MockInstance;

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.stubGlobal('Blob', NodeBlob);
  poe2Xml = await readFile(resolve(process.cwd(), '../wasm2/exposition/testdata/sorceress_spark.xml'), 'utf8');
  poe2Enabled.set(true);
}, 120_000);

const entry = async () => ((await workerInstance.ListBuilds('')) as storage.DirEntry[]).find((e) => e.Name === NAME);

describe('a PoE2 build in the build list', () => {
  it('is saved as the PoE2 worker serializes it, and lists with its class, level, main skill and a PoE2 badge', async () => {
    currentBuildPath.current = '/Some PoE1 build';
    poe2Build.set(VIEW);
    currentBuildPath.current = '';
    const Poe2 = (await import('./AppPoe2Build.svelte')).default;
    const view = within(render(Poe2, { view: VIEW }).getByRole('region', { name: 'Path of Exile 2 build' }));
    await fireEvent.click(view.getByRole('button', { name: 'Save' }));
    const dialog = within(await screen.findByRole('dialog', { name: 'Save build as' }));
    await fireEvent.input(dialog.getByRole('textbox', { name: 'Build name' }), { target: { value: NAME } });
    await fireEvent.click(dialog.getByRole('button', { name: 'Save' }));
    await vi.waitFor(() => expect(currentBuildPath.current).toBe(`/${NAME}`));
    expect(serializeXml).toHaveBeenCalledTimes(1);
    expect(await entry()).toMatchObject({ Class: 'Sorceress', Level: 40, Ascendancy: '', MainSkill: 'Spark', Game: 'poe2' });

    await fireEvent.click(view.getByRole('button', { name: 'Save' }));
    await vi.waitFor(() => expect(serializeXml).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('dialog', { name: 'Save build as' })).toBeNull();
  });

  it('reopens in the PoE2 worker from the list, with the saved XML', async () => {
    poe2Build.set(undefined);
    open.mockClear();
    const List = (await import('./AppBuildList.svelte')).default;
    const page = render(List);
    const link = await page.findByRole('link', { name: NAME });
    const row = link.closest('[role="row"]') as HTMLElement;
    expect(within(row).getByText('PoE2').getAttribute('data-game')).toBe('poe2');
    expect(within(row).getByText('Spark')).toBeTruthy();
    await fireEvent.click(link);
    await vi.waitFor(() => expect(get(poe2Build)).toEqual(VIEW));
    expect(open).toHaveBeenCalledWith(poe2Xml);
    expect(currentBuildPath.current).toBe(`/${NAME}`);
    expect(tick).not.toHaveBeenCalledWith('OpenBuild');
  });

  it("filters by the PoE2 tree's classes once a PoE2 build is listed, and asks the PoE2 worker only with the flag on", async () => {
    poe2Enabled.set(false);
    treeClasses.mockClear();
    const List = (await import('./AppBuildList.svelte')).default;
    const off = render(List);
    await off.findByRole('link', { name: NAME });
    const offChips = within(off.getByRole('group', { name: 'Filter by class' }));
    expect(offChips.queryByRole('button', { name: 'Sorceress' })).toBeNull();
    expect(treeClasses).not.toHaveBeenCalled();
    off.unmount();

    poe2Enabled.set(true);
    const page = render(List);
    await page.findByRole('link', { name: NAME });
    const chips = within(page.getByRole('group', { name: 'Filter by class' }));
    const names = () => chips.getAllByRole('button').map((b) => b.textContent?.trim());
    await vi.waitFor(() =>
      expect(names()).toEqual([
        'All',
        'Witch',
        'Shadow',
        'Marauder',
        'Ranger',
        'Duelist',
        'Templar',
        'Scion',
        'Warrior',
        'Sorceress',
        'Huntress',
        'Mercenary',
        'Monk',
        'Druid'
      ])
    );
    expect(treeClasses).toHaveBeenCalledTimes(1);
    await fireEvent.click(chips.getByRole('button', { name: 'Sorceress' }));
    expect(page.getByRole('link', { name: NAME })).toBeTruthy();
    await fireEvent.click(chips.getByRole('button', { name: 'Monk' }));
    await vi.waitFor(() => expect(page.queryByRole('link', { name: NAME })).toBeNull());
  });

  it('a PoE1 build in the list has no badge', async () => {
    await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
    await workerInstance.SaveBuildAs('/A PoE1 build');
    expect(await workerInstance.ListBuilds('')).toContainEqual(expect.objectContaining({ Name: 'A PoE1 build', Game: '' }));
  });
});

describe('with the PoE2 flag off', () => {
  it('opening a saved PoE2 build from the list says why it is refused and leaves the open PoE1 build as it was', async () => {
    poe2Enabled.set(false);
    await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
    currentBuildPath.current = '/A PoE1 build';
    const build = workerInstance.currentBuild;
    open.mockClear();
    toasts.set([]);
    const { accountSession } = await import('./account_session.svelte');
    const character = { name: 'Char', realm: 'pc', league: 'Standard', class: 'Witch', level: 90 };
    Object.assign(accountSession, {
      stage: 'SELECTCHAR',
      status: 'Character list successfully retrieved.',
      accountName: 'acc',
      characters: [character],
      selected: character
    });
    const session = JSON.stringify(accountSession);
    const List = (await import('./AppBuildList.svelte')).default;
    const page = render(List);
    await fireEvent.click(await page.findByRole('link', { name: NAME }));
    await vi.waitFor(() => expect(get(toasts).map((t) => t.message)).toEqual([POE2_DISABLED]));
    expect(currentBuildPath.current).toBe('/A PoE1 build');
    expect(workerInstance.currentBuild).toBe(build);
    expect(open).not.toHaveBeenCalled();
    expect(get(poe2Build)).toBeUndefined();
    expect(JSON.stringify(accountSession)).toBe(session);
  });
});

describe('Compare', () => {
  it('refuses a PoE2 build, by code or from the list, and keeps the PoE1 build', async () => {
    await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
    const code = types.pob.CompressEncode(poe2Xml).unwrap();
    await expect(workerInstance.LoadCompareCode(code)).rejects.toThrow(COMPARE_MIXED);
    await expect(workerInstance.LoadCompareSaved(`/${NAME}`)).rejects.toThrow(COMPARE_MIXED);
    expect(workerInstance.GetCompareInfo().seq).toBe(0);
    expect(workerInstance.currentBuild).toBeDefined();
  });

  it('says so on the Compare page', async () => {
    await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
    const Compare = (await import('./AppCompare.svelte')).default;
    const page = render(Compare);
    const code = types.pob.CompressEncode(poe2Xml).unwrap();
    const section = within(await page.findByRole('region', { name: 'Choose a build to compare' }));
    await fireEvent.input(section.getByRole('textbox', { name: 'Comparison build code' }), { target: { value: code } });
    await fireEvent.click(section.getByRole('button', { name: 'Compare' }));
    expect((await section.findByText(/^Comparison failed:/)).textContent).toBe(`Comparison failed: ${COMPARE_MIXED}`);
  });
});
