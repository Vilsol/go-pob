import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { SHOW_AFTER_MS, jobs, resetJobs, startJob, type JobHandle } from './jobs';
import { stageReporter } from './worker_jobs';

beforeEach(() => {
  vi.useFakeTimers();
  resetJobs();
});
afterEach(() => {
  resetJobs();
  vi.useRealTimers();
});

describe('stageReporter', () => {
  it('enters a phase once per stage, reports counts within it, and starts over on a new job', () => {
    let job: JobHandle | undefined;
    const report = stageReporter(() => job);
    report('Downloading', 0, 3);
    job = startJob('data', 'Loading game data');
    report('Downloading', 0, 3);
    report('Downloading', 2, 3);
    report('Mods', 0, 2);
    report('Indexing', 0, 0);
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    const [first] = get(jobs);
    expect(first.phases.map((p) => [p.name, p.done, p.total])).toEqual([
      ['Downloading', 2, 3],
      ['Mods', 0, 2],
      ['Indexing', undefined, undefined]
    ]);
    expect(first.unit).toBeUndefined();
    job.end();

    job = startJob('data', 'Loading game data');
    report('Indexing', 0, 0);
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(
      get(jobs)
        .find((j) => j.state === 'running')
        ?.phases.map((p) => p.name)
    ).toEqual(['Indexing']);
  });
});
