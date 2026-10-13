// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

afterEach(cleanup);

const pageErrors: string[] = [];
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted component threw').toEqual([]);
});

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

const LINES = ['1,000 ^8(base)', 'x 1.20 ^8(increased/reduced)', '= 1,200'];
let pass: MockInstance;
const asked: string[] = [];

beforeAll(async () => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  await bootWorker(currentBuild);
  await loadCorpusBuild('0001-necromancer-unknown.xml');
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  pass = vi.spyOn(workerInstance as never, 'runSidebarPass').mockImplementation((() => ({
    Breakdown: (_minion: boolean, stat: string) => {
      asked.push(stat);
      return stat === 'Life' ? LINES : undefined;
    },
    release: () => {}
  })) as never);
}, 120_000);

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const row = async (container: HTMLElement, label: string) =>
  vi.waitFor(() => {
    const s = [...container.querySelectorAll('span.stat-label')].find((e) => e.textContent?.trim() === label);
    if (!s) throw new Error(`no ${label} row`);
    return s.closest('button')!;
  });

describe('the sidebar tooltip breakdown', () => {
  it('asks after the pointer rests, once per recalculation', async () => {
    const Sidebar = (await import('./AppSidebar.svelte')).default;
    outputs.set({ Output: { Life: 1200, Mana: 300, ManaUnreserved: 300 } } as never);
    const view = render(Sidebar);
    const life = await row(view.container, 'Total Life');
    const mana = await row(view.container, 'Total Mana');

    await fireEvent.mouseEnter(mana);
    await sleep(40);
    await fireEvent.mouseLeave(mana);
    await fireEvent.mouseEnter(life);
    await sleep(40);
    await fireEvent.mouseLeave(life);
    await sleep(300);
    expect(pass, 'a sweep made a pass').not.toHaveBeenCalled();

    await fireEvent.mouseEnter(life);
    const steps = await view.findByRole('group', { name: 'Calculation' }, { timeout: 2000 });
    expect([...steps.querySelectorAll('[data-step]')].map((s) => s.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
      '1,000 (base)',
      '× 1.20 (increased/reduced)',
      '= 1,200'
    ]);
    expect(pass).toHaveBeenCalledTimes(1);
    await fireEvent.mouseLeave(life);

    await fireEvent.mouseEnter(mana);
    await vi.waitFor(() => expect(asked).toContain('Mana'));
    await fireEvent.mouseLeave(mana);
    expect(pass).toHaveBeenCalledTimes(1);

    (workerInstance as unknown as { ticksStarted: number }).ticksStarted++;
    outputs.set({ Output: { Life: 1300, Mana: 300, ManaUnreserved: 300 } } as never);
    await fireEvent.mouseEnter(await row(view.container, 'Total Life'));
    await vi.waitFor(() => expect(pass).toHaveBeenCalledTimes(2));
    await sleep(300);
    expect(pass).toHaveBeenCalledTimes(2);
    outputs.set(undefined);
  });
});
