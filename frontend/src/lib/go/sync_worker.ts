import { expose } from 'comlink';
import '../../wasm_exec.js';
import { initializeCrystalline, cache, raw, config } from '../types';
import localforage from 'localforage';

const obj = {
  boot(wasm: ArrayBuffer) {
    return new Promise((resolve) => {
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      const go = new Go();
      WebAssembly.instantiate(wasm, go.importObject).then(async (result) => {
        go.run(result.instance);

        initializeCrystalline();

        config.InitLogging();

        await cache.InitializeDiskCache(
          async (key: string) => {
            const item = await localforage.getItem(key);
            if (item) {
              return item as Uint8Array;
            }
            return new Uint8Array(0);
          },
          async (key: string, value: Uint8Array | undefined) => {
            await localforage.setItem(key, value);
          },
          async (key: string) => (await localforage.getItem(key)) instanceof Uint8Array
        );

        resolve(undefined);
      });
    });
  },
  async loadData() {
    const err = await raw.InitializeAll('3.18');
    if (err) {
      console.error(err);
    }
  }
} as const;

expose(obj);

export type WorkerType = typeof obj;
