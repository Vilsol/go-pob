// @vitest-environment jsdom
// ImportTab.lua:286-294, 312 (PoB-PoE2): "Import as comparison" over the Import / Export dialog and the real PoE2 engine

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';
import { get } from 'svelte/store';

afterEach(async () => {
  cleanup();
  (await import('./import_modal.svelte')).importDialog.open = false;
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

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

const goto = vi.fn((url: string) => Promise.resolve(void url));
vi.mock('$app/navigation', () => ({ goto }));

const WASM2 = resolve(process.cwd(), 'static/go-pob2.wasm');
const FIXTURES = resolve(process.cwd(), '../wasm2/exposition/testdata');
const ORACLE = resolve(process.cwd(), '../testdata/oracle');
const fixture = (name: string) => readFile(resolve(FIXTURES, name), 'utf8');
// PoB-PoE2's own calculation of a fixture (oracle/poe2_sidebar.lua)
const pob2Life = async (name: string) =>
  (JSON.parse(await readFile(resolve(ORACLE, `poe2_sidebar_${name}.json`), 'utf8')) as { output: { Life: number } }).output.Life;

const { worker2Instance } = await import('$lib/go/sync_worker2');
const poe2Worker = vi.fn(() => Promise.resolve(worker2Instance));
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
// the PoE1 worker is not reached by a PoE2 comparison
const syncWrap = { LoadCompareCode: vi.fn(() => Promise.reject(new Error('the PoE1 worker was asked'))) };
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));

const { game } = await import('$lib/go/engine');
const { poe2Build, poe2Enabled } = await import('$lib/global');
const { compareLoaded } = await import('./import_modal.svelte');

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
  await worker2Instance.boot(new Uint8Array(await readFile(WASM2)) as unknown as ArrayBuffer);
  poe2Enabled.set(true);
  poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
}, 120_000);

const encode = (xml: string) => deflateSync(xml).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

const importAs = async (code: string) => {
  (await import('./import_modal.svelte')).openImport('import');
  const view = render((await import('./AppImportModal.svelte')).default);
  const box = within(within(view.getByRole('dialog', { name: 'Import / Export' })).getByRole('region', { name: 'Build import' }));
  await fireEvent.input(box.getByRole('textbox', { name: /^To import a build/ }), { target: { value: code } });
  await fireEvent.click(box.getByRole('radio', { name: /^Import as a comparison/ }));
  await fireEvent.click(box.getByRole('button', { name: /^Import$/ }));
  return box;
};

describe('Import as a comparison, with a PoE2 build open', () => {
  it('loads a PoE2 code beside the open build in the PoE2 worker and opens the Compare page', async () => {
    expect(get(game)).toBe('poe2');
    const seq = compareLoaded.seq;
    await importAs(encode(await fixture('warrior_dual_mace.xml')));
    await waitFor(() => expect(goto).toHaveBeenCalledWith(expect.stringMatching(/\/compare$/)), { timeout: 30_000 });
    expect(compareLoaded.seq).toBe(seq + 1);
    const info = worker2Instance.GetCompareInfo();
    expect(info.names).toEqual(['Imported comparison']);
    expect(worker2Instance.CompareSides()?.b.view.output.Life).toBe(await pob2Life('warrior_dual_mace'));
    expect(syncWrap.LoadCompareCode).not.toHaveBeenCalled();
  });

  it('refuses a PoE1 build code as the Compare page does', async () => {
    goto.mockClear();
    const box = await importAs(encode('<PathOfBuilding><Build level="1"/></PathOfBuilding>'));
    expect((await box.findByText(/cannot be compared/)).textContent).toContain('A Path of Exile 1 build cannot be compared with a Path of Exile 2 build.');
    expect(goto).not.toHaveBeenCalled();
    expect(worker2Instance.GetCompareInfo().names).toEqual(['Imported comparison']);
  });
});
