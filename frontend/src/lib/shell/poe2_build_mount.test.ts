// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import type { Poe2Calc } from '$lib/go/sync_worker2';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

const WASM2 = resolve(process.cwd(), 'static/go-pob2.wasm');
const FIXTURES = resolve(process.cwd(), '../wasm2/exposition/testdata');
const ORACLE = resolve(process.cwd(), '../testdata/oracle');

const { worker2Instance } = await import('$lib/go/sync_worker2');
const issued: string[] = [];
const poe2Worker = vi.fn(() =>
  Promise.resolve(
    new Proxy(worker2Instance, {
      get: (target, prop, receiver) => {
        const v = Reflect.get(target, prop, receiver) as unknown;
        if (typeof v !== 'function') return v;
        return (...args: unknown[]) => {
          issued.push(`${String(prop)}(${args.map((a) => JSON.stringify(a)).join(',')})`);
          return (v as (...a: unknown[]) => unknown).apply(target, args);
        };
      }
    })
  )
);
vi.mock('$lib/go/worker2', () => ({ poe2Worker }));
// the PoE2 worker caches game data through zenfs, which the harness stubs; a Map stands in for the calls it makes
vi.mock('@zenfs/core', () => {
  const files = new Map<string, Uint8Array>();
  return {
    configure: () => Promise.resolve(undefined),
    fs: {
      promises: {
        exists: (path: string) => Promise.resolve(files.has(path)),
        readFile: (path: string) => (files.has(path) ? Promise.resolve(files.get(path)!) : Promise.reject(new Error(`ENOENT: ${path}`))),
        writeFile: (path: string, value: Uint8Array) => {
          files.set(path, new Uint8Array(value));
          return Promise.resolve();
        },
        readdir: (dir: string) => Promise.resolve([...files.keys()].filter((k) => k.startsWith(`${dir}/`)).map((k) => k.slice(dir.length + 1))),
        rm: (path: string) => {
          for (const k of [...files.keys()]) if (k === path || k.startsWith(`${path}/`)) files.delete(k);
          return Promise.resolve();
        }
      }
    }
  };
});
// the PoE1 worker is not part of a PoE2 build's page
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));
// jsdom has no WebGL; the tree page only needs a renderer to hand its state to
vi.mock('$lib/components/skill-tree/gl_renderer2', () => ({
  createTree2Renderer: () => ({ kind: 'canvas', setView: () => {}, setState: () => {}, destroy: () => {} })
}));

const { poe2Build } = await import('$lib/global');
const Poe2Page = (await import('./Poe2Page.svelte')).default;
const AppSidebar = (await import('./AppSidebar.svelte')).default;
const { poe2Engine } = await import('$lib/go/engine_poe2.svelte');

// the frame sidebar's value for a stat, found by the label PoE2's stat list gives it
const statOf = (container: HTMLElement, name: string) => {
  const line = get(poe2Engine.sidebar)
    ?.sections.flatMap((s) => s.groups.flat())
    .find((l) => l.stat === name);
  if (!line) return undefined;
  const row = [...container.querySelectorAll<HTMLElement>('.stat-row')].find((r) => r.querySelector('.stat-label')?.textContent?.trim() === line.label);
  return row?.querySelector('.stat-value')?.textContent?.trim();
};

const fixture = (name: string) => readFile(resolve(FIXTURES, name), 'utf8');

beforeAll(async () => {
  if (!existsSync(WASM2)) throw new Error(`REFUSING: the PoE2 wasm module is missing — run \`task build-go2\`\n  missing: ${WASM2}`);
  // Go's fetch passes jsdom's AbortSignal, which node's fetch refuses, and reads a streamed body as
  // node's Uint8Array, which is not jsdom's; a whole-body response avoids both
  const nodeFetch = globalThis.fetch;
  vi.stubGlobal('fetch', async (input: RequestInfo | URL, init?: RequestInit) => {
    const res = await nodeFetch(input, { ...init, signal: undefined });
    const body = await res.arrayBuffer();
    return {
      ok: res.ok,
      status: res.status,
      statusText: res.statusText,
      headers: res.headers,
      url: res.url,
      redirected: res.redirected,
      body: undefined,
      arrayBuffer: () => Promise.resolve(body)
    };
  });
  const wasm = await readFile(WASM2);
  // a typed array: jsdom's ArrayBuffer is another realm's, which boot's instanceof check would refuse
  await worker2Instance.boot(new Uint8Array(wasm) as unknown as ArrayBuffer);
}, 120_000);

// the values are PoB-PoE2's: oracle/dump_build.lua and oracle/poe2_sidebar.lua on the pinned checkout
describe('the PoE2 config and main skill writes', () => {
  it('a config option lands in the build and recalculates', async () => {
    const opened = await worker2Instance.Open(await fixture('sorceress_spark.xml'));
    expect(opened.output.FireResist).toBe(-15);
    const set = await worker2Instance.SetConfigOption('resistancePenalty', 0);
    expect(set.output.FireResist).toBe(45);
    expect(set.configInputs).toEqual({ resistancePenalty: 0 });
    expect(worker2Instance.Serialize()).toMatch(/<Input name="resistancePenalty" number="0"\/>/);
    const reset = await worker2Instance.SetConfigOption('resistancePenalty', -60);
    expect(reset.output.FireResist).toBe(-15);
    expect(worker2Instance.Serialize()).not.toMatch(/resistancePenalty/);
  }, 120_000);

  it.each(['sorceress_spark', 'sorceress_three_spells'])(
    "shows the config options PoB-PoE2's config tab shows for %s",
    async (name) => {
      const oracle = JSON.parse(await readFile(resolve(ORACLE, `poe2_sidebar_${name}.json`), 'utf8')) as { configShown: string[] };
      poe2Build.set(await worker2Instance.Open(await fixture(`${name}.xml`)));
      const page = render(Poe2Page, { route: 'configuration' });
      const shown = [...page.container.querySelectorAll<HTMLElement>('[data-config]')].map((e) => e.dataset.config!).sort();
      expect(shown).toEqual(oracle.configShown);
    },
    120_000
  );

  it('editing a config option on the page recalculates the sidebar', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    issued.length = 0;
    const page = render(Poe2Page, { route: 'configuration' });
    const side = render(AppSidebar);
    const stat = (name: string) => statOf(side.container, name);
    expect(stat('FireResist')).toBe('-15%');
    const pick = page.container.querySelector<HTMLSelectElement>('[data-config="resistancePenalty"]')!;
    pick.value = '0';
    await fireEvent.change(pick);
    await vi.waitFor(() => expect(get(poe2Build)?.output.FireResist).toBe(45));
    expect(issued).toEqual(['SetConfigOption("resistancePenalty",0)']);
    await vi.waitFor(() => expect(stat('FireResist')).toBe('45%'));
    expect(stat('LightningResist')).toBe('10%');
  }, 120_000);

  it('switching the main skill recalculates the sidebar', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_three_spells.xml')));
    issued.length = 0;
    const page = render(AppSidebar);
    const box = within(page.getByRole('complementary', { name: 'Main skill and stats' }));
    const stat = (name: string) => statOf(page.container, name);
    expect(stat('TotalDPS')).toBe('24.9');
    const skill = box.getByRole<HTMLSelectElement>('combobox', { name: 'Active skill' });
    expect([...skill.options].map((o) => o.text)).toEqual(['Spark', 'Fireball']);
    skill.value = '2';
    await fireEvent.change(skill);
    await vi.waitFor(() => expect(get(poe2Build)?.output.TotalDPS).toBe(32.76875));
    await vi.waitFor(() => expect(stat('TotalDPS')).toBe('32.8'));

    const group = box.getByRole<HTMLSelectElement>('combobox', { name: 'Main skill' });
    expect([...group.options].map((o) => o.text)).toEqual(['Spark, Fireball', 'Arc']);
    // the frame's picker counts groups from 0
    group.value = '1';
    await fireEvent.change(group);
    await vi.waitFor(() => expect(get(poe2Build)?.output.TotalDPS).toBeCloseTo(25.020454545455, 9));
    expect(issued).toEqual(['SetMainActiveSkill(2)', 'SetMainSocketGroup(2)']);
  }, 120_000);
});

// the values are PoB-PoE2's: AllocNode on node 44871 under spec.allocMode 1, then Undo and Redo, with
// PassiveSpec:Save and the calculated Energy Shield after each
describe('the PoE2 tree undo and redo', () => {
  beforeAll(() => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it('Ctrl+Z and Ctrl+Y on the tree page restore the allocation and recalculate', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    // the keys reach the page through the tree view; the whole PoE2 tree is slow to mount under jsdom
    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const page = render(Poe2Page, { route: 'tree' });
    await vi.waitFor(() => page.getByRole('img', { name: 'Passive skill tree' }), { timeout: 60_000 });
    treeData.mockRestore();
    poe2Build.set((await worker2Instance.TreeClick(44871, false, 0, false, 1)).calc);
    expect(get(poe2Build)?.output.EnergyShield).toBe(10);
    expect((get(poe2Build) as Poe2Calc | undefined)?.tree.allocMode).toEqual({ 44871: 1 });
    issued.length = 0;

    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await vi.waitFor(() => expect(get(poe2Build)?.output.EnergyShield).toBe(0));
    expect((get(poe2Build) as Poe2Calc | undefined)?.tree.alloc).toEqual([54447]);
    expect(worker2Instance.Serialize()).toMatch(/nodes="54447"/);

    await fireEvent.keyDown(window, { key: 'y', ctrlKey: true });
    await vi.waitFor(() => expect(get(poe2Build)?.output.EnergyShield).toBe(10));
    expect(worker2Instance.Serialize()).toMatch(/<WeaponSet1 nodes="44871"\/>/);
    expect(issued).toEqual(['TreeUndo(false)', 'TreeUndo(true)']);
  }, 120_000);
});

// the values are PoB-PoE2's Calcs tab over its main calculation: oracle/poe2_calcs.lua on the pinned checkout
describe('the PoE2 Calcs page', () => {
  it("shows PoB-PoE2's Life section cells and recalculates with the build", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const oracle = JSON.parse(await readFile(resolve(ORACLE, 'poe2_calcs_sorceress_spark.json'), 'utf8')) as { cells: Record<string, string> };
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    issued.length = 0;
    const page = render(Poe2Page, { route: 'calcs' });
    const life = await vi.waitFor(() => within(page.getByRole('region', { name: 'Calculations' })).getByRole('region', { name: 'Life' }), { timeout: 60_000 });
    const row = (label: string) => [...life.querySelectorAll<HTMLElement>('[data-key]')].find((r) => r.textContent?.trim().startsWith(label));
    const cells = (r: HTMLElement | undefined) => [...(r?.querySelectorAll<HTMLElement>('.font-mono > span') ?? [])].map((c) => c.textContent?.trim());
    const want = Object.entries(oracle.cells)
      .filter(([k]) => k.startsWith('Life:0:'))
      .sort(([a], [b]) => Number(a.split(':')[2]) - Number(b.split(':')[2]))
      .map(([, v]) => pobPlain(v));
    const shown = [...life.querySelectorAll<HTMLElement>('[data-key]')].flatMap((r) => cells(r));
    expect(shown.length).toBeGreaterThan(5);
    expect(shown).toEqual(want);
    expect(cells(row('Base from Gear'))).toEqual(['90']);
    expect(issued).toEqual(['CalcsMods()']);

    // the mod cells follow a recalculation
    const set = await worker2Instance.SetConfigOption('resistancePenalty', 0);
    poe2Build.set(set);
    await vi.waitFor(() => expect(issued).toEqual(['CalcsMods()', 'CalcsMods()']));
    const resists = within(page.getByRole('region', { name: 'Calculations' })).getByRole('region', { name: 'Resists' });
    await vi.waitFor(() => expect(resists.textContent).toContain('45%'));
    await worker2Instance.SetConfigOption('resistancePenalty', -60);
  }, 120_000);
});

// the values are PoB-PoE2's Skills and Items tabs for the fixture: oracle/poe2_pages.lua on the pinned checkout
describe('the PoE2 Skills and Items pages', () => {
  type Pages = {
    groups: { label: string; gems: { name: string; level: number; quality: number }[] }[];
    items: { slot: string; name: string[]; lines: string[][] }[];
  };
  const pages = async () => JSON.parse(await readFile(resolve(ORACLE, 'poe2_pages_sorceress_items.json'), 'utf8')) as Pages;

  it("lists PoB-PoE2's socket groups and gems, and setting the main group recalculates", async () => {
    const oracle = await pages();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    issued.length = 0;
    const page = render(Poe2Page, { route: 'skills' });
    const list = await vi.waitFor(() => page.getByRole('listbox', { name: 'Socket groups' }));
    expect(
      within(list)
        .getAllByRole('option')
        .map((o) => o.textContent?.trim())
    ).toEqual([`${oracle.groups[0].label} (Active) (FullDPS)`, `${oracle.groups[1].label} (Disabled)`]);
    const gems = () =>
      [...page.container.querySelectorAll<HTMLElement>('[data-gem]')].map((r) => [...r.querySelectorAll('td')].map((c) => c.textContent?.trim()));
    expect(gems().map((g) => g.slice(0, 3))).toEqual(oracle.groups[0].gems.map((g) => [g.name, String(g.level), String(g.quality)]));

    await fireEvent.click(within(list).getAllByRole('option')[1]);
    expect(gems().map((g) => g[0])).toEqual(oracle.groups[1].gems.map((g) => g.name));
    await fireEvent.click(page.getByRole('button', { name: 'Set as main skill' }));
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.mainSocketGroup).toBe(2));
    // the page refetches as soon as the recalculated build is shown
    await vi.waitFor(() => expect(issued).toEqual(['SkillGroups()', 'SetMainSocketGroup(2)', 'SkillGroups()']));
    await vi.waitFor(() => expect(within(list).getAllByRole('option')[1].textContent?.trim()).toBe(`${oracle.groups[1].label} (Disabled) (Forced Active)`));
  }, 120_000);

  it("shows PoB-PoE2's equipped items on the weapon set in use, with their mod lines", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const oracle = await pages();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    const page = render(Poe2Page, { route: 'items' });
    await vi.waitFor(() => page.getByRole('region', { name: 'Body Armour' }));
    const cards = [...page.container.querySelectorAll<HTMLElement>('[data-slot]')];
    expect(cards.map((c) => c.dataset.slot)).toEqual(oracle.items.map((i) => i.slot));
    expect(cards.map((c) => [...c.querySelectorAll('[data-lines]')].map((g) => [...g.children].map((l) => l.textContent)))).toEqual(
      oracle.items.map((i) => i.lines.map((g) => g.map(pobPlain)))
    );
    expect(page.getByLabelText('Weapon set').textContent).toBe('Weapon Set 2');
  }, 120_000);
});
