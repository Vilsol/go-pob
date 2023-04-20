import { browser } from '$app/environment';
import SyncWorker from './sync_worker?worker';
import * as Comlink from 'comlink';
import type { WorkerType } from './sync_worker';

function getWorker(): { syncWorker: Worker; syncWrap: Comlink.Remote<WorkerType> } {
  if (typeof globalThis !== 'undefined') {
    if ('cachedWorker' in globalThis) {
      // eslint-disable-next-line
      // @ts-ignore
      return globalThis.cachedWorker;
    }
  }

  console.log('Creating sync worker');
  const theWorker = new SyncWorker();
  const obj = Comlink.wrap<WorkerType>(theWorker);
  const result = { syncWorker: theWorker, syncWrap: obj };
  if (typeof globalThis !== 'undefined') {
    // eslint-disable-next-line
    // @ts-ignore
    globalThis.cachedWorker = result;
  }
  return result;
}

export const { syncWorker, syncWrap } = browser ? getWorker() : { syncWorker: null, syncWrap: null };
