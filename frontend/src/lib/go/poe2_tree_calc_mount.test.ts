// @vitest-environment jsdom
// PassiveTreeView.lua:1967-2053 and CalcsTab.lua:551-709, TreeTab.lua:932-1048 (PoB-PoE2) through the PoE2 worker

import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { cleanup, render } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { get } from 'svelte/store';

afterEach(cleanup);

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
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()) }));
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

const { poe2Build } = await import('$lib/global');
const Host = (await import('./poe2_tree_calc_host.test.svelte')).default;
const TreeHost = (await import('./poe2_tree_host.test.svelte')).default;
const CALC2_FIXTURES = resolve(process.cwd(), '../calculator2/testdata');

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

// node 44871 is +10 to maximum Energy Shield; PoB-PoE2's sidebar with it allocated (oracle/poe2_sidebar.lua) shows
// Energy Shield 10 where the build has none
describe('the PoE2 tree calculations', () => {
  it("a node's hover shows its stat differences and pathing, and leaves the build as it was", async () => {
    const opened = await worker2Instance.Open(await readFile(resolve(FIXTURES, 'sorceress_spark.xml'), 'utf8'));
    poe2Build.set(opened);
    let calc: import('./engine_poe2_tree_calc.svelte').Poe2TreeCalc | undefined;
    render(Host, { onmodel: (m: typeof calc) => (calc = m) });
    await vi.waitFor(() => expect(calc).toBeDefined());
    await calc!.readPowerStats();
    expect(calc!.powerStats[0]).toEqual({ Stat: '', Label: 'Offence/Defence' });

    calc!.hoverNode(44871);
    await vi.waitFor(() => expect(calc!.diff?.pending).toBe(false));
    expect(calc!.diff!.blocks.map((b) => b.header)).toEqual(['Allocating this node will give you:']);
    const es = calc!.diff!.blocks[0].sections.flatMap((s) => s.lines).find((l) => l.label === 'Energy Shield');
    expect(es?.value).toBe('+10');
    expect(calc!.tip?.pathLines).toEqual(['1 points to node']);

    const before = worker2Instance.Serialize();
    calc!.setPowerDepth(1);
    calc!.setPowerStat('EnergyShield');
    calc!.setPowerOn(true);
    await vi.waitFor(() => expect(calc!.powerRows).toBeDefined(), { timeout: 60_000 });
    expect(calc!.power?.nodes.get(44871)?.single).toBe(10);
    expect(calc!.powerRows!.find((r) => r.ID === 44871)).toMatchObject({ Power: 10, PathDist: 1, Allocated: false });
    expect(calc!.powerProgress?.finished).toBe(true);
    expect(worker2Instance.Serialize()).toBe(before);
    expect(get(poe2Build)).toBe(opened);
    expect(issued).not.toContain('Open');
  }, 120_000);

  // PoB-PoE2 on warrior_ascendancy (level 80): PassiveTreeView.lua:2026-2057 shows 14,520 Gold to unallocate the ascendancy
  // node 27418, and TreeTab.lua:85-112 37,752 Gold, 3 passive and 2 ascendancy nodes to switch to the "Respec" tree
  it('the node and tree list tooltips show the gold a respec costs', async () => {
    poe2Build.set(await worker2Instance.Open(await readFile(resolve(CALC2_FIXTURES, 'warrior_ascendancy.xml'), 'utf8')));
    let tree: ReturnType<typeof import('./engine_poe2_tree.svelte').poe2TreeModel> | undefined;
    render(TreeHost, { onmodel: (m: typeof tree) => (tree = m) });
    await vi.waitFor(() => expect(tree?.specs).toEqual(['Default', 'Respec']));

    tree!.calc.hoverNode(27418);
    await vi.waitFor(() => expect(tree!.calc.tip?.pathLines).toEqual(['14520 Gold required to unallocate this node']));

    tree!.hoverSpec(1);
    await vi.waitFor(() =>
      expect(tree!.specTip?.footer).toBe(
        '37,752 Gold required to switch to this tree.\n3 Passive nodes to be refunded.\n2 Ascendancy nodes to be refunded.\nGame Version: 0_5'
      )
    );
    tree!.hoverSpec(0);
    await vi.waitFor(() => expect(tree!.specTip?.footer).toBe('Game Version: 0_5'));
  }, 120_000);
});
