// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';
import type { Poe2Calc } from '$lib/go/sync_worker2';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

const WASM2 = resolve(process.cwd(), 'static/go-pob2.wasm');
const FIXTURES = resolve(process.cwd(), '../wasm2/exposition/testdata');

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
        readdir: () => Promise.resolve([]),
        rm: () => Promise.resolve()
      }
    }
  };
});
vi.mock('$lib/go/worker', () => ({ syncWrap: undefined, syncWorker: undefined }));
vi.mock('$lib/components/skill-tree/gl_renderer2', () => ({
  createTree2Renderer: () => ({ kind: 'canvas', setView: () => {}, setState: () => {}, destroy: () => {} })
}));

const { poe2Build } = await import('$lib/global');
const { poe2Engine, poe2Run } = await import('$lib/go/engine_poe2.svelte');
const AppTree2 = (await import('$lib/shell/AppTree2.svelte')).default;
const AppSkills = (await import('$lib/shell/AppSkills.svelte')).default;

const fixture = (name: string) => readFile(resolve(FIXTURES, name), 'utf8');
const undo = poe2Engine.undo!;

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

// each edit lands further apart than the history joins a burst of edits into one step
const spaced = () => {
  let now = Date.UTC(2026, 0, 1);
  return vi.spyOn(Date, 'now').mockImplementation(() => (now += 5_000));
};

describe('the PoE2 whole-build undo history', () => {
  it('names an edit on each page as a step, and undoing each one gives back the build as it was before it', async () => {
    spaced();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    expect((await undo.history())?.steps).toEqual([]);

    const states = [worker2Instance.Serialize()];
    const edits: [string, () => Promise<unknown>][] = [
      ['Reset passive tree Default', () => poe2Run((w) => w.TreeReset())],
      ['Enabled Rapid Casting I in Spark', () => poe2Run((w) => w.EditSkills({ op: 'gemEnabled', group: 1, gem: 2, state: true }))],
      ['Emptied Weapon 2', () => poe2Run((w) => w.EditItems({ op: 'equip', slot: 'Weapon 2', id: 0 }))],
      ['Turned on Do you use Power Charges', () => poe2Run((w) => w.SetConfigOption('usePowerCharges', true))],
      ['Set the character level to 50', () => poe2Run((w) => w.SetLevel(50))]
    ];
    for (const [, edit] of edits) {
      await edit();
      states.push(worker2Instance.Serialize());
    }
    expect(new Set(states).size, 'every edit changed the build').toBe(states.length);
    const view = await undo.history();
    expect(view?.steps.map((s) => s.action)).toEqual(edits.map(([label]) => label).reverse());
    expect(view?.steps.map((s) => s.page)).toEqual(['Tree', 'Config', 'Items', 'Skills', 'Tree']);
    // the level step's stat changes are the two calculations' difference
    expect(view?.steps[0].delta?.length).toBeGreaterThan(0);

    for (let i = edits.length - 1; i >= 0; i--) {
      const shown = await undo.undo();
      expect(shown?.undone).toBe(edits.length - i);
      expect(worker2Instance.Serialize(), `undoing "${edits[i][0]}"`).toBe(states[i]);
    }
    expect(get(poe2Build)?.level).toBe(40);
    expect((await undo.undo())?.undone, 'nothing left to undo').toBe(edits.length);

    await undo.redo();
    expect(worker2Instance.Serialize()).toBe(states[1]);
    await undo.undoTo(0);
    expect(worker2Instance.Serialize()).toBe(states.at(-1));
    expect((get(poe2Build) as Poe2Calc | undefined)?.level).toBe(50);

    // a new edit after an undo drops the steps above it, as PoE1's history does
    await undo.undoTo(2);
    await poe2Run((w) => w.SetLevel(45));
    expect((await undo.history())?.steps.map((s) => s.action)).toEqual([
      'Set the character level to 45',
      'Emptied Weapon 2',
      'Enabled Rapid Casting I in Spark',
      'Reset passive tree Default'
    ]);
    vi.restoreAllMocks();
  }, 180_000);

  it('records a change no recalculation made (the notes) before undoing past it', async () => {
    spaced();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    const opened = worker2Instance.Serialize();
    await poe2Run((w) => w.SetLevel(30));
    const level = worker2Instance.Serialize();
    worker2Instance.SetNotes('a note', '');
    await undo.undo();
    expect(worker2Instance.Serialize()).toBe(level);
    expect((await undo.history())?.steps.map((s) => s.action)).toEqual(['Edited the notes', 'Set the character level to 30']);
    await undo.undo();
    expect(worker2Instance.Serialize()).toBe(opened);
    vi.restoreAllMocks();
  }, 120_000);

  it("opening a build starts its history over, and Ctrl+Z on the tree and Skills pages is the history's, not the tab's", async () => {
    spaced();
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_items.xml')));
    await poe2Run((w) => w.SetLevel(50));
    poe2Build.set(await worker2Instance.Open(await fixture('sorceress_spark.xml')));
    expect((await undo.history())?.steps).toEqual([]);

    const { tinyTree } = await import('$lib/components/skill-tree/tree2_fixture');
    const treeData = vi.spyOn(worker2Instance, 'TreeData').mockResolvedValue(tinyTree());
    const tree = render(AppTree2);
    await vi.waitFor(() => tree.getByRole('img', { name: 'Passive skill tree' }), { timeout: 60_000 });
    treeData.mockRestore();
    poe2Build.set((await worker2Instance.TreeClick(44871, false, 0, false, 1)).calc);
    issued.length = 0;
    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await fireEvent.keyDown(window, { key: 'y', ctrlKey: true });
    tree.unmount();

    const skills = render(AppSkills);
    await vi.waitFor(() => skills.getByRole('region', { name: 'Socket groups' }));
    await fireEvent.keyDown(window, { key: 'z', ctrlKey: true });
    await new Promise((done) => setTimeout(done, 50));
    expect(issued.filter((c) => /Undo|Redo/.test(c))).toEqual([]);
    vi.restoreAllMocks();
  }, 120_000);
});
