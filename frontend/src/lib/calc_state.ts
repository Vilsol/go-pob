import { tick } from 'svelte';
import type { Outputs, TickMeta } from './custom_types';
import { buildOpens, outputs } from './global';
import { OPEN_SOURCES } from './go/undo';
import { epochNow } from './utils';
import { trackEditDelta } from './display/stat_delta';

export type CalcTiming = TickMeta & {
  receivedAt: number;
  renderedAt: number;
};

declare global {
  // eslint-disable-next-line no-var
  var __pobCalcTimings: CalcTiming[] | undefined;
}

const MAX_TIMINGS = 5000;

export const onTick = (out: Outputs | undefined, meta: TickMeta) => {
  const receivedAt = epochNow();
  if (OPEN_SOURCES.has(meta.source)) {
    buildOpens.update((n) => n + 1);
  }
  if (out) {
    outputs.set(out);
  }
  trackEditDelta(out, meta);

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
