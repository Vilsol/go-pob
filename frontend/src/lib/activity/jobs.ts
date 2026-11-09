import { derived, get, writable } from 'svelte/store';
import { logText } from '$lib/app_log';
import { recordDuration, typicalDuration } from './durations';

export type JobUnit = 'bytes' | 'count';
export type JobResult = 'ok' | 'error' | 'cancelled';
export type JobPhase = { name: string; start: number; end?: number; done?: number; total?: number; unit?: JobUnit };
export type Job = {
  id: string;
  kind: string;
  label: string;
  start: number;
  end?: number;
  state: 'running' | JobResult;
  error?: string;
  phases: JobPhase[];
  done?: number;
  total?: number;
  unit?: JobUnit;
  // boot, import and data downloads also get a progress toast
  long: boolean;
  // a job is drawn once it has run SHOW_AFTER_MS, or once it failed; a fast one never flickers into view
  shown: boolean;
};
export type JobOptions = { long?: boolean; startedAt?: number };
export type JobHandle = {
  readonly id: string;
  phase: (name: string) => JobHandle;
  progress: (done: number, total?: number, unit?: JobUnit) => JobHandle;
  end: (result?: JobResult, error?: unknown) => void;
};

export const SHOW_AFTER_MS = 300;
export const KEEP_FINISHED = 20;
// every finished job, the fast ones included, for tests and the logs
const LOG_CAP = 200;

declare global {
  // eslint-disable-next-line no-var
  var __pobJobs: Job[] | undefined;
}

const store = writable<Job[]>([]);
let nextId = 1;
const reveal = new Map<string, ReturnType<typeof setTimeout>>();

// every job that is drawn: running ones first in start order, then the finished ones, newest first
export const jobs = derived(store, (l) => l.filter((j) => j.shown));
export const runningJobs = derived(jobs, (l) => l.filter((j) => j.state === 'running'));
export const activityPanelOpen = writable(false);

const change = (id: string, f: (j: Job) => Job | undefined) =>
  store.update((l) => {
    const i = l.findIndex((j) => j.id === id);
    if (i < 0) return l;
    const next = f(l[i]);
    if (next === l[i]) return l;
    const out = l.slice();
    if (next) out[i] = next;
    else out.splice(i, 1);
    return out;
  });

const order = (l: Job[]) => {
  const running = l.filter((j) => j.state === 'running');
  const done = l
    .filter((j) => j.state !== 'running')
    .sort((a, b) => (b.end ?? 0) - (a.end ?? 0))
    .slice(0, KEEP_FINISHED);
  return [...running, ...done];
};

const errorText = (e: unknown) => (e instanceof Error ? e.message : e === undefined ? undefined : logText(e));

const closePhase = (j: Job, at: number): JobPhase[] => {
  const last = j.phases.at(-1);
  if (!last || last.end !== undefined) return j.phases;
  return [...j.phases.slice(0, -1), { ...last, end: at, done: j.done, total: j.total, unit: j.unit }];
};

export const startJob = (kind: string, label: string, opts: JobOptions = {}): JobHandle => {
  const id = `job_${nextId++}`;
  const start = opts.startedAt ?? Date.now();
  store.update((l) => [...l, { id, kind, label, start, state: 'running', phases: [], long: !!opts.long, shown: false }]);
  reveal.set(
    id,
    setTimeout(
      () => {
        reveal.delete(id);
        change(id, (j) => (j.state === 'running' ? { ...j, shown: true } : j));
      },
      Math.max(0, start + SHOW_AFTER_MS - Date.now())
    )
  );
  const handle: JobHandle = {
    id,
    phase: (name) => {
      const at = Date.now();
      change(id, (j) =>
        j.state === 'running' ? { ...j, phases: [...closePhase(j, at), { name, start: at }], done: undefined, total: undefined, unit: undefined } : j
      );
      return handle;
    },
    progress: (done, total, unit = 'count') => {
      change(id, (j) => (j.state === 'running' ? { ...j, done, total, unit } : j));
      return handle;
    },
    end: (result = 'ok', error) => {
      const t = reveal.get(id);
      if (t !== undefined) clearTimeout(t);
      reveal.delete(id);
      const at = Date.now();
      let finished: Job | undefined;
      store.update((l) => {
        const j = l.find((x) => x.id === id);
        if (!j || j.state !== 'running') return l;
        finished = {
          ...j,
          end: at,
          state: result,
          error: result === 'error' ? (errorText(error) ?? 'failed') : undefined,
          phases: closePhase(j, at),
          shown: j.shown || result === 'error'
        };
        const rest = l.filter((x) => x !== j);
        return order(finished.shown && result !== 'cancelled' ? [...rest, finished] : rest);
      });
      if (!finished) return;
      if (result === 'ok') recordDuration(finished.kind, at - finished.start);
      const log = (globalThis.__pobJobs ??= []);
      if (log.length >= LOG_CAP) log.shift();
      log.push(finished);
    }
  };
  return handle;
};

// the job runs as long as the promise; a rejection ends it in error and is passed on
export const track = <T>(promise: Promise<T>, kind: string, label: string, opts?: JobOptions): Promise<T> => {
  const job = startJob(kind, label, opts);
  promise.then(
    () => job.end('ok'),
    (e: unknown) => job.end('error', e)
  );
  return promise;
};

export const clearFinished = () => store.update((l) => l.filter((j) => j.state === 'running'));

// drops every job and timer, for tests
export const resetJobs = () => {
  for (const t of reveal.values()) clearTimeout(t);
  reveal.clear();
  store.set([]);
  activityPanelOpen.set(false);
  globalThis.__pobJobs = [];
};

export const jobById = (id: string) => get(store).find((j) => j.id === id);

export const elapsed = (j: Job, now: number) => (j.end ?? now) - j.start;

// with determinate progress, the time left at the rate the current phase has kept so far
export const remaining = (j: Job, now: number): number | undefined => {
  if (j.state !== 'running' || !j.total || !j.done || j.done >= j.total) return undefined;
  const since = j.phases.at(-1)?.start ?? j.start;
  const spent = now - since;
  return spent > 0 ? (spent * (j.total - j.done)) / j.done : undefined;
};

export const usually = (j: Job) => typicalDuration(j.kind);

// the fraction done over every running job that knows its total, or undefined when none does
export const overallFraction = (l: Job[]): number | undefined => {
  const known = l.filter((j) => j.state === 'running' && j.total && j.total > 0);
  if (!known.length) return undefined;
  return known.reduce((s, j) => s + Math.min(1, (j.done ?? 0) / j.total!), 0) / known.length;
};
