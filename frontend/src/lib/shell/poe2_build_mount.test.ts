// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, onTestFinished, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import type { Poe2Calc } from '$lib/go/sync_worker2';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

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
// a worker call named here answers that many ms late, as a busy worker would
const delays: Record<string, number> = {};
const poe2Worker = vi.fn(() =>
  Promise.resolve(
    new Proxy(worker2Instance, {
      get: (target, prop, receiver) => {
        const v = Reflect.get(target, prop, receiver) as unknown;
        if (typeof v !== 'function') return v;
        return (...args: unknown[]) => {
          // a line's verdict is a read, which the Config page asks for whenever it shows lines
          if (prop !== 'ParseModLines') issued.push(`${String(prop)}(${args.map((a) => JSON.stringify(a)).join(',')})`);
          const r = (v as (...a: unknown[]) => unknown).apply(target, args);
          const ms = delays[String(prop)];
          return ms ? new Promise((done) => setTimeout(() => done(r), ms)) : r;
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
const AppTree2 = (await import('./AppTree2.svelte')).default;
const AppConfig = (await import('./AppConfig.svelte')).default;
const AppCalcs = (await import('./AppCalcs.svelte')).default;
const AppSkills = (await import('./AppSkills.svelte')).default;
const AppItemsPoe2 = (await import('./AppItemsPoe2.svelte')).default;
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

// the Config page's options in their sections; the Most used group repeats some of them
const shownOptions = (container: HTMLElement) =>
  [...container.querySelectorAll<HTMLElement>('[data-var]')].filter((e) => !e.closest('[aria-label="Most used"]')).map((e) => e.dataset.var!);
const optionRow = (container: HTMLElement, name: string) =>
  [...container.querySelectorAll<HTMLElement>(`[data-var="${name}"]`)].find((e) => !e.closest('[aria-label="Most used"]'));
const openConfig = async (name: string) => {
  poe2Build.set(await worker2Instance.Open(await fixture(`${name}.xml`)));
  issued.length = 0;
  const page = render(AppConfig);
  await vi.waitFor(() => expect(shownOptions(page.container).length).toBeGreaterThan(0));
  return page;
};

beforeAll(async () => {
  if (!existsSync(WASM2)) throw new Error(`REFUSING: the PoE2 wasm module is missing — run \`mise run build-go2\`\n  missing: ${WASM2}`);
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
      const page = await openConfig(name);
      expect(shownOptions(page.container).sort()).toEqual(oracle.configShown);
      // PoE1's reset has no PoE2 counterpart yet
      expect(page.queryByRole('button', { name: 'Reset to defaults' })).toBeNull();
      expect(issued).toEqual([]);
    },
    120_000
  );

  // ConfigTab.lua:167-178, ConfigSetService.lua:17-46 (PoB-PoE2): a new set starts at the defaults and is made
  // active; each set keeps its own options
  it('config sets keep their own options, and switching recalculates', async () => {
    const page = await openConfig('sorceress_spark');
    const setSelect = () => page.getByRole('combobox', { name: 'Config set' }) as HTMLSelectElement;
    expect([...setSelect().options].map((o) => o.textContent)).toContain('Default');
    await worker2Instance.SetConfigOption('resistancePenalty', 0);
    poe2Build.set(await worker2Instance.SetConfigOption('resistancePenalty', 0));
    issued.length = 0;

    await fireEvent.change(setSelect(), { target: { value: '-1' } });
    await fireEvent.click(page.getByRole('button', { name: 'New' }));
    await fireEvent.input(page.getByRole('textbox', { name: 'Name for this config set' }), { target: { value: 'Boss' } });
    await fireEvent.click(page.getByRole('button', { name: 'Save' }));
    await vi.waitFor(() =>
      expect(worker2Instance.ConfigSets()).toEqual({
        sets: [
          { id: 1, title: 'Default' },
          { id: 2, title: 'Boss' }
        ],
        active: 2
      })
    );
    expect(issued).toEqual(['EditConfigSet({"op":"new","title":"Boss"})']);
    await vi.waitFor(() => expect(get(poe2Build)?.output.FireResist).toBe(-15));
    expect(worker2Instance.Serialize()).toMatch(
      /<Config activeConfigSet="2">\s*<ConfigSet id="1" title="Default">(?:(?!<\/ConfigSet>)[\s\S])*<Input name="resistancePenalty" number="0"\/>\s*<\/ConfigSet>\s*<ConfigSet id="2" title="Boss">(?:(?!<\/ConfigSet>)[\s\S])*<\/ConfigSet>\s*<\/Config>/
    );

    issued.length = 0;
    await vi.waitFor(() => expect([...setSelect().options].map((o) => o.textContent)).toContain('Boss'));
    await fireEvent.change(setSelect(), { target: { value: '0' } });
    await vi.waitFor(() => expect(get(poe2Build)?.output.FireResist).toBe(45));
    expect(issued).toEqual(['SelectConfigSet(1)']);
    expect(worker2Instance.ConfigSets().active).toBe(1);
    await worker2Instance.EditConfigSet({ op: 'delete', id: 2 });
    await worker2Instance.SetConfigOption('resistancePenalty', -60);
  }, 120_000);

  // ConfigTab.lua:24-98, 1200-1223 (PoB-PoE2); Life 683 with the fixture's custom block on and 651 with it off is
  // oracle/poe2_sidebar.lua on the fixture and on a copy with the block disabled
  it("custom modifier groups parse their lines, and turning one off recalculates as PoB-PoE2's does", async () => {
    const page = await openConfig('sorceress_spark');
    const custom = await page.findByRole('region', { name: 'Custom modifiers' });
    expect(get(poe2Build)?.output.Life).toBe(683);
    await vi.waitFor(() => expect(within(custom).getByText('1 applied')).toBeTruthy());
    expect(within(custom).queryByRole('button', { name: 'Mod browser' })).toBeNull();
    issued.length = 0;

    await fireEvent.click(within(custom).getByRole('checkbox', { name: 'Enable Default' }));
    await vi.waitFor(() => expect(get(poe2Build)?.output.Life).toBe(651));
    expect(issued).toEqual(['SetCustomModifierBlocks([{"Title":"Default","Enabled":"false","Text":"+30 to maximum Life"}])']);
    expect(worker2Instance.Serialize()).toMatch(/<CustomModifierBlock enabled="false" title="Default">\s*\+30 to maximum Life\s*<\/CustomModifierBlock>/);
    expect(worker2Instance.ParseModLines(['+30 to maximum Life', '# note', '', 'not a modifier']).map((l) => l.Status)).toEqual([
      'parsed',
      'comment',
      'blank',
      'unsupported'
    ]);
    poe2Build.set(await worker2Instance.SetCustomModifierBlocks([{ Title: 'Default', Enabled: 'true', Text: '+30 to maximum Life' }]));
  }, 120_000);

  it('editing a config option on the page recalculates the sidebar', async () => {
    const page = await openConfig('sorceress_spark');
    const side = render(AppSidebar);
    const stat = (name: string) => statOf(side.container, name);
    expect(stat('FireResist')).toBe('-15%');
    const pick = within(optionRow(page.container, 'resistancePenalty')!).getByRole('combobox');
    expect(pick.textContent).toBe('Endgame (-60%)');
    await fireEvent.click(pick);
    await fireEvent.click(within(page.getByRole('listbox')).getByRole('option', { name: 'Act 1 (0%)' }));
    await vi.waitFor(() => expect(get(poe2Build)?.output.FireResist).toBe(45));
    expect(issued).toEqual(['SetConfigOption("resistancePenalty",0)']);
    await vi.waitFor(() => expect(stat('FireResist')).toBe('45%'));
    expect(stat('LightningResist')).toBe('10%');
    await vi.waitFor(() => expect(pick.textContent).toBe('Act 1 (0%)'));
    await worker2Instance.SetConfigOption('resistancePenalty', -60);
  }, 120_000);

  // ConfigTab.lua:207-219 (PoB-PoE2): Show All brings back build-gated options, never parent-gated or legacy ones
  it('showing every option adds the build-gated ones only', async () => {
    const { configurations2, isVariable2 } = await import('$lib/display/configurations2');
    const page = await openConfig('sorceress_spark');
    const before = shownOptions(page.container);
    await fireEvent.click(page.getByRole('checkbox', { name: /Show options that don.t affect this build/ }));
    const after = shownOptions(page.container);
    const rows = configurations2.flatMap((s) => s.rows).filter(isVariable2);
    const gated = rows.filter((r) => r.ifOption !== undefined || r.legacy).map((r) => r.var);
    expect(after.length).toBeGreaterThan(before.length);
    expect(before.every((v) => after.includes(v))).toBe(true);
    expect(after.filter((v) => !before.includes(v) && gated.includes(v))).toEqual([]);
    expect(new Set(after.filter((v) => !before.includes(v)))).toEqual(
      new Set(rows.filter((r) => !before.includes(r.var) && !gated.includes(r.var)).map((r) => r.var))
    );
    expect(issued).toEqual([]);
  }, 120_000);

  // ConfigTab.lua:735-770 (PoB-PoE2): a set option whose condition the build lacks stays, marked invalid
  it('an option set while the build cannot use it shows as invalid', async () => {
    const { configurations2, isVariable2 } = await import('$lib/display/configurations2');
    const oracle = JSON.parse(await readFile(resolve(ORACLE, 'poe2_sidebar_sorceress_spark.json'), 'utf8')) as { configShown: string[] };
    const unused = configurations2
      .flatMap((s) => s.rows)
      .filter(isVariable2)
      .find((r) => r.type === 'check' && !r.hideIfInvalid && r.ifCond !== undefined && !oracle.configShown.includes(r.var))!;
    await worker2Instance.Open(await fixture('sorceress_spark.xml'));
    poe2Build.set(await worker2Instance.SetConfigOption(unused.var, true));
    const page = render(AppConfig);
    const row = await vi.waitFor(() => {
      const found = optionRow(page.container, unused.var);
      if (!found) throw new Error(`${unused.var} is not shown`);
      return found;
    });
    expect(row.className).toContain('text-worse');
    expect(within(row).getByRole<HTMLInputElement>('checkbox').checked).toBe(true);
  }, 120_000);

  it('a check and a number option write through the PoE2 worker', async () => {
    const { options2 } = await import('$lib/display/configurations2');
    const page = await openConfig('sorceress_spark');
    const shown = shownOptions(page.container);
    const check = shown.find((v) => options2[v]?.type === 'check')!;
    const count = shown.find((v) => options2[v]?.type === 'count' && options2[v].defaultState === undefined)!;

    await fireEvent.click(within(optionRow(page.container, check)!).getByRole('checkbox'));
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.configInputs[check]).toBe(true));

    const box = within(optionRow(page.container, count)!).getByRole<HTMLInputElement>('spinbutton');
    await fireEvent.input(box, { target: { value: '5' } });
    await fireEvent.blur(box);
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.configInputs[count]).toBe(5));
    await vi.waitFor(() => expect(box.value).toBe('5'));
    await fireEvent.input(box, { target: { value: '' } });
    await fireEvent.blur(box);
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.configInputs).not.toHaveProperty(count));
    expect(issued).toEqual([
      `SetConfigOption(${JSON.stringify(check)},true)`,
      `SetConfigOption(${JSON.stringify(count)},5)`,
      `SetConfigOption(${JSON.stringify(count)},)`
    ]);
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

// TreeTab.lua:355-364, SkillsTab.lua:594-599 (PoB-PoE2): the tabs' own undo binds the keys only when no whole-build
// history does
const withoutHistory = () => {
  const history = poe2Engine.undo;
  poe2Engine.undo = undefined;
  onTestFinished(() => {
    poe2Engine.undo = history;
  });
};

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
    withoutHistory();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    // the keys reach the page through the tree view; the whole PoE2 tree is slow to mount under jsdom
    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const page = render(AppTree2);
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

// the levels and warnings are PoB-PoE2's Build.lua acts table and EstimatePlayerProgress run under luajit on the
// fixture's point counts
describe('the PoE2 character level', () => {
  beforeAll(() => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it('the level box and the Auto button set the level the build saves with, and Auto follows the points used', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const page = render(AppTree2);
    await vi.waitFor(() => page.getByRole('img', { name: 'Passive skill tree' }), { timeout: 60_000 });
    treeData.mockRestore();
    const points = page.getByRole('status', { name: 'Points used' });
    expect(points.textContent?.replace(/\s+/g, ' ').trim()).toBe('Points 0 / 123 · Weapon set 1 0 / 24 · Weapon set 2 0 / 24 · Ascendancy 0 / 8');
    expect(points.getAttribute('title')).toBe('Required Level: 1\nEstimated Progress:\nAct: Act 1\nExtra Skillpoints: 0');
    const level = page.getByRole('spinbutton', { name: 'Character level' }) as HTMLInputElement;
    expect(level.value).toBe('40');
    issued.length = 0;

    await fireEvent.input(level, { target: { value: '55' } });
    await fireEvent.blur(level);
    await vi.waitFor(() => expect(get(poe2Build)?.level).toBe(55));
    expect(worker2Instance.Serialize()).toMatch(/characterLevelAutoMode="false"[^>]* level="55"/);

    await fireEvent.click(page.getByRole('button', { name: 'Manual' }));
    await vi.waitFor(() => expect(get(poe2Build)?.level).toBe(1));
    expect(page.getByRole('button', { name: 'Auto' }).getAttribute('aria-pressed')).toBe('true');
    expect(worker2Instance.Serialize()).toMatch(/characterLevelAutoMode="true"[^>]* level="1"/);
    expect(issued).toEqual(['SetLevel(55)', 'SetLevelAutoMode(true)']);

    poe2Build.set((await worker2Instance.TreeClick(44871, false, 0, false, 1)).calc);
    expect(get(poe2Build)?.level).toBe(2);
    expect(get(poe2Engine.sidebar)?.warnings).toContainEqual({ kind: 'points', text: 'You have 1 Weapon set 2 passives available' });
    await worker2Instance.TreeUndo(false);
  }, 120_000);

  // TreeTab.lua:145-161 (PoB-PoE2)
  it('Reset Tree asks first, then deallocates every node but the class start', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const page = render(AppTree2);
    await vi.waitFor(() => page.getByRole('img', { name: 'Passive skill tree' }), { timeout: 60_000 });
    treeData.mockRestore();
    poe2Build.set((await worker2Instance.TreeClick(44871, false, 0, false, 0)).calc);
    expect((get(poe2Build) as Poe2Calc | undefined)?.tree.alloc).toEqual([44871, 54447]);
    issued.length = 0;

    await fireEvent.click(page.getByRole('button', { name: 'Reset Tree' }));
    const dialog = page.getByRole('dialog', { name: 'Reset Tree' });
    expect(dialog.textContent).toContain('Warning: resetting your passive tree cannot be undone.');
    await fireEvent.click(within(dialog).getByRole('button', { name: 'Reset' }));
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.tree.alloc).toEqual([54447]));
    expect(issued).toEqual(['TreeReset()']);
    expect(worker2Instance.Serialize()).toMatch(/nodes="54447"/);
  }, 120_000);
});

// the trees and nodes are PoB-PoE2's tree list and class dropdown driven under luajit:
// wasm2/exposition/testdata/tree_specs_oracle.lua over tree_specs.xml
describe('the PoE2 tree list and class dropdown', () => {
  beforeAll(() => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it('Manage trees adds a tree, the tree set switches to it, and Connect Path keeps the tree on a class change', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('tree_specs.xml')));
    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const page = render(AppTree2);
    await vi.waitFor(() => expect((page.getByRole('combobox', { name: 'Class' }) as HTMLSelectElement).value).toBe('Warrior'), { timeout: 60_000 });
    treeData.mockRestore();
    issued.length = 0;

    await fireEvent.keyDown(window, { key: 'm', ctrlKey: true });
    const manager = page.getByRole('dialog', { name: 'Manage passive trees' });
    await fireEvent.click(within(manager).getByRole('button', { name: 'New' }));
    await fireEvent.input(within(manager).getByRole('textbox', { name: 'Name for this passive tree' }), { target: { value: 'Second' } });
    await fireEvent.click(within(manager).getByRole('button', { name: 'Save' }));
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.treeSpecs?.labels).toEqual(['Radius', 'Weapon sets', 'Second']));
    const second = /<Spec [^>]*title="Second"[^>]*>/.exec(worker2Instance.Serialize())?.[0] ?? '';
    expect(second).toContain('classId="6"');
    expect(second).toContain('nodes="47175"');
    await fireEvent.click(within(manager).getByRole('button', { name: 'Done' }));

    await fireEvent.keyDown(window, { key: 'ArrowDown' });
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.treeSpecs?.active).toBe(2));
    expect(get(poe2Build)?.className).toBe('Sorceress');
    await fireEvent.keyDown(window, { key: 'ArrowUp' });
    await vi.waitFor(() => expect(get(poe2Build)?.className).toBe('Warrior'));

    await fireEvent.change(page.getByRole('combobox', { name: 'Class' }), { target: { value: 'Sorceress' } });
    const ask = await page.findByRole('dialog', { name: 'Class Change' });
    expect(ask.textContent).toContain('This can be avoided by connecting one of the Sorceress starting nodes to your tree.');
    await fireEvent.click(within(ask).getByRole('button', { name: 'Connect Path' }));
    await vi.waitFor(() => expect(get(poe2Build)?.className).toBe('Sorceress'));
    expect(/<Spec [^>]*title="Radius"[^>]*>/.exec(worker2Instance.Serialize())?.[0]).toContain(
      'nodes="1207,3041,3936,4739,5710,6744,6839,10156,13397,16725,17468,18407,22419,22558,22616,22928,26725,27373,32194,33556,38323,39581,46325,47555,49357,51052,53405,53697,53921,54447,54811,55473,55933,57703,58838,59795"'
    );
    expect(get(poe2Build)?.output.Life).toBe(1812);
    expect(issued.filter((c) => !c.startsWith('TreeSpecRows('))).toEqual([
      'EditTreeSpecs({"op":"new","title":"Second"})',
      'SetActiveTreeSpec(2)',
      'SetActiveTreeSpec(1)',
      'ChangeTreeClass(7,0)',
      'ChangeTreeClass(7,2)'
    ]);
  }, 120_000);
});

// the values are PoB-PoE2's Calcs tab over its main calculation: oracle/poe2_calcs.lua on the pinned checkout
describe('the PoE2 Calcs page', () => {
  it.each(['sorceress_spark', 'warrior_dual_mace'])(
    "shows PoB-PoE2's cells over the calculation: %s",
    async (name) => {
      const { calcSections2 } = await import('$lib/calcs/calc_sections2');
      const { checkFlag2, formatCalcStr2 } = await import('$lib/display/calcs2');
      const { cellId, elementsOf } = await import('$lib/display/calcs');
      type CalcSectionRow2 = import('$lib/calcs/calc_section_types2').CalcSectionRow2;
      type CalcSectionColumn2 = import('$lib/calcs/calc_section_types2').CalcSectionColumn2;
      const oracle = JSON.parse(await readFile(resolve(ORACLE, `poe2_calcs_${name}.json`), 'utf8')) as {
        cells: Record<string, string>;
        extras: Record<string, string>;
        enabled: Record<string, boolean>;
      };
      await worker2Instance.Open(await fixture(`${name}.xml`));
      const page = worker2Instance.CalcsPage();
      const output: Record<string, unknown> = { ...page.output, ...page.flags, ...page.strings };
      if (page.mainHand) output.MainHand = page.mainHand;
      if (page.offHand) output.OffHand = page.offHand;
      const actor = { output: output as never, skillFlags: page.skillFlags, skillData: page.skillData, mods: page.mods };
      const cells: Record<string, string> = {};
      const extras: Record<string, string> = {};
      const enabled: Record<string, boolean> = {};
      for (const [, id, , , subs] of calcSections2) {
        const on = checkFlag2({ flag: subs[0].data.flag, notFlag: subs[0].data.notFlag }, actor);
        let any = false;
        subs.forEach((sub, j) => {
          elementsOf<CalcSectionRow2>(sub.data).forEach((row, k) => {
            if (!on || !checkFlag2(row, actor)) return;
            any = true;
            elementsOf<CalcSectionColumn2>(row).forEach((col, l) => {
              if (col.format !== undefined && checkFlag2(col, actor)) cells[cellId(id, j, k, l)] = formatCalcStr2(col.format, actor, col, cellId(id, j, k, l));
            });
          });
          if (on && sub.data.extra) extras[`${id}:${j}`] = formatCalcStr2(sub.data.extra, actor);
        });
        enabled[id] = on && any;
      }
      const diff = (a: Record<string, string>, b: Record<string, string>) =>
        [...new Set([...Object.keys(a), ...Object.keys(b)])]
          .filter((k) => a[k] !== b[k])
          .map((k) => `${k} PoB ${JSON.stringify(a[k])} ours ${JSON.stringify(b[k])}`);
      expect(Object.keys(oracle.cells).length).toBeGreaterThan(250);
      expect(enabled).toEqual(oracle.enabled);
      expect(diff(oracle.cells, cells)).toEqual([]);
      expect(diff(oracle.extras, extras)).toEqual([]);
    },
    120_000
  );

  it("shows PoB-PoE2's per-hand ailment rows for a dual wielding attack", async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('warrior_dual_mace.xml')));
    issued.length = 0;
    const page = render(AppCalcs);
    const bleed = await vi.waitFor(() => within(page.getByRole('region', { name: 'Calculations' })).getByRole('region', { name: 'Bleed' }), {
      timeout: 60_000
    });
    const row = (label: string) => [...bleed.querySelectorAll<HTMLElement>('[data-key]')].find((r) => r.textContent?.trim().startsWith(label));
    // Bleed:0:6:0 and Bleed:0:7:0
    await vi.waitFor(() => expect(row('MH Source Physical')?.textContent).toContain('7 to 12'));
    expect(row('OH Source Physical')?.textContent).toContain('8 to 13');
    expect(issued.toSorted()).toEqual(['CalcsInputs()', 'CalcsPage()']);
  }, 120_000);

  // CalcsTab.lua:41-177, 512-513 (PoB-PoE2): the tab calculates its own socket group, active skill and mode, and the
  // sidebar's stay; the values are PoB-PoE2's (oracle/poe2_calcs.lua with the same inputs)
  it('picks its own socket group, active skill and calculation mode, apart from the sidebar', async () => {
    const want = async (name: string) =>
      (JSON.parse(await readFile(resolve(ORACLE, `poe2_calcs_${name}.json`), 'utf8')) as { output: Record<string, number> }).output;
    const sidebar = await worker2Instance.Open(await fixture('sorceress_three_spells.xml'));
    poe2Build.set(sidebar);
    issued.length = 0;
    const page = render(AppCalcs);
    await vi.waitFor(() => within(page.getByRole('region', { name: 'Calculations' })).getByRole('region', { name: 'Life' }), { timeout: 60_000 });
    const group = within(page.container).getByRole<HTMLSelectElement>('combobox', { name: 'Socket Group' });
    const skill = within(page.container).getByRole<HTMLSelectElement>('combobox', { name: 'Active Skill' });
    const mode = within(page.container).getByRole<HTMLSelectElement>('combobox', { name: 'Calculation Mode' });
    expect([...group.options].map((o) => o.text)).toEqual(['Spark, Fireball', 'Arc']);
    expect([...skill.options].map((o) => o.text)).toEqual(['Spark', 'Fireball']);
    expect(page.queryByRole('button', { name: 'Mod Debug' })).toBeNull();

    // Fireball in combat
    skill.value = '2';
    await fireEvent.change(skill);
    mode.value = 'In Combat';
    await fireEvent.change(mode);
    const combat = await want('sorceress_three_spells_fireball_combat');
    await vi.waitFor(() => expect(worker2Instance.CalcsPage().output.TotalDPS).toBe(combat.TotalDPS));
    expect((get(poe2Build) as Poe2Calc | undefined)?.output.TotalDPS).toBe(sidebar.output.TotalDPS);

    // Arc, unbuffed
    group.value = '1';
    await fireEvent.change(group);
    mode.value = 'Unbuffed';
    await fireEvent.change(mode);
    const unbuffed = await want('sorceress_three_spells_arc_unbuffed');
    await vi.waitFor(() => expect(worker2Instance.CalcsPage().output.TotalDPS).toBe(unbuffed.TotalDPS));
    await vi.waitFor(() => expect([...skill.options].map((o) => o.text)).toEqual(['Arc']));
    expect((get(poe2Build) as Poe2Calc | undefined)?.mainSocketGroup).toBe(1);
    expect(issued.filter((c) => c.startsWith('Set'))).toEqual([
      'SetMainActiveSkillCalcs(2)',
      'SetCalcsBuffMode("COMBAT")',
      'SetCalcsSkillNumber(2)',
      'SetCalcsBuffMode("UNBUFFED")'
    ]);
  }, 120_000);

  it("shows PoB-PoE2's Life section cells and recalculates with the build", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const oracle = JSON.parse(await readFile(resolve(ORACLE, 'poe2_calcs_sorceress_spark.json'), 'utf8')) as { cells: Record<string, string> };
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    issued.length = 0;
    const page = render(AppCalcs);
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
    expect(issued.toSorted()).toEqual(['CalcsInputs()', 'CalcsPage()']);

    // the mod cells follow a recalculation
    const set = await worker2Instance.SetConfigOption('resistancePenalty', 0);
    poe2Build.set(set);
    await vi.waitFor(() => expect(issued.filter((c) => c === 'CalcsPage()')).toEqual(['CalcsPage()', 'CalcsPage()']));
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

  const groupButtons = (container: HTMLElement) =>
    [...within(within(container).getByRole('region', { name: 'Socket groups' })).getAllByRole('button')].filter((b) => b.hasAttribute('aria-current'));
  const levels = (container: HTMLElement) =>
    within(within(container).getByRole('region', { name: 'Selected group' }))
      .queryAllByRole<HTMLInputElement>('spinbutton', { name: / level$/ })
      .map((el) => Number(el.value));

  it("lists PoB-PoE2's socket groups and gems on the Skills page, and setting the main group recalculates", async () => {
    const oracle = await pages();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    issued.length = 0;
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(oracle.groups.length));
    const shown = groupButtons(page.container).map((b) => b.textContent ?? '');
    expect(shown.map((t, i) => t.includes(oracle.groups[i].label))).toEqual(oracle.groups.map(() => true));
    expect(levels(page.container)).toEqual(oracle.groups[0].gems.map((g) => g.level));

    await fireEvent.click(groupButtons(page.container)[1]);
    expect(levels(page.container)).toEqual(oracle.groups[1].gems.map((g) => g.level));
    await fireEvent.click(page.getByRole('button', { name: 'Set as main' }));
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.mainSocketGroup).toBe(2));
    // the page refetches as soon as the recalculated build is shown
    await vi.waitFor(() =>
      expect(issued.filter((c) => c.startsWith('SkillGroups') || c.startsWith('SetMain'))).toEqual(['SkillGroups()', 'SetMainSocketGroup(2)', 'SkillGroups()'])
    );
    await vi.waitFor(() => expect(groupButtons(page.container)[1].textContent).toContain('MAIN'));
  }, 120_000);

  // calculator2/testdata/skills_edit_pob.jsonl: PoB-PoE2's sidebar values after the same edits through its Skills tab
  it("edits groups and gems as PoB-PoE2's Skills tab does, and the sidebar values are PoB-PoE2's", async () => {
    const rows = (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
    const after = (step: string) => rows.find((r) => r.step === step)!.output;
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));
    const output = () => (get(poe2Build) as Poe2Calc | undefined)?.output ?? {};
    const settle = async (step: string) =>
      vi.waitFor(() => {
        const want = after(step);
        for (const k of ['TotalDPS', 'ManaCost', 'AverageDamage']) expect([step, k, output()[k]]).toEqual([step, k, want[k]]);
      });

    await fireEvent.click(page.getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await vi.waitFor(() => expect(groupButtons(page.container)[1].getAttribute('aria-current')).toBe('true'));
    await fireEvent.click(page.getByRole('button', { name: 'Set as main' }));
    await settle('main group 2');

    const add = async (name: string, step: string) => {
      const box = page.getByRole('combobox', { name: 'Add a gem' });
      await fireEvent.focus(box);
      await fireEvent.input(box, { target: { value: name } });
      await fireEvent.keyDown(box, { key: 'Enter' });
      await settle(step);
    };
    await add('Fireball', 'add Fireball');
    await add('Elemental Focus', 'add Elemental Focus');
    await vi.waitFor(() => expect(levels(page.container)).toEqual([20, 1]));

    const spin = (name: string) => page.getByRole('spinbutton', { name }) as HTMLInputElement;
    await fireEvent.input(spin('Fireball level'), { target: { value: '15' } });
    await fireEvent.keyDown(spin('Fireball level'), { key: 'Enter' });
    await settle('Fireball level 15');
    await fireEvent.click(page.getAllByRole('checkbox', { name: 'Elemental Focus enabled' })[0]);
    await settle('disable Elemental Focus');
    await fireEvent.click(page.getAllByRole('checkbox', { name: 'Elemental Focus enabled' })[0]);
    await settle('enable Elemental Focus');

    const edits = issued.filter((c) => c.startsWith('EditSkills'));
    expect(edits.map((c) => (JSON.parse(c.slice('EditSkills('.length, -1)) as { op: string }).op)).toEqual([
      'newGroup',
      'addGem',
      'addGem',
      'level',
      'gemEnabled',
      'gemEnabled'
    ]);
  }, 180_000);

  // calculator2/testdata/skills_sets_pob.jsonl: PoB-PoE2's sidebar values after the same skill set edits
  it("adds, copies and switches skill sets as PoB-PoE2's skill set manager does", async () => {
    const rows = (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_sets_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; groups: string[]; output: Record<string, number> });
    const at = (step: string) => rows.find((r) => r.step === step)!;
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppSkills);
    const select = await vi.waitFor(() => page.getByRole('combobox', { name: 'Skill set' }) as HTMLSelectElement);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));
    const settle = async (step: string) =>
      vi.waitFor(() => {
        const out = (get(poe2Build) as Poe2Calc | undefined)?.output ?? {};
        expect([step, out.TotalDPS, groupButtons(page.container).length]).toEqual([step, at(step).output.TotalDPS, at(step).groups.length]);
      });
    const manage = async (button: string, title: string) => {
      await fireEvent.change(select, { target: { value: '-1' } });
      const dialog = await vi.waitFor(() => page.getByRole('dialog', { name: 'Manage skill sets' }));
      await fireEvent.click(within(dialog).getByRole('button', { name: button }));
      const name = within(dialog).getByRole<HTMLInputElement>('textbox', { name: 'Name for this skill set' });
      await fireEvent.input(name, { target: { value: title } });
      await fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }));
      await fireEvent.click(within(dialog).getByRole('button', { name: 'Done' }));
    };

    await manage('New', 'Second');
    await settle('new set Second');
    await vi.waitFor(() => expect([...select.options].map((o) => o.text).slice(0, 2)).toEqual(['Default', 'Second']));
    expect(select.value).toBe('1');
    await fireEvent.change(select, { target: { value: '0' } });
    await settle('load');
    await manage('Copy', 'Copy');
    await settle('copy set 1');
    await vi.waitFor(() => expect(select.value).toBe('2'));
    expect(issued.filter((c) => c.startsWith('EditSkills')).map((c) => (JSON.parse(c.slice('EditSkills('.length, -1)) as { op: string }).op)).toEqual([
      'newSet',
      'activeSet',
      'copySet'
    ]);
  }, 180_000);

  // calculator2/testdata/gem_sort_pob.json: PoB-PoE2's gem list sort for the group's new gem slot, by Combined DPS
  it("sorts the gem list by the DPS each gem gives, as PoB-PoE2's gem list does", async () => {
    const oracle = JSON.parse(await readFile(resolve(process.cwd(), '../calculator2/testdata/gem_sort_pob.json'), 'utf8')) as {
      base: number;
      gems: { id: string; canSupport: boolean; dps: number }[];
    };
    const names = new Map((await worker2Instance.GemList()).map((g) => [g.id, g.name]));
    // GemSelectControl.lua:353-373 (PoB-PoE2)
    const top = oracle.gems
      .filter((g) => g.canSupport)
      .map((g) => ({ name: names.get(g.id)!, dps: g.dps }))
      .sort((a, b) => (a.dps !== b.dps ? b.dps - a.dps : a.name < b.name ? -1 : 1))
      .slice(0, 8);
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8')));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));
    const editor = within(page.getByRole('region', { name: 'Selected group' }));
    await fireEvent.focus(editor.getByRole('combobox', { name: 'Add a gem' }));
    await vi.waitFor(
      () => {
        expect(editor.queryByLabelText('Sort progress')).toBeNull();
        const shown = within(editor.getByRole('listbox', { name: 'Gem results' }))
          .getAllByRole('option')
          .slice(0, top.length)
          .map((o) => o.textContent ?? '');
        expect(shown.map((t, i) => t.trim().startsWith(top[i].name))).toEqual(top.map(() => true));
        const delta = Math.round(((top[0].dps - oracle.base) / oracle.base) * 1000) / 10;
        expect(shown[0]).toContain(`+${delta}%`);
      },
      { timeout: 60_000 }
    );
  }, 180_000);

  // calculator2/testdata/skills_paste_pob.jsonl: the text PoB-PoE2 copies for the oracle's Fire group
  it("Ctrl+C copies a group as PoB-PoE2's text, and pasting it adds the group", async () => {
    const rows = (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_paste_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; text?: string; groups?: string[] });
    const want = rows.find((r) => r.step === 'copied')!.text!;
    const pasted = rows.find((r) => r.step === 'paste the Fire group')!.groups!;
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_paste.xml'), 'utf8')));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    let copied = '';
    const clip = { writeText: vi.fn((t: string) => Promise.resolve(void (copied = t))) };
    Object.defineProperty(navigator, 'clipboard', { value: clip, configurable: true });
    await fireEvent.keyDown(groupButtons(page.container)[1], { key: 'c', ctrlKey: true });
    await vi.waitFor(() => expect(copied).toBe(want));

    const paste = new Event('paste') as ClipboardEvent;
    Object.defineProperty(paste, 'clipboardData', { value: { getData: () => copied } });
    window.dispatchEvent(paste);
    await vi.waitFor(() => expect(groupButtons(page.container).map((b) => pasted.some((l) => b.textContent?.includes(l)))).toEqual(pasted.map(() => true)));
    await vi.waitFor(() => expect(groupButtons(page.container)[2].getAttribute('aria-current')).toBe('true'));
  }, 120_000);

  // calculator2/testdata/skills_edit_generated_pob.jsonl: PoB-PoE2's sidebar values after the same edits
  it("the Corrupt dropdown corrupts a gem as PoB-PoE2's does", async () => {
    const rows = (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit_generated_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
    const after = (step: string) => rows.find((r) => r.step === step)!.output;
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit_generated.xml'), 'utf8')));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    const settle = async (step: string) =>
      vi.waitFor(() => {
        const out = (get(poe2Build) as Poe2Calc | undefined)?.output ?? {};
        for (const k of ['TotalDPS', 'AverageDamage', 'ManaCost']) expect([step, k, out[k]]).toEqual([step, k, after(step)[k]]);
      });
    await fireEvent.click(page.getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(3));
    const box = page.getByRole('combobox', { name: 'Add a gem' });
    await fireEvent.focus(box);
    await fireEvent.input(box, { target: { value: 'Rolling Slam' } });
    await fireEvent.keyDown(box, { key: 'Enter' });
    await settle('add Rolling Slam');
    await fireEvent.click(page.getByRole('button', { name: 'Set as main' }));
    await settle('main group new');
    const corrupt = await vi.waitFor(() => page.getByRole('combobox', { name: 'Rolling Slam corruption' }) as HTMLSelectElement);
    expect([...corrupt.options].map((o) => o.text)).toEqual(['Not Corrupted', '+1 to Gem Level', 'Corrupted', '-1 to Gem Level']);
    expect(corrupt.value).toBe('1');
    await fireEvent.change(corrupt, { target: { value: '2' } });
    await settle('Rolling Slam +1 level corruption');
    await vi.waitFor(() => expect((page.getByRole('combobox', { name: 'Rolling Slam corruption' }) as HTMLSelectElement).value).toBe('2'));
  }, 120_000);

  // SkillListControl.lua:45-54, 139-173 (PoB-PoE2)
  it('deleting a group with gems, or every group, asks first; an empty group goes at once', async () => {
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_paste.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    const deletes = () => issued.filter((c) => c.startsWith('EditSkills')).map((c) => (JSON.parse(c.slice('EditSkills('.length, -1)) as { op: string }).op);

    await fireEvent.click(groupButtons(page.container)[1]);
    await fireEvent.click(page.getByRole('button', { name: 'Delete group' }));
    expect(page.getByRole('group', { name: 'Confirm delete' }).textContent).toContain("Are you sure you want to delete 'Fire'?");
    await fireEvent.click(page.getByRole('button', { name: 'Cancel' }));
    expect(page.queryByRole('group', { name: 'Confirm delete' })).toBeNull();
    expect(deletes()).toEqual([]);
    await fireEvent.click(page.getByRole('button', { name: 'Delete group' }));
    await fireEvent.click(within(page.getByRole('group', { name: 'Confirm delete' })).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));

    await fireEvent.click(page.getByRole('button', { name: '+ New group' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await fireEvent.click(page.getByRole('button', { name: 'Delete group' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(1));
    expect(page.queryByRole('group', { name: 'Confirm delete' })).toBeNull();

    await fireEvent.click(page.getByRole('button', { name: 'Delete all' }));
    expect(page.getByRole('group', { name: 'Confirm delete' }).textContent).toContain('Are you sure you want to delete all socket groups in this build?');
    await fireEvent.click(within(page.getByRole('group', { name: 'Confirm delete' })).getByRole('button', { name: 'Delete' }));
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(0));
    expect(deletes()).toEqual(['deleteGroup', 'newGroup', 'deleteGroup', 'deleteAll']);
  }, 120_000);

  // calculator2/testdata/skills_paste_pob.jsonl: the error message PoB-PoE2 gives a gem name it cannot find
  it("a gem whose name finds no gem shows PoB-PoE2's error message", async () => {
    await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8'));
    poe2Build.set(await worker2Instance.EditSkills({ op: 'paste', text: 'Label: Typed\r\nSpark 5/0  1\r\nNot A Gem 1/0  1\r\n' }));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(2));
    await fireEvent.click(groupButtons(page.container)[1]);
    await vi.waitFor(() =>
      expect([...page.container.querySelectorAll('[data-gem-error]')].map((e) => e.textContent?.trim())).toEqual(["Unrecognised gem name 'Not A Gem'"])
    );
  }, 120_000);

  // calculator2/testdata/weapon_sets_crossbow_plain_pob.json: PoB-PoE2's Set 1 and Set 2 boxes for the build's groups
  it("the weapon set boxes are marked and locked where PoB-PoE2's are", async () => {
    const oracle = JSON.parse(await readFile(resolve(process.cwd(), '../calculator2/testdata/weapon_sets_crossbow_plain_pob.json'), 'utf8')) as {
      sets: { valid: boolean; enabled: boolean }[];
    }[];
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/crossbow_plain.xml'), 'utf8')));
    const page = render(AppSkills);
    await vi.waitFor(() => expect(groupButtons(page.container).length).toBe(oracle.length));
    for (const [i, want] of oracle.entries()) {
      await fireEvent.click(groupButtons(page.container)[i]);
      await vi.waitFor(() => {
        const boxes = [1, 2].map((n) => page.getByRole('checkbox', { name: `Set ${n}` }) as HTMLInputElement);
        expect(boxes.map((b) => [!b.closest('label')?.querySelector('.tone-error'), !b.disabled])).toEqual(want.sets.map((s) => [s.valid, s.enabled]));
      });
    }
    const tip = () => page.getByRole('checkbox', { name: 'Set 2' }).closest('[title]')?.getAttribute('title');
    expect(tip()).toBe('Default attack skills can only be used in the weapon set that grants them.');
    await fireEvent.click(groupButtons(page.container)[0]);
    await vi.waitFor(() => expect(tip()).toBe('This skill cannot be used with the weapons equipped in Set 2.'));
  }, 120_000);

  // calculator2/testdata/skills_undo_pob.jsonl: PoB-PoE2's groups after the same edits, undos and redos
  it("Ctrl+Z and Ctrl+Y on the Skills page undo and redo edits as PoB-PoE2's Skills tab does", async () => {
    withoutHistory();
    const rows = (await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_undo_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; groups: string[] });
    const at = (step: string) => rows.find((r) => r.step === step)!.groups;
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/skills_edit.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppSkills);
    const shown = () => groupButtons(page.container).map((b) => b.textContent ?? '');
    const settle = (step: string) =>
      vi.waitFor(() => expect([step, shown().length, shown().every((t, i) => t.includes(at(step)[i]))]).toEqual([step, at(step).length, true]));
    await settle('load');
    await fireEvent.click(page.getByRole('button', { name: '+ New group' }));
    await settle('new group');
    const box = page.getByRole('combobox', { name: 'Add a gem' });
    await fireEvent.focus(box);
    await fireEvent.input(box, { target: { value: 'Fireball' } });
    await fireEvent.keyDown(box, { key: 'Enter' });
    await settle('add Fireball');
    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await settle('undo Fireball');
    await fireEvent.keyDown(window, { key: 'y', ctrlKey: true });
    await settle('redo Fireball');
    expect(issued.filter((c) => c.startsWith('SkillsUndo'))).toEqual(['SkillsUndo(false)', 'SkillsUndo(true)']);
  }, 120_000);

  // testdata/oracle/poe2_gem_search.json: PoB-PoE2's gem picker list for each search and support filter, sorted by name
  it("searches PoE2's gems as PoB-PoE2's gem picker does", async () => {
    const { populateGems2, searchGems, TERM_ALIASES2 } = await import('$lib/design/gem_search');
    type Case = { showSupport: string; showLegacy: boolean; results: { query: string; names?: string[] }[] };
    const cases = JSON.parse(await readFile(resolve(ORACLE, 'poe2_gem_search.json'), 'utf8')) as Case[];
    const gems = (await worker2Instance.GemList()).map((g) => ({ ...g, searchName: g.name }));
    expect(cases.length).toBe(6);
    for (const c of cases) {
      const listed = populateGems2(gems, { showSupport: c.showSupport, matchLevel: false, characterLevel: 1, showLegacy: c.showLegacy });
      for (const r of c.results) {
        const got = searchGems(listed, r.query, undefined, undefined, TERM_ALIASES2).map((g) => g.name);
        expect([c.showSupport, c.showLegacy, r.query, got]).toEqual([c.showSupport, c.showLegacy, r.query, r.names ?? []]);
      }
    }
  }, 120_000);

  it("shows PoB-PoE2's equipped items on the weapon set in use, with their mod lines", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const oracle = await pages();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    const page = render(AppItemsPoe2);
    const slot = (name: string) => page.getByRole('button', { name: new RegExp(`^${name}: `) });
    await vi.waitFor(() => slot('Body Armour'));
    for (const i of oracle.items.filter((x) => !x.slot.startsWith('Jewel'))) {
      const shown = slot(i.slot.replace(/ Swap$/, ''));
      expect(shown.getAttribute('aria-label')).toBe(`${i.slot.replace(/ Swap$/, '')}: ${pobPlain(i.name[0])}`);
      await fireEvent.mouseEnter(shown);
      const tip = await vi.waitFor(() => page.getByRole('tooltip'));
      for (const line of i.lines.flat()) expect(tip.textContent).toContain(pobPlain(line));
      await fireEvent.mouseLeave(shown);
    }
    expect(
      within(page.getByRole('group', { name: 'Weapon set' }))
        .getByRole('button', { name: 'II' })
        .getAttribute('aria-pressed')
    ).toBe('true');
  }, 120_000);

  it('shows the items of the newest read when an older read, waiting on the open editor, answers last', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    const page = render(AppItemsPoe2);
    await vi.waitFor(() => page.getByRole('button', { name: /^Body Armour: / }));
    const id = worker2Instance.ItemsPage().slots.find((s) => s.name === 'Body Armour')!.itemId;
    worker2Instance.EditDisplayItem({ op: 'edit', id });
    delays.DisplayItemAffixes = 300;
    try {
      poe2Build.set({ ...get(poe2Build)! });
      await vi.waitFor(() => expect(issued.at(-1)).toBe('DisplayItemAffixes()'));
      worker2Instance.EditDisplayItem({ op: 'close' });
      poe2Build.set(await worker2Instance.EditItems({ op: 'delete', id }));
      await new Promise((done) => setTimeout(done, 800));
      expect(
        page
          .getAllByRole('button')
          .map((b) => b.getAttribute('aria-label') ?? b.textContent)
          .filter((n) => /Body Armour/.test(n ?? ''))
      ).toEqual(['Body Armour: Empty']);
    } finally {
      delete delays.DisplayItemAffixes;
    }
  }, 120_000);

  // calculator2/testdata/items_edit_pob.jsonl: PoB-PoE2's sidebar values after the same edits through its Items tab
  it("pastes, equips and edits items as PoB-PoE2's Items tab does, and the sidebar values are PoB-PoE2's", async () => {
    const data = resolve(process.cwd(), '../calculator2/testdata');
    const rows = (await readFile(resolve(data, 'items_edit_pob.jsonl'), 'utf8'))
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
    const pastes = (await readFile(resolve(data, 'items_edit_pastes.txt'), 'utf8')).replace(/\n$/, '').split('\n=====\n');
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(data, 'items_edit.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppItemsPoe2);
    const output = () => (get(poe2Build) as Poe2Calc | undefined)?.output ?? {};
    const settle = (step: string) =>
      vi.waitFor(() => {
        const want = rows.find((r) => r.step === step)!.output;
        for (const k of ['Life', 'Mana', 'Armour', 'FireResist', 'ColdResist', 'TotalDPS']) expect([step, k, output()[k]]).toEqual([step, k, want[k]]);
      });
    const paste = async (text: string, step: string) => {
      const box = await vi.waitFor(() => page.getByRole('textbox', { name: 'Item text' }));
      await fireEvent.input(box, { target: { value: text } });
      await fireEvent.click(await vi.waitFor(() => page.getByRole('button', { name: 'Save to build' })));
      await settle(step);
    };
    await paste(pastes[0], 'paste helmet');
    await paste(pastes[1], 'paste wand');
    await vi.waitFor(() => page.getByRole('button', { name: 'Weapon 1: Doom Song' }));

    const weaponSet = (n: string) => within(page.getByRole('group', { name: 'Weapon set' })).getByRole('button', { name: n });
    await fireEvent.click(weaponSet('II'));
    await settle('weapon set II');
    await fireEvent.click(weaponSet('I'));
    await settle('weapon set I');

    // ItemListControl.lua:321-328, ItemsTab.lua:613-624, 792-800: the body armour opened in the editor takes two runes
    await fireEvent.click(page.getByRole('button', { name: 'Body Armour: Storm Shell' }));
    await fireEvent.click(await vi.waitFor(() => within(page.getByRole('dialog')).getByRole('button', { name: 'Edit as text' })));
    const sockets = await vi.waitFor(() => page.getByRole('spinbutton', { name: 'Sockets' }) as HTMLInputElement);
    await fireEvent.input(sockets, { target: { value: '2' } });
    await fireEvent.keyDown(sockets, { key: 'Enter' });
    const rune = (n: number) => vi.waitFor(() => page.getByRole('combobox', { name: `Rune #${n}` }) as HTMLSelectElement);
    await fireEvent.change(await rune(1), { target: { value: 'Desert Rune' } });
    await vi.waitFor(async () => expect((await rune(1)).value).toBe('Desert Rune'));
    await fireEvent.change(await rune(2), { target: { value: 'Glacial Rune' } });
    await vi.waitFor(async () => expect((await rune(2)).value).toBe('Glacial Rune'));
    await fireEvent.click(page.getByRole('button', { name: 'Save' }));
    await settle('body armour runes');

    const edits = issued.filter((c) => c.startsWith('EditItems'));
    expect(edits.map((c) => (JSON.parse(c.slice('EditItems('.length, -1)) as { op: string }).op)).toEqual(['add', 'add', 'swap', 'swap', 'add']);
  }, 180_000);
});
