// @vitest-environment jsdom
// CompareTab.lua:3836-3837, 3979-3980 and CompareBuySimilar.lua:47-206, 211-283, 458-511 over the shared Buy Similar
// dialog and the real engine

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { cachedAsset } from '../../../e2e/support/data_cache';

afterEach(cleanup);
const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const TESTDATA = resolve(process.cwd(), '../testdata');

beforeAll(async () => {
  // go-pob-data's assets come from the Go bundle cache when a Go test cached them, else from the network
  const nodeFetch = globalThis.fetch;
  vi.stubGlobal('fetch', async (input: RequestInfo | URL, init?: RequestInit) => {
    const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
    const cached = (() => {
      try {
        return cachedAsset(href);
      } catch {
        return undefined;
      }
    })();
    if (cached?.status === 200) {
      const body = new Uint8Array(cached.body).buffer;
      return {
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Headers(),
        url: href,
        redirected: false,
        body: undefined,
        arrayBuffer: () => Promise.resolve(body)
      };
    }
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
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
  // the item tables, the trade site's stats list and the stat descriptions load through an in-memory cache
  const cache = new Map<string, Uint8Array | undefined>();
  await types.storage.InitializeStorage(
    () => [],
    (bucket, key) => cache.get(`${bucket}/${key}`),
    (bucket, key, value) => void cache.set(`${bucket}/${key}`, value),
    (bucket, key) => cache.has(`${bucket}/${key}`),
    () => {},
    () => {}
  );
  const data = await types.raw.InitializeAll(types.raw.GameDataVersion(), () => {});
  if (!data.ok) throw data.error;
  // the page's stats are not under test: the calculation is stubbed, and the Items view needs only some outputs
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  outputs.set({ Output: {} } as never);
  // the compared build is a fixture oracle/buy_similar.lua records (`mise run buy-similar`)
  await workerInstance.LoadCompareCode(types.pob.CompressEncode(await readFile(resolve(TESTDATA, 'many-builds/7.xml'), 'utf8')).unwrap(), 'B');
}, 180_000);

type Search = { base: string; query: unknown };

describe('Buy Similar on the Compare page', () => {
  it("lists B's item's mods and searches the checked ones as PoB does", async () => {
    const pob = JSON.parse(await readFile(resolve(TESTDATA, 'oracle/buy_similar.json'), 'utf8')) as Record<
      string,
      Record<string, { mods: { label: string; min?: string; canSearch: boolean }[]; modSearch: Search }>
    >;
    const opened: string[] = [];
    const spy = vi.spyOn(window, 'open').mockImplementation((url) => {
      opened.push(String(url));
      return null;
    });
    const Compare = (await import('./AppCompare.svelte')).default;
    const page = render(Compare);
    await fireEvent.click(await page.findByRole('tab', { name: 'Items' }, { timeout: 60_000 }));
    const region = await page.findByRole('region', { name: 'Item differences' }, { timeout: 30_000 });
    const rowOf = (label: string) =>
      [...region.querySelectorAll<HTMLElement>('[role="row"]')].find((r) => r.querySelector('[role="cell"]')?.textContent?.trim() === label);
    await waitFor(() => expect(rowOf('Amulet')).toBeTruthy(), { timeout: 30_000 });
    await fireEvent.click(within(rowOf('Amulet')!).getByRole('button', { name: 'Buy' }));
    const dialog = within(await page.findByRole('dialog', { name: 'Buy Similar' }));
    const group = await dialog.findByRole('group', { name: 'Mods' }, { timeout: 60_000 });
    const want = pob['7.xml'].Amulet;
    expect(
      want.mods.some((m) => !m.canSearch),
      'the fixture item has no row the trade site cannot search'
    ).toBe(true);
    const boxes = within(group).getAllByRole<HTMLInputElement>('checkbox');
    const plain = (label: string) => label.replace(/\^x[0-9A-Fa-f]{6}|\^[0-9A-Fa-f]/g, '');
    expect(boxes.map((b) => b.parentElement?.textContent?.trim())).toEqual(want.mods.map((m) => plain(m.label).replaceAll('\n', '')));
    for (const [i, m] of want.mods.entries()) {
      const name = plain(m.label).replaceAll('\n', ' ');
      expect(boxes[i].disabled).toBe(!m.canSearch);
      if (m.min !== undefined) expect(within(group).getByRole<HTMLInputElement>('textbox', { name: `${name} min` }).value).toBe(m.min);
      if (m.canSearch) await fireEvent.click(boxes[i]);
      if (m.canSearch && m.min !== undefined && i % 2 === 0)
        await fireEvent.input(within(group).getByRole('textbox', { name: `${name} max` }), { target: { value: '999' } });
    }
    await fireEvent.click(dialog.getByRole('button', { name: 'Search' }));
    await waitFor(() => expect(opened).toHaveLength(1));
    const [base, q] = opened[0].split('?q=');
    expect({ base, query: JSON.parse(decodeURIComponent(q)) as unknown }).toEqual(want.modSearch);
    spy.mockRestore();
  }, 180_000);
});
