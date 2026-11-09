// how long each kind of job took the last few times it finished, kept across visits so "usually ~N s" means something on
// the next start; without storage the history lives for the session only
const KEY = 'activity:durations';
const KEEP = 10;

let memory: Record<string, number[]> | undefined;

const load = (): Record<string, number[]> => {
  if (memory) return memory;
  memory = {};
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(KEY) ?? '{}');
    if (raw && typeof raw === 'object')
      for (const [kind, l] of Object.entries(raw)) {
        if (Array.isArray(l)) memory[kind] = l.filter((v): v is number => typeof v === 'number' && Number.isFinite(v) && v >= 0).slice(-KEEP);
      }
  } catch {
    // no storage, or a value that is not ours
  }
  return memory;
};

export const recordDuration = (kind: string, ms: number) => {
  if (!Number.isFinite(ms) || ms < 0) return;
  const all = load();
  all[kind] = [...(all[kind] ?? []), Math.round(ms)].slice(-KEEP);
  try {
    localStorage.setItem(KEY, JSON.stringify(all));
  } catch {
    // kept in memory
  }
};

// the median of the recent runs, so one cold start does not set the expectation
export const typicalDuration = (kind: string): number | undefined => {
  const l = load()[kind];
  if (!l?.length) return undefined;
  const s = [...l].sort((a, b) => a - b);
  const mid = s.length >> 1;
  return s.length % 2 ? s[mid] : (s[mid - 1] + s[mid]) / 2;
};

export const forgetDurations = () => {
  memory = undefined;
  try {
    localStorage.removeItem(KEY);
  } catch {
    // nothing stored
  }
};
