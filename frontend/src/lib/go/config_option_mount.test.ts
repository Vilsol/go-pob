// @vitest-environment jsdom

import { describe, expect, it, beforeAll, vi } from 'vitest';

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild('0008-necromancer-vaal-summon-skeletons.xml');
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
}, 120_000);

describe('GetConfigOption', () => {
  it('reads a number, a string and a boolean option', () => {
    workerInstance.SetConfigOption('enemyLevel', 70);
    workerInstance.SetConfigOption('usePowerCharges', true);
    workerInstance.SetConfigOption('bandit', 'Alira');
    expect(workerInstance.GetConfigOption('enemyLevel')).toBe(70);
    expect(workerInstance.GetConfigOption('usePowerCharges')).toBe(true);
    expect(workerInstance.GetConfigOption('bandit')).toBe('Alira');
  });
});
