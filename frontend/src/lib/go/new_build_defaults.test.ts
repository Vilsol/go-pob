// @vitest-environment jsdom
// Build.lua:79-84, SkillsTab.lua:92

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
  vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
}, 120_000);

describe('new build defaults', () => {
  it('a new build has the default level and gem quality', async () => {
    await workerInstance.NewBuild({ level: 90, gemQuality: 20 });
    const b = workerInstance.currentBuild!;
    expect([b.Build.Level, b.Build.CharacterLevelAutoMode, b.Skills.DefaultGemQuality]).toEqual([90, false, 20]);
    await workerInstance.NewBuild({ level: 1, gemQuality: 0 });
    const c = workerInstance.currentBuild!;
    expect([c.Build.Level, c.Build.CharacterLevelAutoMode, c.Skills.DefaultGemQuality]).toEqual([1, true, 0]);
  });
});
