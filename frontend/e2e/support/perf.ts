import { appendFileSync, mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

export const RESULTS_DIR = resolve(import.meta.dirname, '../.results');
export const SAMPLES = resolve(RESULTS_DIR, 'samples.jsonl');

export type Pctl = { p50: number; p95: number; p99: number };

export const pctl = (xs: number[]): Pctl => {
  const s = [...xs].sort((a, b) => a - b);
  const at = (p: number) => (s.length ? +s[Math.min(s.length - 1, Math.ceil((p / 100) * s.length) - 1)].toFixed(2) : NaN);
  return { p50: at(50), p95: at(95), p99: at(99) };
};

export type Sample =
  | { kind: 'cold-start'; build: string; navToBootMs: number; navToDataMs: number; newBuildToStatsMs: number }
  | {
      kind: 'action';
      action: string;
      build: string;
      settleMs: number;
      ticks: number;
      calcMs: number;
      dumpMs: number;
      messageMs: number;
      renderMs: number;
      dispatchMs: number;
    }
  | { kind: 'longtask'; build: string; phase: string; count: number; totalMs: number; maxMs: number }
  | { kind: 'interaction'; build: string; phase: string; name: string; durationMs: number }
  | {
      kind: 'frames';
      build: string;
      gesture: string;
      frames: number;
      dropped: number;
      maxGapMs: number;
      durationMs: number;
      gapMs?: Pctl;
      draws?: number;
      renderMs?: Pctl;
      longTasks?: number;
      longTaskMs?: number;
      gpu?: string;
    }
  | { kind: 'tree-latency'; build: string; what: string; ms: number | null; renderMs?: number; gpu?: string }
  | { kind: 'memory'; build: string; when: string; mainHeapMB: number; workerHeapMB: number | null; wasmMemMB: number | null; goLiveHeapMB: number | null };

export const PERF = process.env.E2E_PERF === '1' || process.env.E2E_BASELINE === '1';

export const record = (s: Sample) => {
  if (!PERF) return;
  mkdirSync(RESULTS_DIR, { recursive: true });
  appendFileSync(SAMPLES, JSON.stringify(s) + '\n');
};
