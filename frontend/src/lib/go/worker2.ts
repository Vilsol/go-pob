import SyncWorker2 from './sync_worker2?worker';
import * as Comlink from 'comlink';
import { assets } from '$app/paths';
import type { Worker2Type } from './sync_worker2';
import { forwardWorkerLog } from '$lib/app_log';
import { WASM2_FILE } from './wasm_file';

let worker2: Promise<Comlink.Remote<Worker2Type>> | undefined;

const wasm2Bytes = async () => {
  const res = await fetch(assets + WASM2_FILE);
  if (!res.ok) throw new Error(`Failed to load the PoE2 wasm runtime: ${res.status} ${WASM2_FILE}`);
  return res.arrayBuffer();
};

// created on the first PoE2 build, never at page load
export const poe2Worker = () =>
  (worker2 ??= (async () => {
    console.log('Creating PoE2 sync worker');
    const w = new SyncWorker2({ name: 'calc2' });
    forwardWorkerLog(w);
    const wrap = Comlink.wrap<Worker2Type>(w);
    const data = await wasm2Bytes();
    await wrap.boot(Comlink.transfer(data, [data]));
    return wrap;
  })().catch((err: unknown) => {
    worker2 = undefined;
    throw err;
  }));
