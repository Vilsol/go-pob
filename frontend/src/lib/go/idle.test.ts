import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => vi.useRealTimers());

describe('whenIdle', () => {
  it('runs only once no call has arrived for idleMs, each call postponing it', async () => {
    vi.useFakeTimers();
    const { noteCall, whenIdle } = await import('./idle');
    const run = vi.fn();
    noteCall();
    whenIdle(10_000, run);
    await vi.advanceTimersByTimeAsync(6_000);
    expect(run, 'ran while the worker had a call 6 s ago').not.toHaveBeenCalled();
    noteCall();
    await vi.advanceTimersByTimeAsync(6_000);
    expect(run, 'ran 6 s after a later call').not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(4_100);
    expect(run).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(run).toHaveBeenCalledTimes(1);
  });

  it('marks the start and the end of the job it ran', async () => {
    vi.useFakeTimers();
    const { noteCall, whenIdle } = await import('./idle');
    noteCall();
    const marked = vi.spyOn(performance, 'mark');
    const names = () => marked.mock.calls.map((c) => c[0]);
    let finish!: () => void;
    whenIdle(1_000, () => new Promise<void>((r) => (finish = r)));
    await vi.advanceTimersByTimeAsync(1_100);
    expect(names()).toEqual(['pob:idle-job-start']);
    finish();
    await vi.advanceTimersByTimeAsync(0);
    expect(names()).toEqual(['pob:idle-job-start', 'pob:idle-job-end']);
  });
});
