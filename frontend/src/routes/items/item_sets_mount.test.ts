// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, screen } from '@testing-library/svelte';

afterEach(cleanup);
const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2.xml';

const goActive = () => {
  const calc = types.calculator.NewCalculator(workerInstance.currentBuild!)!;
  try {
    return calc.PoB!.Items.ActiveItemSet;
  } finally {
    calc.release();
  }
};

let tick: MockInstance<typeof workerInstance.Tick>;
let getItems: MockInstance<typeof workerInstance.GetItems>;
let setActive: MockInstance<typeof workerInstance.SetActiveItemSet>;

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'GetEquippedItems').mockResolvedValue([]);
  vi.spyOn(workerInstance, 'GetSlotItemLists').mockResolvedValue({});
  vi.spyOn(workerInstance, 'GetItemsTabSlots').mockResolvedValue({ abyssal: [], sockets: [] });
  vi.spyOn(workerInstance, 'GetItemSetTooltips').mockResolvedValue({});
  vi.spyOn(workerInstance, 'GetConditionalSlots').mockResolvedValue([]);
  vi.spyOn(workerInstance, 'GetItemListFilter').mockResolvedValue({ filters: [], items: [] });
  vi.spyOn(workerInstance, 'GetTreeSpecs').mockResolvedValue(undefined);
  vi.spyOn(workerInstance, 'GetSharedItemSets').mockResolvedValue([]);
  vi.spyOn(workerInstance, 'ParseItem').mockResolvedValue({ error: 'no game data in the component harness' });
  vi.spyOn(workerInstance, 'PrepareStatCompare').mockResolvedValue(undefined);
  setActive = vi.spyOn(workerInstance, 'SetActiveItemSet').mockImplementation((id: number) => {
    workerInstance.currentBuild!.SetActiveItemSet(id);
    return Promise.resolve();
  });
  getItems = vi.spyOn(workerInstance, 'GetItems');
}, 120_000);

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  render(Page);
  return screen.findByRole<HTMLSelectElement>('combobox', { name: 'Item set' });
};

const lastRead = () => getItems.mock.results.at(-1)?.value as { ActiveItemSet?: number } | undefined;

describe('the item-set select', () => {
  it('opens on the active set by id, is live, and writes nothing on mount', async () => {
    expect(goActive()).toBe(2);
    tick.mockClear();
    const select = await mount();
    expect(select.value).toBe('2');
    expect(select.disabled).toBe(false);
    expect([...select.options].map((o) => `${o.value}:${o.textContent}`)).toEqual(['1:Default', '2:enmity']);
    expect(setActive).not.toHaveBeenCalled();
    expect(tick).not.toHaveBeenCalled();
  }, 20_000);

  it('switching issues the chosen set’s id once and then reads Go’s sets again; switching back restores it', async () => {
    const select = await mount();
    await vi.waitFor(() => expect(lastRead()?.ActiveItemSet).toBe(2));
    setActive.mockClear();

    await fireEvent.change(select, { target: { value: '1' } });
    await vi.waitFor(() => expect(goActive()).toBe(1));
    expect(setActive.mock.calls).toEqual([[1]]);
    await vi.waitFor(() => expect(lastRead()?.ActiveItemSet).toBe(1));
    expect(select.value).toBe('1');

    await fireEvent.change(select, { target: { value: '2' } });
    await vi.waitFor(() => expect(goActive()).toBe(2));
    expect(setActive.mock.calls).toEqual([[1], [2]]);
    await vi.waitFor(() => expect(lastRead()?.ActiveItemSet).toBe(2));
  }, 20_000);
});
