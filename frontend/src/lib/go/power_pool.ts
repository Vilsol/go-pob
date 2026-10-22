import type { calculator } from '../types';
import PowerWorker from './power_worker?worker';
import { emitLog, isLogMessage, log } from '$lib/app_log';

export type PowerPoolRequest =
  | { type: 'boot'; module: WebAssembly.Module }
  | { type: 'run'; seq: number; xml: string; stat: string; maxDepth: number; index: number; count: number }
  | { type: 'cancel' };

export type PowerPoolReply =
  | { type: 'ready'; ms: number }
  | { type: 'progress'; seq: number; done: number; tasks: number; current: number; shard?: calculator.PowerShardRows }
  | { type: 'done'; seq: number; tasks: number; shard: calculator.PowerShardRows; memory: number }
  | { type: 'error'; seq: number; message: string };

export const POWER_POOL_IDLE_MS = 5 * 60_000;

type Shard = {
  seq: number;
  done: number;
  tasks: number;
  current: number;
  partial?: calculator.PowerShardRows;
  settle: (rows: calculator.PowerShardRows | undefined) => void;
  fail: (err: Error) => void;
};

type Slot = { worker: Worker; spawned: number; bootMs?: number; memoryMB?: number; ready: Promise<void>; shard?: Shard; warmed?: boolean };

const WARM_SEQ = -1;

export type PowerPoolRun = {
  shards: Promise<calculator.PowerShardRows | undefined>[];
  progress(): { done: number; tasks: number }[];
  partial(): calculator.PowerShardRows[];
  current(): number[];
};

export class PowerPool {
  private slots: Slot[] = [];
  private idle?: ReturnType<typeof setTimeout>;

  constructor(private readonly module: () => WebAssembly.Module | undefined) {}

  private spawn(): Slot {
    const module = this.module();
    if (!module) throw new Error('the wasm module is not compiled yet');
    const worker = new PowerWorker({ name: 'power' });
    let ready!: () => void;
    let failed!: (err: Error) => void;
    const slot: Slot = {
      worker,
      spawned: performance.now(),
      ready: new Promise<void>((res, rej) => {
        ready = res;
        failed = rej;
      })
    };
    slot.ready.catch(() => {});
    const fail = (err: Error) => {
      failed(err);
      slot.shard?.fail(err);
      slot.shard = undefined;
      this.drop(slot);
    };
    worker.onmessage = (e: MessageEvent<PowerPoolReply>) => {
      if (isLogMessage(e.data)) {
        emitLog(e.data.pobLog);
        return;
      }
      const m = e.data;
      if (m.type === 'ready') {
        slot.bootMs = performance.now() - slot.spawned;
        log('info', 'power', `power worker booted in ${Math.round(slot.bootMs)} ms (wasm + data ${Math.round(m.ms)} ms)`);
        ready();
        return;
      }
      if (m.type === 'error' && m.seq === 0) {
        fail(new Error(m.message));
        return;
      }
      if (m.seq === WARM_SEQ) {
        if (m.type !== 'progress') slot.warmed = true;
        return;
      }
      const s = slot.shard;
      if (!s || s.seq !== m.seq) return;
      if (m.type === 'progress') {
        s.done = m.done;
        s.tasks = m.tasks;
        s.current = m.current;
        if (m.shard) s.partial = m.shard;
      } else if (m.type === 'done') {
        s.done = 1;
        s.tasks = m.tasks;
        s.current = 0;
        slot.memoryMB = m.memory / 2 ** 20;
        slot.shard = undefined;
        slot.warmed = true;
        s.settle(m.shard);
      } else {
        fail(new Error(m.message));
      }
    };
    worker.onerror = (e) => {
      e.preventDefault();
      fail(new Error(`power worker failed: ${e.message}`));
    };
    worker.postMessage({ type: 'boot', module } satisfies PowerPoolRequest);
    return slot;
  }

  private drop(slot: Slot) {
    slot.worker.terminate();
    this.slots = this.slots.filter((s) => s !== slot);
  }

  run(seq: number, xml: string, stat: string, maxDepth: number, count: number): PowerPoolRun {
    clearTimeout(this.idle);
    this.idle = undefined;
    this.cancel(false);
    for (const slot of this.slots.slice(count - 1)) this.drop(slot);
    while (this.slots.length < count - 1) this.slots.push(this.spawn());
    const states: Shard[] = [];
    const shards = this.slots.map((slot, i) => {
      const index = i + 1;
      return new Promise<calculator.PowerShardRows | undefined>((settle, fail) => {
        const state: Shard = { seq, done: 0, tasks: 0, current: 0, settle, fail };
        states.push(state);
        slot.shard = state;
        slot.ready
          .then(() => {
            if (slot.shard === state) slot.worker.postMessage({ type: 'run', seq, xml, stat, maxDepth, index, count } satisfies PowerPoolRequest);
          })
          .catch(fail);
      });
    });
    void Promise.allSettled(shards).then(() => this.release());
    return {
      shards,
      progress: () => states.map((s) => ({ done: s.done, tasks: s.tasks })),
      partial: () => states.flatMap((s) => (s.partial ? [s.partial] : [])),
      current: () => states.flatMap((s) => (s.current ? [s.current] : []))
    };
  }

  warm(count: number, xml: string): boolean {
    while (this.slots.length < count - 1) {
      const slot = this.spawn();
      const index = this.slots.push(slot);
      slot.ready
        .then(() => {
          if (!slot.shard) slot.worker.postMessage({ type: 'run', seq: WARM_SEQ, xml, stat: 'Life', maxDepth: 1, index, count } satisfies PowerPoolRequest);
        })
        .catch(() => {});
    }
    this.release();
    return this.slots.slice(0, count - 1).every((s) => s.warmed);
  }

  cancel(idle = true) {
    for (const slot of this.slots) {
      if (!slot.shard) continue;
      slot.worker.postMessage({ type: 'cancel' } satisfies PowerPoolRequest);
      const s = slot.shard;
      slot.shard = undefined;
      s.settle(undefined);
    }
    if (idle) this.release();
  }

  private release() {
    if (this.slots.some((s) => s.shard)) return;
    clearTimeout(this.idle);
    this.idle = setTimeout(() => this.terminate(), POWER_POOL_IDLE_MS);
  }

  terminate() {
    clearTimeout(this.idle);
    this.idle = undefined;
    this.cancel(false);
    for (const slot of [...this.slots]) this.drop(slot);
  }

  stats(): { workers: number; warm: number; running: number; bootMs: (number | undefined)[]; memoryMB: (number | undefined)[] } {
    return {
      workers: this.slots.length,
      warm: this.slots.filter((s) => s.warmed).length,
      running: this.slots.filter((s) => s.shard).length,
      bootMs: this.slots.map((s) => s.bootMs),
      memoryMB: this.slots.map((s) => s.memoryMB)
    };
  }
}
