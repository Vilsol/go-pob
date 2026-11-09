// @vitest-environment jsdom
// CompareTab.lua (PoB-PoE2) over the shared Compare page and the real PoE2 engine

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';
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

// jsdom has no ResizeObserver; the config view's columns follow the panel's width
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

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
const fixture = (name: string) => readFile(resolve(FIXTURES, name), 'utf8');

const { worker2Instance } = await import('$lib/go/sync_worker2');
// the PoE2 worker's calls in the order the page issues them
const issued: string[] = [];
const poe2Worker = vi.fn(() =>
  Promise.resolve(
    new Proxy(worker2Instance, {
      get: (target, prop, receiver) => {
        const v = Reflect.get(target, prop, receiver) as unknown;
        if (typeof v !== 'function') return v;
        return (...args: unknown[]) => {
          issued.push(String(prop));
          return (v as (...a: unknown[]) => unknown).apply(target, args);
        };
      }
    })
  )
);
vi.mock('$lib/go/worker2', () => ({ poe2Worker }));
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
        readdir: () => Promise.resolve([]),
        rm: () => Promise.resolve()
      }
    }
  };
});
// the PoE1 worker lists and reads the saved builds; the build list holds one build of each game
const saved: Record<string, string> = {};
const syncWrap = {
  ListBuilds: vi.fn((dir: string) =>
    Promise.resolve(
      dir === '' || dir === '/'
        ? [
            { Name: 'Three spells', Type: 'file', Class: 'Sorceress', Level: 1, Game: 'poe2', LastEdit: '2026-01-01T00:00:00Z' },
            { Name: 'A PoE1 build', Type: 'file', Class: 'Witch', Level: 90, Game: '', LastEdit: '2026-01-01T00:00:00Z' }
          ]
        : []
    )
  ),
  OnBuildsChanged: vi.fn(() => Promise.resolve()),
  ReadBuild: vi.fn((path: string) => Promise.resolve(saved[path]))
};
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));

const { poe2Build } = await import('$lib/global');
const { summaryStats2 } = await import('./compare_poe2');
const AppCompare = (await import('./AppCompare.svelte')).default;

beforeAll(async () => {
  if (!existsSync(WASM2)) throw new Error(`REFUSING: the PoE2 wasm module is missing — run \`mise run build-go2\`\n  missing: ${WASM2}`);
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
  await worker2Instance.boot(new Uint8Array(wasm) as unknown as ArrayBuffer);
  saved['/Three spells'] = await fixture('sorceress_three_spells.xml');
}, 120_000);

// a build code: zlib, then base64url
const encode = (xml: string) => deflateSync(xml).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// the open build as the PoE2 worker last calculated it
const view = () => get(poe2Build) as Poe2Calc | undefined;

const open = async (name: string) => {
  poe2Build.set(await worker2Instance.Open(await fixture(name)));
  issued.length = 0;
  return render(AppCompare);
};

const compareCode = async (page: Awaited<ReturnType<typeof open>>, code: string) => {
  const section = within(await page.findByRole('region', { name: 'Choose a build to compare' }, { timeout: 10_000 }));
  await fireEvent.input(section.getByRole('textbox', { name: 'Comparison build code' }), { target: { value: code } });
  await fireEvent.click(section.getByRole('button', { name: 'Compare' }));
  return section;
};

// each summary row as "label: A a, B b"
const rows = (container: HTMLElement) =>
  [...within(container).getByRole('region', { name: 'Stat comparison' }).querySelectorAll<HTMLElement>('[role="group"][aria-label*=": A "]')].map((r) =>
    /^(.*): A (.*), B (.*?)(?:, [AB] better|, equal)$/.exec(r.getAttribute('aria-label')!)!.slice(1, 4)
  );

// PoB-PoE2's own calculation of a fixture: oracle/poe2_sidebar.lua's output and skill flags
const oracle = async (name: string) => {
  const o = JSON.parse(await readFile(resolve(ORACLE, `poe2_sidebar_${name}.json`), 'utf8')) as {
    output: Record<string, unknown>;
    skillFlags: Record<string, boolean>;
  };
  const output = Object.fromEntries(Object.entries(o.output).filter((e): e is [string, number] => typeof e[1] === 'number'));
  return { output, skillFlags: o.skillFlags } as unknown as Poe2Calc;
};

describe('the PoE2 Compare page', () => {
  it('compares a build code: each side shows its own calculation, as PoB-PoE2 calculates each build alone', async () => {
    const page = await open('sorceress_spark.xml');
    await compareCode(page, encode(await fixture('warrior_dual_mace.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    const want = summaryStats2(await oracle('sorceress_spark'), await oracle('warrior_dual_mace'));
    const expected = want.a.flatMap((g, gi) => g.map((s, si) => [s.label, s.text, want.b[gi][si].text]));
    expect(expected.length).toBeGreaterThan(10);
    expect(expected.filter(([, a, b]) => a !== b).length).toBeGreaterThan(10);
    await waitFor(() => expect(rows(page.container)).toEqual(expected));
    // the open build's own sidebar figures are not moved by the comparison
    expect(view()?.output.Life).toBe((await oracle('sorceress_spark')).output.Life);
    expect(issued).toContain('LoadCompare');
  });

  it('refuses a PoE1 build code and a code that does not decode', async () => {
    const page = await open('sorceress_spark.xml');
    const section = await compareCode(page, encode('<PathOfBuilding><Build level="1"/></PathOfBuilding>'));
    expect((await section.findByText(/^Comparison failed:/)).textContent).toBe(
      'Comparison failed: A Path of Exile 1 build cannot be compared with a Path of Exile 2 build.'
    );
    await fireEvent.input(section.getByRole('textbox', { name: 'Comparison build code' }), { target: { value: 'not-a-code' } });
    await fireEvent.click(section.getByRole('button', { name: 'Compare' }));
    await waitFor(() =>
      expect(section.getByText(/^Comparison failed:/).textContent).toBe('Comparison failed: this is not a Path of Building build code: it could not be decoded')
    );
    expect(issued).not.toContain('LoadCompare');
  });

  it('compares a saved PoE2 build, listing only the PoE2 builds', async () => {
    const page = await open('sorceress_spark.xml');
    await fireEvent.click(await page.findByRole('button', { name: 'Choose a saved build…' }));
    const list = await page.findByRole('listbox', { name: /^Builds in/ });
    await waitFor(() =>
      expect(
        within(list)
          .getAllByRole('option')
          .map((o) => o.textContent?.split('Sorceress')[0].trim())
      ).toEqual(['Three spells'])
    );
    await fireEvent.dblClick(within(list).getByRole('option'));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    expect(syncWrap.ReadBuild).toHaveBeenCalledWith('/Three spells');
    expect(page.getByRole('combobox', { name: 'Compare with' }).textContent).toBe('Three spells');
  });

  it("sets the compared build's config apart from the open build's, and uses its value", async () => {
    const page = await open('sorceress_three_spells.xml');
    await compareCode(page, encode(await fixture('sorceress_spark.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    await fireEvent.click(page.getByRole('tab', { name: 'Config' }));
    const b = await page.findByRole('combobox', { name: 'Elemental Resistance penalty: (B)' });
    const aBefore = view()?.output.FireResist;
    await fireEvent.change(b, { target: { value: '0' } });
    // PoB-PoE2: sorceress_spark's Fire Resistance is 45 with no resistance penalty
    await waitFor(() => expect(worker2Instance.CompareSides()?.b.view.output.FireResist).toBe(45));
    expect(view()?.output.FireResist).toBe(aBefore);
    const row = (await page.findByRole('combobox', { name: 'Elemental Resistance penalty: (A)' })).closest<HTMLElement>('[role="row"]')!;
    await fireEvent.click(within(row).getByRole('button', { name: 'Use B’s value' }));
    await waitFor(() => expect(view()?.configInputs.resistancePenalty).toBe(0));
    expect(issued).toEqual(expect.arrayContaining(['SetCompareConfigOption', 'UseCompareConfig']));
  });

  // CompareTab.lua:4583-4652 (PoB-PoE2); the cell texts are PoB-PoE2's (oracle/poe2_calcs.lua)
  it("shows each build's Calcs values side by side, and only the differences when asked", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const { calcSections2 } = await import('$lib/calcs/calc_sections2');
    const { elementsOf } = await import('$lib/display/calcs');
    const cells = async (name: string) =>
      (JSON.parse(await readFile(resolve(ORACLE, `poe2_calcs_${name}.json`), 'utf8')) as { cells: Record<string, string> }).cells;
    const [ca, cb] = [await cells('sorceress_spark'), await cells('warrior_dual_mace')];
    const life = calcSections2.find((sec) => sec[1] === 'Life')!;
    const want = elementsOf<{ label?: string }>(life[4][0].data)
      .map((row, k) => [row.label, ca[`Life:0:${k}:0`], cb[`Life:0:${k}:0`]])
      .filter((r): r is [string, string, string] => r.every((x) => x !== undefined))
      .map((r) => r.map(pobPlain));
    expect(want.length).toBeGreaterThan(3);
    expect(want.some(([, a, b]) => a !== b)).toBe(true);

    const page = await open('sorceress_spark.xml');
    await compareCode(page, encode(await fixture('warrior_dual_mace.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    await fireEvent.click(page.getByRole('tab', { name: 'Calcs' }));
    const only = await page.findByRole('checkbox', { name: 'Show only differences' });
    const shown = () =>
      [...page.container.querySelectorAll<HTMLElement>('[role="table"]')]
        .filter((t) => t.querySelector('[role="columnheader"]')?.textContent === 'Life')
        .flatMap((t) => [...t.querySelectorAll<HTMLElement>('[role="row"]')])
        .map((r) => [...r.querySelectorAll<HTMLElement>('[role="rowheader"], [role="cell"]')].map((c) => c.textContent?.trim() ?? ''))
        .filter((r) => r.length === 3 && !r[0].endsWith(':'));
    await waitFor(() => expect(shown().length).toBeGreaterThan(0), { timeout: 10_000 });
    const differing = shown().length;
    await fireEvent.click(only);
    await waitFor(() => expect(shown().length).toBeGreaterThan(differing));
    for (const row of want) expect(shown()).toContainEqual(row);
    expect(issued).toContain('CompareCalcsPages');
    expect(issued).toContain('CompareModsEqual');
    // CompareCalcsHelpers.lua:179-313 (PoB-PoE2): a row's tooltip lists each build's mod rows
    const lifeRow = [...page.container.querySelectorAll<HTMLElement>('[role="table"]')]
      .filter((t) => t.querySelector('[role="columnheader"]')?.textContent === 'Life')
      .flatMap((t) => [...t.querySelectorAll<HTMLElement>('[role="row"][tabindex]')])
      .find((r) => /^(Base|Total)/.test(r.querySelector('[role="rowheader"]')?.textContent?.trim() ?? ''))!;
    expect(lifeRow).toBeTruthy();
    await fireEvent.mouseEnter(lifeRow);
    await waitFor(() => expect(document.getElementById('compare-calcs-tip')?.textContent).toMatch(/Common:|only:/));
    expect(issued).toContain('CompareCalcsModRows');
    // a row whose value and breakdown match is still shown when its mod rows differ
    const { compareCalcs2 } = await import('./compare_poe2');
    const pages = worker2Instance.CompareCalcsPages()!;
    const sides = [
      { view: view()!, page: pages.a },
      { view: worker2Instance.CompareSides()!.b.view, page: pages.b }
    ] as const;
    const rowCount = async (eq: boolean) =>
      (await compareCalcs2(sides[0], sides[1], true, (sds) => Promise.resolve(sds.map(() => eq)))).flatMap((sec) => sec.blocks.flatMap((b) => b.rows)).length;
    expect(await rowCount(false)).toBeGreaterThan(await rowCount(true));
  });

  // CompareTab.lua (PoB-PoE2) is created with each build: its view lasts while the build stays open
  it('keeps the view across an edit and starts afresh with another build', async () => {
    const page = await open('sorceress_spark.xml');
    await compareCode(page, encode(await fixture('sorceress_three_spells.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    const selected = () => page.getAllByRole('tab', { selected: true }).map((t) => t.textContent);
    await fireEvent.click(page.getByRole('tab', { name: 'Config' }));
    await fireEvent.click(await page.findByRole('button', { name: 'Copy Config from Compare Build' }));
    await waitFor(() => expect(issued).toContain('CopyCompareConfig'));
    await waitFor(() => expect(view()?.configInputs).toEqual(worker2Instance.CompareSides()?.b.view.configInputs));
    expect(selected()).toEqual(['Config']);
    cleanup();
    const again = await open('sorceress_spark.xml');
    expect(await again.findByRole('region', { name: 'Choose a build to compare' })).toBeTruthy();
  });

  // CompareTab.lua:2571-3126 (PoB-PoE2); the rows and impacts are PoB-PoE2's (oracle/poe2_compare_power.lua)
  it("reports what each part of the compared build is worth to the open build, as PoB-PoE2's power report does", async () => {
    const { pobPlain } = await import('$lib/design/pob_text');
    const { signed } = await import('$lib/design/compare');
    const pob = JSON.parse(await readFile(resolve(ORACLE, 'poe2_compare_power.json'), 'utf8')) as Record<
      string,
      { category: string; name: string; var?: string; nodeId?: number; impact: number }[]
    >;
    const want = pob.FullDPS.map((r) => [pobPlain(r.category === 'Tree' ? `Take ${r.name}` : r.name).replace(/\s+/g, ' '), signed(r.impact)]).sort();
    expect(want.length).toBeGreaterThan(4);

    const page = await open('compare_power_primary.xml');
    await compareCode(page, encode(await fixture('compare_power_compared.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    await fireEvent.click(page.getByRole('tab', { name: 'Power report' }));
    const table = await page.findByRole('table', { name: 'Power report rows' });
    const shown = () =>
      [...table.querySelectorAll<HTMLElement>('[role="row"]')]
        .slice(1, -1)
        .map((r) => {
          const cells = r.querySelectorAll<HTMLElement>('[role="cell"]');
          const what = (cells[1].querySelector('[role="button"]')?.textContent ?? '').replace(/\s+/g, ' ').trim();
          return [what, /[+−]\S+/.exec(cells[3].textContent ?? '')?.[0] ?? ''];
        })
        .sort();
    await waitFor(() => expect(shown()).toEqual(want), { timeout: 60_000 });
    expect(issued).toContain('BuildComparePower');

    // the config view's impact column is the same report's config row
    const boss = pob.FullDPS.find((r) => r.var === 'enemyIsBoss')!;
    await fireEvent.click(page.getByRole('tab', { name: 'Config' }));
    const row = (await page.findByRole('combobox', { name: 'Is the enemy a Boss? (A)' })).closest<HTMLElement>('[role="row"]')!;
    await waitFor(() => expect(row.querySelectorAll('[role="cell"]')[3].textContent).toContain(signed(boss.impact)));

    // Take on a tree row allocates the node in the open build
    await fireEvent.click(page.getByRole('tab', { name: 'Power report' }));
    await fireEvent.click((await page.findAllByRole('button', { name: 'Take' }))[0]);
    const node = pob.FullDPS.find((r) => r.category === 'Tree')!.nodeId!;
    await waitFor(() => expect(view()?.tree.alloc).toContain(node));
    expect(issued).toContain('TreeClick');
  });

  // CompareTab.lua:3670-3675, 3856-3977 (PoB-PoE2); rows and Equip differences are PoB-PoE2's (oracle/poe2_compare_power.lua)
  it("lists PoB-PoE2's item and jewel rows, each with the Equip comparison and the item's tooltip", async () => {
    const pob = JSON.parse(await readFile(resolve(ORACLE, 'poe2_compare_power.json'), 'utf8')) as {
      items: { slot: string; label: string; p?: string; c?: string }[];
      equip: Record<string, { delta: Record<string, number> }>;
    };
    const page = await open('compare_power_primary.xml');
    await compareCode(page, encode(await fixture('compare_power_compared.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    await fireEvent.click(page.getByRole('tab', { name: 'Items' }));
    const region = await page.findByRole('region', { name: 'Item differences' });
    const rowOf = (label: string) =>
      [...region.querySelectorAll<HTMLElement>('[role="row"]')].find((r) => r.querySelector('[role="cell"]')?.textContent?.trim() === label);
    await waitFor(() => expect(rowOf('Socket')).toBeTruthy());
    // each row: its label, then each side's item name (PoB-PoE2's "title, base" keeps the title) or Empty
    const cellText = (r: HTMLElement, i: number) => r.querySelectorAll<HTMLElement>('[role="cell"]')[i].textContent ?? '';
    const shownRows = [...region.querySelectorAll<HTMLElement>('[role="row"]')]
      .filter((r) => r.querySelectorAll('[role="cell"]').length === 4)
      .map((r) => [
        cellText(r, 0).trim(),
        cellText(r, 1).includes('Empty') ? 'Empty' : cellText(r, 1),
        cellText(r, 2).includes('Empty') ? 'Empty' : cellText(r, 2)
      ]);
    expect(shownRows.map((r) => r[0])).toEqual(pob.items.map((w) => w.label));
    pob.items.forEach((w, i) => {
      expect(shownRows[i][1]).toContain(w.p?.split(', ')[0] ?? 'Empty');
      expect(shownRows[i][2]).toContain(w.c?.split(', ')[0] ?? 'Empty');
    });
    // Ring 1: B's ring raises the Effective Hit Pool by PoB-PoE2's difference
    const ehp = `Effective Hit Pool +${Math.round(pob.equip['Ring 1'].delta.TotalEHP)}`;
    expect(ehp).toBe('Effective Hit Pool +37');
    await waitFor(() => expect(rowOf('Ring 1')!.querySelectorAll('[role="cell"]')[3].textContent).toContain(ehp));
    expect(issued).toContain('CompareEquipImpact');

    const sapphire = within(rowOf('Socket')!).getByRole('button', { name: /Sapphire of Thought/ });
    await fireEvent.mouseEnter(sapphire);
    await waitFor(() => expect(document.getElementById('compare-item-tip')?.textContent).toContain('12% increased Spell Damage'));
    const tip = document.getElementById('compare-item-tip')!.textContent!;
    expect(tip).toContain('7% increased Cast Speed');
    expect(tip).not.toContain('^x');

    // CompareTab.lua:3494-3659 (PoB-PoE2): the expanded mode shows each side's lines, B's ring's mods marked as B's only
    const { pobPlain } = await import('$lib/design/pob_text');
    const cards = (pob as unknown as { cards: Record<string, { a: string[]; b: string[] }> }).cards['Ring 1'];
    await fireEvent.click(within(region).getByRole('checkbox', { name: 'Expanded mode' }));
    const group = await within(region).findByRole('group', { name: 'Ring 1' });
    await waitFor(() =>
      expect([...group.querySelectorAll('ul')].map((u) => [...u.querySelectorAll('li')].map((l) => l.textContent))).toEqual([
        cards.a.map(pobPlain),
        cards.b.map(pobPlain)
      ])
    );
    expect(issued).toContain('CompareItemCards');
  });

  it("copies the compared build's item and tree into the open build", async () => {
    const page = await open('sorceress_spark.xml');
    await compareCode(page, encode(await fixture('warrior_dual_mace.xml')));
    await page.findByRole('region', { name: 'Stat comparison' }, { timeout: 30_000 });
    await fireEvent.click(page.getByRole('tab', { name: 'Items' }));
    const equip = await page.findAllByRole('button', { name: 'Equip' });
    await fireEvent.click(equip[0]);
    await waitFor(() => expect(worker2Instance.CompareSides()?.a.items.map((i) => i.slot)).toContain('Weapon 1'));
    await fireEvent.click(page.getByRole('tab', { name: 'Tree' }));
    await fireEvent.click(await page.findByRole('button', { name: 'Copy and use' }));
    await waitFor(() => expect(view()?.treeSpecs?.labels.at(-1)).toBe('Default (Compared)'));
    expect(view()?.className).toBe('Warrior');
  });
});
