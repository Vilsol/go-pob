// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { SHOW_AFTER_MS, KEEP_FINISHED, jobs, overallFraction, remaining, resetJobs, runningJobs, startJob, track, usually } from './jobs';
import { forgetDurations, recordDuration, typicalDuration } from './durations';
import { applyJobMessage, endSourceJobs, forwardWorkerJobs, isJobPost } from './worker_jobs';
import { formatAbout, formatBytes, formatProgress, formatSeconds } from './format';

beforeEach(() => {
  vi.useFakeTimers();
  resetJobs();
  forgetDurations();
});
afterEach(() => {
  resetJobs();
  vi.useRealTimers();
});

describe('a job', () => {
  it('stays hidden while it is shorter than the reveal delay, and is never drawn when it ends inside it', () => {
    const j = startJob('save', 'Saving');
    vi.advanceTimersByTime(SHOW_AFTER_MS - 1);
    expect(get(jobs)).toEqual([]);
    j.end();
    vi.advanceTimersByTime(SHOW_AFTER_MS * 10);
    expect(get(jobs)).toEqual([]);
    expect(globalThis.__pobJobs?.map((x) => [x.kind, x.state])).toEqual([['save', 'ok']]);
  });

  it('is drawn once it has run the reveal delay, and stays listed as finished', () => {
    const j = startJob('import', 'Importing');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(get(runningJobs).map((x) => x.label)).toEqual(['Importing']);
    vi.advanceTimersByTime(700);
    j.end();
    expect(get(runningJobs)).toEqual([]);
    expect(get(jobs).map((x) => [x.label, x.state, x.end! - x.start])).toEqual([['Importing', 'ok', 1000]]);
  });

  it('that fails is drawn at once however fast, with its message', () => {
    startJob('open', 'Opening').end('error', new Error('no such build'));
    expect(get(jobs).map((x) => [x.state, x.error])).toEqual([['error', 'no such build']]);
  });

  it('that is cancelled is neither drawn nor learned from', () => {
    const j = startJob('page', 'Opening Items');
    vi.advanceTimersByTime(SHOW_AFTER_MS + 1);
    j.end('cancelled');
    expect(get(jobs)).toEqual([]);
    expect(typicalDuration('page')).toBeUndefined();
  });

  it('records each phase with its own duration and the progress it reached', () => {
    const j = startJob('boot', 'Starting');
    j.phase('Downloading').progress(10, 100, 'bytes');
    vi.advanceTimersByTime(400);
    j.progress(100, 100, 'bytes');
    j.phase('Booting');
    vi.advanceTimersByTime(250);
    const live = get(jobs)[0];
    expect([live.done, live.total], 'a new phase starts with no progress').toEqual([undefined, undefined]);
    j.end();
    const done = get(jobs)[0];
    expect(done.phases.map((p) => [p.name, p.end! - p.start, p.done, p.total, p.unit])).toEqual([
      ['Downloading', 400, 100, 100, 'bytes'],
      ['Booting', 250, undefined, undefined, undefined]
    ]);
  });

  it('ignores calls after it ended', () => {
    const j = startJob('x', 'X');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    j.end();
    j.phase('late').progress(1, 2);
    j.end('error', 'late');
    expect(get(jobs).map((x) => [x.state, x.phases.length])).toEqual([['ok', 0]]);
  });

  it('keeps only the newest finished jobs', () => {
    for (let i = 0; i < KEEP_FINISHED + 5; i++) {
      const j = startJob('x', `job ${i}`);
      vi.advanceTimersByTime(SHOW_AFTER_MS);
      j.end();
    }
    const l = get(jobs);
    expect(l).toHaveLength(KEEP_FINISHED);
    expect(l[0].label).toBe(`job ${KEEP_FINISHED + 4}`);
  });

  it('lists running jobs before finished ones', () => {
    const a = startJob('a', 'A');
    const b = startJob('b', 'B');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    a.end();
    expect(get(jobs).map((x) => x.label)).toEqual(['B', 'A']);
    b.end();
  });
});

describe('track', () => {
  it('ends ok with the promise, and passes a rejection on after ending in error', async () => {
    let fail!: (e: Error) => void;
    const p = track(new Promise((_, r) => (fail = r)), 'import', 'Importing');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    fail(new Error('bad code'));
    await expect(p).rejects.toThrow('bad code');
    expect(get(jobs).map((x) => [x.state, x.error])).toEqual([['error', 'bad code']]);
    await track(Promise.resolve(1), 'x', 'X');
    await Promise.resolve();
    expect(globalThis.__pobJobs?.at(-1)?.state).toBe('ok');
  });
});

describe('typical durations', () => {
  it('are the median of the recent successful runs of a kind, kept in localStorage', () => {
    for (const [ms, next] of [
      [1000, 1000],
      [3000, 2000],
      [2000, 2000]
    ] as const) {
      const j = startJob('boot', 'Boot');
      vi.advanceTimersByTime(ms);
      j.end();
      expect(usually(get(jobs)[0])).toBe(next);
    }
    expect(JSON.parse(localStorage.getItem('activity:durations')!)).toEqual({ boot: [1000, 3000, 2000] });
  });

  it('keep the last ten', () => {
    for (let i = 1; i <= 12; i++) recordDuration('x', i * 1000);
    expect((JSON.parse(localStorage.getItem('activity:durations')!) as Record<string, number[]>).x).toEqual(
      [3, 4, 5, 6, 7, 8, 9, 10, 11, 12].map((n) => n * 1000)
    );
  });

  it('work without storage, and ignore a stored value that is not theirs', () => {
    localStorage.setItem('activity:durations', '{"x":["a",-1,500],"y":3}');
    expect(typicalDuration('x')).toBe(500);
    expect(typicalDuration('y')).toBeUndefined();
    forgetDurations();
    const spy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('denied');
    });
    const set = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('denied');
    });
    recordDuration('z', 700);
    expect(typicalDuration('z')).toBe(700);
    spy.mockRestore();
    set.mockRestore();
  });
});

describe('estimates', () => {
  it('give the time left at the current phase rate only with determinate progress', () => {
    const j = startJob('data', 'Data');
    j.phase('Downloading');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(remaining(get(jobs)[0], Date.now())).toBeUndefined();
    vi.advanceTimersByTime(700);
    j.progress(25, 100, 'bytes');
    expect(remaining(get(jobs)[0], Date.now())).toBe(3000);
    j.end();
  });

  it('average the fraction over the jobs that know a total', () => {
    const a = startJob('a', 'A').progress(1, 4);
    const b = startJob('b', 'B').progress(3, 4);
    const c = startJob('c', 'C');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(overallFraction(get(jobs))).toBe(0.5);
    a.end();
    b.end();
    expect(overallFraction(get(jobs)), 'indeterminate when no running job has a total').toBeUndefined();
    c.end();
  });
});

describe('job messages', () => {
  it('start, move and end a job from a source, keyed apart from another source', () => {
    applyJobMessage({ id: '1', kind: 'data', label: 'Loading game data', long: true, phase: 'Downloading', done: 5, total: 10, unit: 'bytes' }, 'calc');
    applyJobMessage({ id: '1', kind: 'calc2', label: 'Other worker' }, 'calc2');
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(get(runningJobs).map((j) => [j.label, j.long, j.phases.at(-1)?.name, j.done, j.total, j.unit])).toEqual([
      ['Loading game data', true, 'Downloading', 5, 10, 'bytes'],
      ['Other worker', false, undefined, undefined, undefined, undefined]
    ]);
    applyJobMessage({ id: '1', end: 'error', error: 'gone' }, 'calc');
    expect(get(jobs).map((j) => [j.label, j.state, j.error])).toEqual([
      ['Other worker', 'running', undefined],
      ['Loading game data', 'error', 'gone']
    ]);
    endSourceJobs('calc2', 'the worker stopped');
    expect(get(runningJobs)).toEqual([]);
  });

  it('without a kind for an unknown id are ignored', () => {
    applyJobMessage({ id: 'x', phase: 'p' });
    applyJobMessage({ id: 'x', end: 'ok' });
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(get(jobs)).toEqual([]);
  });

  it('arrive from a worker as { pobJob } posts', () => {
    const target = new EventTarget();
    forwardWorkerJobs(target, 'w');
    target.dispatchEvent(new MessageEvent('message', { data: { pobJob: { id: '1', kind: 'k', label: 'From the worker' } } }));
    target.dispatchEvent(new MessageEvent('message', { data: { id: 'comlink', value: 1 } }));
    vi.advanceTimersByTime(SHOW_AFTER_MS);
    expect(get(runningJobs).map((j) => j.label)).toEqual(['From the worker']);
    expect(isJobPost({ pobJob: { id: 3 } })).toBe(false);
    endSourceJobs('w', 'done');
  });
});

describe('format', () => {
  it('writes durations, sizes and counts', () => {
    expect([formatSeconds(1234), formatSeconds(42_400), formatSeconds(125_000)]).toEqual(['1.2 s', '42 s', '2 min 5 s']);
    expect([formatAbout(400), formatAbout(2600), formatAbout(180_000)]).toEqual(['<1 s', '~3 s', '~3 min']);
    expect([formatBytes(512), formatBytes(2048), formatBytes(5.5 * 1024 * 1024), formatBytes(40 * 1024 * 1024)]).toEqual(['512 B', '2 KB', '5.5 MB', '40 MB']);
    expect([formatProgress(1200, 3000), formatProgress(5, undefined), formatProgress(1024, 2048, 'bytes')]).toEqual(['1,200 / 3,000', '5', '1 KB / 2 KB']);
  });
});
