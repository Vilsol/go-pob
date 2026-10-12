// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

afterEach(cleanup);

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0009-trickster-cobra-lash.xml';

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
}, 120_000);

describe('the sidebar main-skill list', () => {
  it("lists the active skill set's groups when the set's id is not its position", async () => {
    const b = workerInstance.currentBuild!;
    const index = b.ActiveSkillSetIndex();
    expect(index, `${BUILD}: the active set sits at its id's position, so this test proves nothing`).not.toBe(b.Skills.ActiveSkillSet - 1);
    const want = workerInstance.GetSocketGroupSummaries(index).map((r) => r.label);
    expect(want.length).toBeGreaterThan(0);

    const Sidebar = (await import('./AppSidebar.svelte')).default;
    outputs.set({ Output: {} } as never);
    const view = render(Sidebar);
    const select = (await view.findByRole('combobox', { name: 'Main skill' })) as HTMLSelectElement;
    await vi.waitFor(() => expect([...select.options].map((o) => o.textContent)).toEqual(want));
    outputs.set(undefined);
  });
});
