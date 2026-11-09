import { tick } from 'svelte';
import type { Outputs, TickMeta } from './custom_types';
import { buildOpens, outputs } from './global';
import { OPEN_SOURCES } from './go/undo';
import { epochNow } from './utils';
import { trackEditDelta } from './display/stat_delta';
import { startJob, type JobHandle } from './activity/jobs';

export type CalcTiming = TickMeta & {
  receivedAt: number;
  renderedAt: number;
};

declare global {
  // eslint-disable-next-line no-var
  var __pobCalcTimings: CalcTiming[] | undefined;
}

const MAX_TIMINGS = 5000;

// edits that queue up behind a calculation: a job from the first tick that leaves others in flight to the tick that
// leaves none, counting the ticks it has seen against those plus the ones still in flight
let queued: { job: JobHandle; seen: number } | undefined;
const followQueue = (meta: TickMeta) => {
  if (meta.pending > 0) {
    queued ??= { job: startJob('calc', 'Recalculating', { startedAt: meta.startedAt }).phase('Working through queued edits'), seen: 0 };
    queued.seen++;
    queued.job.progress(queued.seen, queued.seen + meta.pending);
  } else if (queued) {
    queued.job.end();
    queued = undefined;
  }
};

export const onTick = (out: Outputs | undefined, meta: TickMeta) => {
  const receivedAt = epochNow();
  if (OPEN_SOURCES.has(meta.source)) {
    buildOpens.update((n) => n + 1);
  }
  if (out) {
    outputs.set(out);
  }
  trackEditDelta(out, meta);
  followQueue(meta);

  void tick().then(() =>
    requestAnimationFrame(() => {
      const timings = (globalThis.__pobCalcTimings ??= []);
      if (timings.length >= MAX_TIMINGS) {
        timings.shift();
      }
      timings.push({ ...meta, receivedAt, renderedAt: epochNow() });

      document.documentElement.dataset.calcPending = String(meta.pending);
      document.documentElement.dataset.calcSeq = String(meta.seq);
    })
  );
};
