// @vitest-environment jsdom

import { describe, expect, it, beforeAll, vi } from 'vitest';

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

beforeAll(async () => {
  await bootWorker(currentBuild);
}, 120_000);

// TreeTab.lua:513-516
describe('a tree version the program has no data for', () => {
  const unknown = '3_25';
  const reason = /unknown passive tree version "3_25"/;

  it('is refused by name by every tree call, and the worker keeps answering', async () => {
    await expect(Promise.resolve().then(() => workerInstance.CalculateAllocationPaths(unknown, [], [1]))).rejects.toThrow(reason);
    await expect(Promise.resolve().then(() => workerInstance.CalculatePrunableNodes(unknown, [], [1]))).rejects.toThrow(reason);
    await expect(Promise.resolve().then(() => workerInstance.CalculateDependentNodes(unknown, [], 1, [1]))).rejects.toThrow(reason);
    await expect(Promise.resolve().then(() => workerInstance.CalculateDependentNodesWith(unknown, [], 1, [1], []))).rejects.toThrow(reason);
    await expect(workerInstance.GetTree(unknown)).rejects.toThrow(reason);
    expect(workerInstance.GetTreeVersion()).toBeTruthy();
  });
});
