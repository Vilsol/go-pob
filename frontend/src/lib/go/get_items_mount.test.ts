// @vitest-environment jsdom

import { describe, expect, it, beforeAll, vi } from 'vitest';

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
}, 120_000);

const activeSlots = () => {
  const items = workerInstance.GetItems()!;
  const set = items.ItemSets.find((s) => Number(s.ID) === items.ActiveItemSet) ?? items.ItemSets[0];
  return set.Slots ?? [];
};

describe('GetItems', () => {
  it('reads an unequip made after its first read', () => {
    const equipped = activeSlots().find((s) => s.ItemID > 0);
    expect(equipped, `${BUILD} has no equipped slot`).toBeDefined();
    workerInstance.UnequipSlot(equipped!.Name);
    expect(activeSlots().find((s) => s.Name === equipped!.Name)?.ItemID ?? 0).toBe(0);
  });
});
