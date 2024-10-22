import { browser } from '$app/environment';
import SyncWorker from './sync_worker?worker';
import * as Comlink from 'comlink';
import type { WorkerType } from './sync_worker';
import type { ProxiedRemote } from '$lib/type_utils';

type WorkerData = { syncWorker: Worker; syncWrap: ProxiedRemote<WorkerType> };

function getWorker(): WorkerData {
  if (typeof globalThis !== 'undefined') {
    if ('cachedWorker' in globalThis) {
      // @ts-expect-error TS7017
      // eslint-disable-next-line @typescript-eslint/no-unsafe-return
      return globalThis.cachedWorker;
    }
  }

  console.log('Creating sync worker');
  const theWorker = new SyncWorker();
  const obj = Comlink.wrap<WorkerType>(theWorker) as ProxiedRemote<WorkerType>;
  const result = { syncWorker: theWorker, syncWrap: obj };
  if (typeof globalThis !== 'undefined') {
    // eslint-disable-next-line
    // @ts-ignore
    globalThis.cachedWorker = result;
  }
  return result;
}

export const { syncWorker, syncWrap } = browser ? getWorker() : ({ syncWorker: undefined, syncWrap: undefined } as unknown as WorkerData);
