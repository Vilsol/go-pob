import type { JobUnit } from './jobs';

export const formatSeconds = (ms: number) => {
  const s = ms / 1000;
  if (s < 10) return `${s.toFixed(1)} s`;
  if (s < 90) return `${Math.round(s)} s`;
  return `${Math.floor(s / 60)} min ${Math.round(s % 60)} s`;
};

// whole seconds, rounded up, for an expectation ("usually ~3 s")
export const formatAbout = (ms: number) => (ms < 1000 ? '<1 s' : ms < 90_000 ? `~${Math.round(ms / 1000)} s` : `~${Math.round(ms / 60_000)} min`);

export const formatBytes = (n: number) =>
  n < 1024 ? `${n} B` : n < 1024 * 1024 ? `${(n / 1024).toFixed(0)} KB` : `${(n / (1024 * 1024)).toFixed(n < 10 * 1024 * 1024 ? 1 : 0)} MB`;

export const formatProgress = (done: number, total: number | undefined, unit: JobUnit = 'count') => {
  const f = unit === 'bytes' ? formatBytes : (n: number) => n.toLocaleString('en-US');
  return total ? `${f(done)} / ${f(total)}` : f(done);
};
