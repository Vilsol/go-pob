// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';

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

const { poe2Build } = await import('$lib/global');
const AppPoe2Build = (await import('./AppPoe2Build.svelte')).default;

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
      const page = render(AppPoe2Build, { view: get(poe2Build)! });
      const shown = [...page.container.querySelectorAll<HTMLElement>('[data-config]')].map((e) => e.dataset.config!).sort();
      expect(shown).toEqual(oracle.configShown);
    },
    120_000
  );

  it('editing a config option on the page recalculates the sidebar', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    issued.length = 0;
    const page = render(AppPoe2Build, { view: get(poe2Build)! });
    const stat = (name: string) => page.container.querySelector<HTMLElement>(`[data-stat="${name}"]`)?.textContent?.trim();
    expect(stat('FireResist')).toBe('-15%');
    const pick = page.container.querySelector<HTMLSelectElement>('[data-config="resistancePenalty"]')!;
    pick.value = '0';
    await fireEvent.change(pick);
    await vi.waitFor(() => expect(get(poe2Build)?.output.FireResist).toBe(45));
    expect(issued).toEqual(['SetConfigOption("resistancePenalty",0)']);
    await page.rerender({ view: get(poe2Build)! });
    expect(stat('FireResist')).toBe('45%');
    expect(stat('LightningResist')).toBe('10%');
  }, 120_000);

  it('switching the main skill recalculates the sidebar', async () => {
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_three_spells.xml')));
    issued.length = 0;
    const page = render(AppPoe2Build, { view: get(poe2Build)! });
    const box = within(page.getByRole('complementary', { name: 'Stats' }));
    const stat = (name: string) => page.container.querySelector<HTMLElement>(`[data-stat="${name}"]`)?.textContent?.trim();
    expect(stat('TotalDPS')).toBe('24.9');
    const skill = box.getByRole<HTMLSelectElement>('combobox', { name: 'Main active skill' });
    expect([...skill.options].map((o) => o.text)).toEqual(['Spark', 'Fireball']);
    skill.value = '2';
    await fireEvent.change(skill);
    await vi.waitFor(() => expect(get(poe2Build)?.output.TotalDPS).toBe(32.76875));
    await page.rerender({ view: get(poe2Build)! });
    expect(stat('TotalDPS')).toBe('32.8');

    const group = box.getByRole<HTMLSelectElement>('combobox', { name: 'Main socket group' });
    expect([...group.options].map((o) => o.text)).toEqual(['Spark, Fireball', 'Arc']);
    group.value = '2';
    await fireEvent.change(group);
    await vi.waitFor(() => expect(get(poe2Build)?.output.TotalDPS).toBeCloseTo(25.020454545455, 9));
    expect(issued).toEqual(['SetMainActiveSkill(2)', 'SetMainSocketGroup(2)']);
  }, 120_000);
});
