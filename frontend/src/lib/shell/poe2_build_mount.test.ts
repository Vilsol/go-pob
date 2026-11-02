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
const AppConfig = (await import('./AppConfig.svelte')).default;
const AppCalcs = (await import('./AppCalcs.svelte')).default;
const AppSkills = (await import('./AppSkills.svelte')).default;
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
      // PoE1's config sets, custom modifiers and reset have no PoE2 counterpart yet
      expect(page.queryByRole('region', { name: 'Custom modifiers' })).toBeNull();
      expect(page.queryByRole('button', { name: 'Reset to defaults' })).toBeNull();
      expect(page.queryByText('Config set')).toBeNull();
      expect(issued).toEqual([]);
    },
    120_000
  );

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
      const calc = await worker2Instance.Open(await fixture(`${name}.xml`));
      const page = worker2Instance.CalcsPage();
      const output: Record<string, unknown> = { ...calc.output, ...page.flags, ...page.strings };
      if (page.mainHand) output.MainHand = page.mainHand;
      if (page.offHand) output.OffHand = page.offHand;
      const actor = { output: output as never, skillFlags: calc.skillFlags, skillData: calc.skillData, mods: page.mods };
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
    expect(issued).toEqual(['CalcsPage()']);
  }, 120_000);

  // CalcsTab.lua (PoB-PoE2) has no calculation of its own: its socket group and active skill are the sidebar's
  it("picks the main group and active skill the sidebar's way, without PoE1's calculation mode", async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_three_spells.xml')));
    issued.length = 0;
    const page = render(AppCalcs);
    await vi.waitFor(() => within(page.getByRole('region', { name: 'Calculations' })).getByRole('region', { name: 'Life' }), { timeout: 60_000 });
    const group = within(page.container).getByRole<HTMLSelectElement>('combobox', { name: 'Socket Group' });
    const skill = within(page.container).getByRole<HTMLSelectElement>('combobox', { name: 'Active Skill' });
    expect([...group.options].map((o) => o.text)).toEqual(['Spark, Fireball', 'Arc']);
    expect([...skill.options].map((o) => o.text)).toEqual(['Spark', 'Fireball']);
    expect(page.queryByRole('combobox', { name: 'Calculation Mode' })).toBeNull();
    expect(page.queryByRole('button', { name: 'Mod Debug' })).toBeNull();

    skill.value = '2';
    await fireEvent.change(skill);
    await vi.waitFor(() => expect(get(poe2Build)?.output.TotalDPS).toBe(32.76875));
    group.value = '1';
    await fireEvent.change(group);
    await vi.waitFor(() => expect((get(poe2Build) as Poe2Calc | undefined)?.mainSocketGroup).toBe(2));
    await vi.waitFor(() => expect(group.value).toBe('1'));
    expect(issued.filter((c) => c !== 'CalcsPage()')).toEqual(['SetMainActiveSkill(2)', 'SetMainSocketGroup(2)']);
    await vi.waitFor(() => expect(issued.filter((c) => c === 'CalcsPage()')).toHaveLength(3));
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
    expect(issued).toEqual(['CalcsPage()']);

    // the mod cells follow a recalculation
    const set = await worker2Instance.SetConfigOption('resistancePenalty', 0);
    poe2Build.set(set);
    await vi.waitFor(() => expect(issued).toEqual(['CalcsPage()', 'CalcsPage()']));
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
      'gem',
      'gem',
      'level',
      'gemEnabled',
      'gemEnabled'
    ]);
  }, 180_000);

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
