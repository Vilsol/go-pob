// @vitest-environment jsdom
// PartyTab.lua (PoB-PoE2) over the shared Party page and the real PoE2 engine

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';
import { Blob as NodeBlob } from 'node:buffer';
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
// the PoE1 worker must not be reached from a PoE2 build's party page
const syncWrap = new Proxy(
  {},
  {
    get: (_, prop) => {
      if (prop === 'then') return undefined;
      return () => {
        throw new Error(`the PoE1 worker was called: ${String(prop)}`);
      };
    }
  }
);
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));

const { poe2Build } = await import('$lib/global');
const AppParty = (await import('./AppParty.svelte')).default;

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
  // the import box checks a code with Blob.stream, which jsdom's Blob lacks
  expect(typeof Blob.prototype.stream, 'jsdom grew Blob.stream; drop this swap').toBe('undefined');
  vi.stubGlobal('Blob', NodeBlob);
}, 120_000);

// a build code: zlib, then base64url
const encode = (xml: string) => deflateSync(xml).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

type ImportOracle = { inputs: Record<string, string>; scenarios: { name: string; buffers: Record<string, string> }[] };
type CalcOracle = { receivers: { build: string; without: { player: Record<string, number> }; with: { player: Record<string, number> } }[] };
const oracle = async <T>(name: string) => JSON.parse(await readFile(resolve(ORACLE, name), 'utf8')) as T;

const view = () => get(poe2Build) as Poe2Calc | undefined;

const open = async (name: string) => {
  poe2Build.set(await worker2Instance.Open(await fixture(name)));
  issued.length = 0;
  const page = render(AppParty);
  await page.findByRole('region', { name: 'Party import' }, { timeout: 10_000 });
  return page;
};

const importCode = async (page: Awaited<ReturnType<typeof open>>, code: string) => {
  const imp = within(page.getByRole('region', { name: 'Party import' }));
  await fireEvent.input(imp.getByRole('textbox', { name: /Enter a build code/ }), { target: { value: code } });
  const button = imp.getByRole<HTMLButtonElement>('button', { name: 'Import' });
  await waitFor(() => expect(button.disabled).toBe(false));
  await fireEvent.click(button);
  return imp;
};

describe('the PoE2 Party page', () => {
  it("imports a member's build code into every buffer and the calculation, as PoB-PoE2 does", async () => {
    const imports = await oracle<ImportOracle>('poe2_party_import.json');
    const calcs = await oracle<CalcOracle>('poe2_party_giver.json');
    const spark = calcs.receivers.find((r) => r.build === 'warrior_dual_mace.xml')!;
    expect(spark.with.player.CombinedDPS, 'vacuity: the member moves the DPS').not.toBe(spark.without.player.CombinedDPS);
    const page = await open('warrior_dual_mace.xml');
    await importCode(page, encode(imports.inputs.giver));
    const want = imports.scenarios.find((s) => s.name === 'replace all')!.buffers;
    await waitFor(() => expect(worker2Instance.GetParty()?.buffers).toEqual(want), { timeout: 10_000 });
    await waitFor(() => expect(view()?.output.CombinedDPS).toBeCloseTo(spark.with.player.CombinedDPS, 6));
    expect(issued).toContain('ImportPartyMember');
    const panel = await waitFor(() => {
      const p = page.container.querySelector<HTMLElement>('[data-member="0"]');
      expect(p?.dataset.active).toBe('true');
      return p!;
    });
    // switching the member off takes its buffers, and the DPS they gave, away
    await fireEvent.click(within(panel).getByRole('checkbox', { name: 'Warrior active' }));
    await waitFor(() => expect(view()?.output.CombinedDPS).toBeCloseTo(spark.without.player.CombinedDPS, 6));
    expect(worker2Instance.GetParty()?.buffers.Curse).toBe('');
  });

  it('disables the effects and rebuilds them', async () => {
    const calcs = await oracle<CalcOracle>('poe2_party_giver.json');
    const spark = calcs.receivers.find((r) => r.build === 'warrior_dual_mace.xml')!;
    const imports = await oracle<ImportOracle>('poe2_party_import.json');
    const page = await open('warrior_dual_mace.xml');
    const imp = await importCode(page, encode(imports.inputs.giver));
    await waitFor(() => expect(view()?.output.CombinedDPS).toBeCloseTo(spark.with.player.CombinedDPS, 6));
    await fireEvent.click(imp.getByRole('checkbox', { name: 'Disable Party Effects' }));
    await waitFor(() => expect(view()?.output.CombinedDPS).toBeCloseTo(spark.without.player.CombinedDPS, 6));
    await fireEvent.click(imp.getByRole('button', { name: 'Rebuild All' }));
    await waitFor(() => expect(view()?.output.CombinedDPS).toBeCloseTo(spark.with.player.CombinedDPS, 6));
  });

  it('turns on Export Support and shows what PoB-PoE2 exports, which another build imports', async () => {
    const exports = await oracle<Record<string, { name: string; text: string }[]>>('poe2_party_export.json');
    const want = exports['party_giver.xml'];
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(process.cwd(), '../calculator2/testdata/party_giver.xml'), 'utf8')));
    issued.length = 0;
    const page = render(AppParty);
    const exp = within(await page.findByRole('region', { name: 'Party export' }, { timeout: 10_000 }));
    expect(exp.queryByRole('textbox')).toBeNull();
    await fireEvent.click(exp.getByRole('checkbox', { name: 'Export Support' }));
    await waitFor(() => expect(exp.getAllByRole('textbox').map((t) => t.getAttribute('aria-label'))).toEqual(want.map((p) => `${p.name} export`)), {
      timeout: 10_000
    });
    expect(issued).toContain('SetExportParty');
    const aura = exp.getByRole<HTMLTextAreaElement>('textbox', { name: 'Aura export' }).value;
    expect(aura).toContain('Purity of Fire\n120\n36|Skill:PurityOfFirePlayer|FireResist|BASE|-|-|type=GlobalEffect/effectType=Aura');
    const code = encode(worker2Instance.Serialize());
    cleanup();
    const receiver = await open('warrior_dual_mace.xml');
    await importCode(receiver, code);
    await waitFor(() => expect(worker2Instance.GetParty()?.buffers.Aura).toBe(aura), { timeout: 10_000 });
  });

  it('refuses a PoE1 build code', async () => {
    const page = await open('sorceress_spark.xml');
    const imp = await importCode(page, encode('<PathOfBuilding><Build level="1"/></PathOfBuilding>'));
    await waitFor(() => expect(imp.getByText(/Path of Exile 1 build cannot join/)).toBeTruthy());
    expect(worker2Instance.GetPartyMembers()).toEqual([]);
  });
});
