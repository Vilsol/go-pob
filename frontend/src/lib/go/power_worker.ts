import '$lib/console_hook';
import '../../wasm_exec.js';
import { initializeCrystalline, config, raw, builds, exposition } from '../types';
import { releaseWrapper } from '../type_utils';
import { configureStorage } from './storage_config';
import { initializeStorage } from './storage_init';
import type { PowerPoolReply, PowerPoolRequest } from './power_pool';

const post = (m: PowerPoolReply) => postMessage(m);

const yieldToQueue = () =>
  new Promise<void>((resolve) => {
    const { port1, port2 } = new MessageChannel();
    port1.onmessage = () => {
      port1.close();
      resolve();
    };
    port2.postMessage(undefined);
  });

const ROWS_MS = 500;

let seq = 0;
let memory: WebAssembly.Memory | undefined;

const boot = async (module: WebAssembly.Module) => {
  const start = performance.now();
  const go = new Go();
  const instance = await WebAssembly.instantiate(module, go.importObject);
  void go.run(instance);
  memory = instance.exports.mem as WebAssembly.Memory;
  initializeCrystalline();
  config.InitLogging(false);
  await configureStorage();
  await initializeStorage();
  const result = await raw.InitializeAll(raw.GameDataVersion(), () => Promise.resolve());
  if (!result.ok) throw new Error(String(result.error));
  return performance.now() - start;
};

const run = async (r: Extract<PowerPoolRequest, { type: 'run' }>) => {
  const live = () => r.seq === seq;
  const build = builds.ParseBuildStr(r.xml).unwrap();
  const calc = await exposition.GetMiscCalculator(build);
  const builder = calc ? exposition.NewPowerBuilderShard(calc, r.stat, r.maxDepth, r.index, r.count) : undefined;
  try {
    if (!builder) throw new Error(`no power report of ${r.stat}`);
    const tasks = builder.Tasks();
    let sent = performance.now();
    for (let done = 0; done < 1; ) {
      if (!live()) return;
      done = builder.Step(40);
      const now = performance.now();
      const rows = now - sent >= ROWS_MS;
      if (rows) sent = now;
      post({ type: 'progress', seq: r.seq, done, tasks, shard: rows ? builder.Shard() : undefined });
      await yieldToQueue();
    }
    if (live()) post({ type: 'done', seq: r.seq, shard: builder.Shard(), tasks, memory: memory?.buffer.byteLength ?? 0 });
  } finally {
    releaseWrapper(builder);
    releaseWrapper(calc);
    releaseWrapper(build);
  }
};

let booted: Promise<number> | undefined;

onmessage = (e: MessageEvent<PowerPoolRequest>) => {
  const m = e.data;
  if (m.type === 'boot') {
    booted = boot(m.module);
    booted.then((ms) => post({ type: 'ready', ms })).catch((err) => post({ type: 'error', seq: 0, message: String(err) }));
  } else if (m.type === 'run') {
    seq = m.seq;
    void booted?.then(() => run(m)).catch((err) => post({ type: 'error', seq: m.seq, message: String(err) }));
  } else if (m.type === 'cancel') {
    seq = 0;
  }
};
