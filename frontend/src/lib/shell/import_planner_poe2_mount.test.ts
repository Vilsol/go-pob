// @vitest-environment jsdom
// ImportTab.lua:354-476 (PoB-PoE2): Export to in-game build planner over the Import / Export dialog and the real PoE2 engine

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';

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

const WASM2 = resolve(process.cwd(), 'static/go-pob2.wasm');
const FIXTURES = resolve(process.cwd(), '../wasm2/exposition/testdata');
const ORACLE = resolve(process.cwd(), '../testdata/oracle');
const fixture = (name: string) => readFile(resolve(FIXTURES, name), 'utf8');
// BuildExportPoE2.BuildTable over the fixture, as oracle/poe2_build_planner.lua records it
type PlannerCase = { specIndex: number; skillSetId: number; itemSetId: number; meta: number; path: string; build: Record<string, unknown> };
const pob2Cases = async (name: string) =>
  (JSON.parse(await readFile(resolve(ORACLE, `poe2_build_planner_${name}.json`), 'utf8')) as { cases: PlannerCase[] }).cases;

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
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));
// the file the browser would save
const files: { name: string; json: string }[] = [];
vi.mock('./import_export', async (orig) => ({
  ...(await orig<typeof import('./import_export')>()),
  saveBuildFile: (name: string, json: string) => files.push({ name, json })
}));

const { poe2Build, poe2Enabled } = await import('$lib/global');

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
  poe2Build.set(await worker2Instance.Open(await fixture('build_planner.xml')));
}, 120_000);

// the passives and the slots in the order the oracle sorts them; PoB-PoE2 walks both with pairs()
const sorted = (build: Record<string, unknown>) => {
  type Entry = { id?: string; inventory_id?: string; slot_x?: number };
  const key = (v: Entry) => `${v.id ?? v.inventory_id ?? ''}:${v.slot_x ?? 0}`;
  const by = (a: Entry, b: Entry) => (key(a) < key(b) ? -1 : key(a) > key(b) ? 1 : 0);
  return {
    ...build,
    passives: [...(build.passives as Entry[])].sort(by),
    inventory_slots: [...(build.inventory_slots as Entry[])].sort(by)
  };
};

describe('Export to in-game build planner, with a PoE2 build open', () => {
  it('saves the picked tree, skill set and item set as the .build file PoB-PoE2 writes', async () => {
    (await import('./import_modal.svelte')).openImport('export');
    const view = render((await import('./AppImportModal.svelte')).default);
    const box = within(await view.findByRole('region', { name: 'Build planner export' }));
    await waitFor(() => expect(box.getByRole<HTMLSelectElement>('combobox', { name: 'Skill' }).options.length).toBe(2));
    await fireEvent.input(box.getByRole('textbox', { name: 'Build name' }), { target: { value: 'Planner build' } });
    await fireEvent.input(box.getByRole('textbox', { name: 'Author name' }), { target: { value: 'Someone' } });
    await fireEvent.input(box.getByRole('textbox', { name: 'Description' }), { target: { value: 'Line one {braces}' } });
    await fireEvent.click(box.getByRole('checkbox', { name: 'Generated text for empty item notes' }));
    await fireEvent.change(box.getByRole('combobox', { name: 'Tree' }), { target: { value: '2' } });
    await fireEvent.change(box.getByRole('combobox', { name: 'Skill' }), { target: { value: '2' } });
    await fireEvent.change(box.getByRole('combobox', { name: 'Item' }), { target: { value: '2' } });
    await fireEvent.click(box.getByRole('button', { name: 'Export Selected Sets' }));
    await waitFor(() => expect(files).toHaveLength(1));
    const want = (await pob2Cases('build_planner')).find((c) => c.specIndex === 2 && c.skillSetId === 2 && c.itemSetId === 2 && c.meta === 2)!;
    expect(files[0].name).toBe(want.path);
    expect(sorted(JSON.parse(files[0].json) as Record<string, unknown>)).toEqual(sorted(want.build));
    expect(box.getByText(`Build file exported: ${want.path}`)).toBeTruthy();
  });

  it("defaults the name and author to the fields' placeholders, as GetBuildPlannerMetadata does", async () => {
    files.length = 0;
    (await import('./import_modal.svelte')).openImport('export');
    const view = render((await import('./AppImportModal.svelte')).default);
    const box = within(await view.findByRole('region', { name: 'Build planner export' }));
    await waitFor(() => expect(box.getByRole<HTMLSelectElement>('combobox', { name: 'Tree' }).options.length).toBe(2));
    await fireEvent.click(box.getByRole('button', { name: 'Export Selected Sets' }));
    await waitFor(() => expect(files).toHaveLength(1));
    expect(files[0].name).toBe('Unnamed Build [0.5].build');
    expect(JSON.parse(files[0].json)).toMatchObject({ name: 'Unnamed Build', author: 'Author', description: '' });
  });
});
