// @vitest-environment jsdom
// ConfigTab.lua (PoB-PoE2): the config page's tooltip stat comparisons and option placeholders over the real PoE2
// engine; every expected value is PoB-PoE2's (oracle/poe2_config_whatif.lua, `mise run config-whatif-poe2`)

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

// jsdom has no ResizeObserver; the config view's columns follow the panel's width
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

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
        readdir: (dir: string) => Promise.resolve([...files.keys()].filter((k) => k.startsWith(`${dir}/`)).map((k) => k.slice(dir.length + 1))),
        rm: (path: string) => {
          for (const k of [...files.keys()]) if (k === path || k.startsWith(`${path}/`)) files.delete(k);
          return Promise.resolve();
        }
      }
    }
  };
});
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));

const { poe2Build } = await import('$lib/global');
const AppConfig = (await import('./AppConfig.svelte')).default;
const { pobPlain } = await import('$lib/design/pob_text');

type Oracle = {
  options: { var: string; value: boolean | number | string; lines: string[] }[];
  custom: { group: number; lines: string[] }[];
  placeholders: { inputs: string; placeholders: Record<string, number> }[];
};
const oracle = async () => JSON.parse(await readFile(resolve(ORACLE, 'poe2_config_whatif_sorceress_spark.json'), 'utf8')) as Oracle;
const spark = () => readFile(resolve(FIXTURES, 'sorceress_spark.xml'), 'utf8');

// Build.lua:2605-2612 (PoB-PoE2): a stat line as the tooltip draws it, colour codes dropped
const lineText = (sections: StatCompareSection[] | undefined) =>
  (sections ?? []).flatMap((s) => s.lines.map((l) => `${l.value} ${l.label}${l.percent ? ` ${l.percent}` : ''}`));

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
}, 120_000);

describe('the PoE2 config tooltips and placeholders', () => {
  // ConfigTab.lua:774-838, 62-79 (PoB-PoE2)
  it('every option and custom group comparison PoB-PoE2 shows on the fixture, line for line', async () => {
    const pob = await oracle();
    expect(pob.options.length).toBeGreaterThan(50);
    expect(new Set(pob.options.map((o) => typeof o.value))).toEqual(new Set(['boolean', 'string', 'number']));
    poe2Build.set(await worker2Instance.Open(await spark()));
    for (const o of pob.options) {
      const got = lineText(await worker2Instance.ConfigCompare(o.var, o.value));
      expect(got, `${o.var} = ${String(o.value)}`).toEqual(o.lines.slice(1).map(pobPlain));
    }
    expect(pob.custom.length).toBe(1);
    for (const c of pob.custom) {
      expect(pobPlain(c.lines[0])).toBe('Disabling this group will give you:');
      expect(lineText(await worker2Instance.CustomModGroupCompare(c.group))).toEqual(c.lines.slice(1).map(pobPlain));
    }
  }, 120_000);

  // ConfigTab.lua:1051-1052, ConfigOptions.lua:780-794, 1982-2139, 2189-2241 (PoB-PoE2)
  it("every option's placeholder is PoB-PoE2's, for each enemy preset", async () => {
    const pob = await oracle();
    expect(pob.placeholders.length).toBe(6);
    const xml = await spark();
    for (const p of pob.placeholders) {
      const view = await worker2Instance.Open(xml.replace('<ConfigSet id="1" title="Default">', `<ConfigSet id="1" title="Default">${p.inputs}`));
      expect(view.configPlaceholders, p.inputs).toEqual(p.placeholders);
    }
  }, 120_000);

  it('the page shows the placeholders and asks the worker what an option and a group give', async () => {
    const pob = await oracle();
    poe2Build.set(await worker2Instance.Open(await spark()));
    issued.length = 0;
    const page = render(AppConfig);
    const row = (name: string) => [...page.container.querySelectorAll<HTMLElement>(`[data-var="${name}"]`)].find((e) => !e.closest('[aria-label="Most used"]'));
    await vi.waitFor(() => expect(row('enemyLevel')?.querySelector('input')?.placeholder).toBe('82'), { timeout: 10_000 });
    expect(row('enemyArmour')?.querySelector('input')?.placeholder).toBe(String(pob.placeholders[0].placeholders.enemyArmour));

    const onslaught = pob.options.find((o) => o.var === 'buffOnslaught')!;
    await fireEvent.mouseEnter(row('buffOnslaught')!);
    await vi.waitFor(() => expect(document.querySelector('[data-whatif]')?.textContent).toContain('Toggling this option will give you:'));
    const tip = document.querySelector('[data-whatif]')!.textContent!.replace(/\s+/g, ' ');
    // each line as the page lays it out: the label, then the difference and its percentage
    for (const line of onslaught.lines.slice(1).map(pobPlain)) {
      const [, value, label, percent] = /^(\S+) (.+?)(?: (\(.+\)))?$/.exec(line)!;
      expect(tip).toContain(`${label} ${value}${percent ? ` ${percent}` : ''}`);
    }
    expect(issued).toContain('ConfigCompare');

    const custom = await page.findByRole('region', { name: 'Custom modifiers' });
    await fireEvent.mouseOver(within(custom).getByRole('checkbox', { name: 'Enable Default' }));
    await vi.waitFor(() => expect(issued).toContain('CustomModGroupCompare'));
    await vi.waitFor(() => expect(document.body.textContent).toContain('Disabling this group will give you:'));
  }, 120_000);
});
