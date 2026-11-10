import SyncWorker2 from './sync_worker2?worker';
import * as Comlink from 'comlink';
import { assets } from '$app/paths';
import type { Worker2Type } from './sync_worker2';
import { forwardWorkerLog } from '$lib/app_log';
import { WASM2_FALLBACK, WASM2_FILE, engineFile } from './wasm_file';
import { fallBackToGo, runningEngine, watchEngineCrash } from './engine_choice';
import { DESKTOP } from '$lib/pwa/desktop';
import { dataErrorText, poe2DataError } from './data_error';
import type { JobHandle } from '$lib/activity/jobs';
import { readBody } from '$lib/activity/read_body';
import { endSourceJobs, forwardWorkerJobs } from '$lib/activity/worker_jobs';

let worker2: Promise<Comlink.Remote<Worker2Type>> | undefined;

// a page the service worker does not control yet (its first visit) fetches past it; fetched again once it does, so
// the service worker caches it
const cacheOnceControlled = (path: string) => {
  const sw = typeof navigator === 'undefined' ? undefined : navigator.serviceWorker;
  if (!sw || sw.controller) return;
  sw.addEventListener(
    'controllerchange',
    () =>
      void fetch(assets + path)
        .then((r) => r.body?.cancel())
        .catch(() => {}),
    { once: true }
  );
};

// the same byte check +layout.svelte does for the PoE1 wasm: a .br the host did not decode falls back to the .gz
const wasm2Bytes = async (path: string, job?: JobHandle): Promise<ArrayBuffer> => {
  // a .gz arrives as it is stored and is decompressed here, so its bytes are the compressed ones
  job?.phase(path.endsWith('.gz') ? 'Downloading the engine (compressed)' : 'Downloading the engine');
  const res = await fetch(assets + path);
  if (!res.ok) throw new Error(`Failed to load the PoE2 wasm runtime: ${res.status} ${path}`);
  const data = await readBody(res, (done, total) => job?.progress(done, total, 'bytes'));
  const head = new Uint8Array(data, 0, Math.min(4, data.byteLength));
  if (head[0] === 0x00 && head[1] === 0x61 && head[2] === 0x73 && head[3] === 0x6d) {
    cacheOnceControlled(path);
    return data;
  }
  if (head[0] === 0x1f && head[1] === 0x8b) {
    cacheOnceControlled(path);
    job?.phase('Decompressing the engine');
    return new Response(new Response(data).body!.pipeThrough(new DecompressionStream('gzip'))).arrayBuffer();
  }
  const fallback = engineFile(WASM2_FALLBACK, runningEngine);
  if (path !== fallback) {
    console.warn(`${path} arrived still encoded; falling back to ${fallback}`);
    return wasm2Bytes(fallback, job);
  }
  throw new Error('Failed to load the PoE2 wasm runtime: not a wasm module');
};

// created on the first PoE2 build, never at page load; job, on the call that creates it, follows the download and boot
export const poe2Worker = (job?: JobHandle) =>
  (worker2 ??= (async () => {
    console.log('Creating PoE2 sync worker');
    const w = new SyncWorker2({ name: 'calc2' });
    forwardWorkerLog(w);
    forwardWorkerJobs(w, 'calc2');
    watchEngineCrash(w);
    const wrap = Comlink.wrap<Worker2Type>(w);
    const data = await wasm2Bytes(engineFile(WASM2_FILE, runningEngine), job);
    job?.phase('Booting the engine');
    await wrap.boot(
      Comlink.transfer(data, [data]),
      runningEngine,
      Comlink.proxy((message: string) => poe2DataError.set(dataErrorText(message)))
    );
    return wrap;
  })().catch((err: unknown) => {
    worker2 = undefined;
    endSourceJobs('calc2', 'the PoE2 engine did not boot');
    fallBackToGo(`the PoE2 engine did not boot (${err instanceof Error ? err.message : String(err)})`);
    throw err;
  }));

// the desktop PoE2 cache is the PoE1 cache folder, already emptied; on the web a running PoE2 worker clears through its
// own zenfs, the only writer of its database, and with none running the database is deleted whole (a delete another
// tab blocks finishes once that tab closes)
export const clearPoe2DataCache = async (): Promise<number> => {
  if (DESKTOP) return 0;
  if (worker2) {
    const n = await worker2.then((w) => w.ClearDataCache()).catch(() => undefined);
    if (n !== undefined) return n;
  }
  await new Promise<void>((done) => {
    const req = indexedDB.deleteDatabase('cache2');
    req.onsuccess = req.onerror = req.onblocked = () => done();
  });
  return 0;
};
