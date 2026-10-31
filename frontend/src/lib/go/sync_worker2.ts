import { expose } from 'comlink';
import { configure, fs } from '@zenfs/core';
import { IndexedDB } from '@zenfs/dom';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { boot, config, exposition, storage } from '../types2';
import type { Poe2View } from '../global';
import { DESKTOP } from '$lib/pwa/desktop';

// only the data cache bucket; on the web it is its own IndexedDB database, since zenfs allocates inode ids per
// instance and two instances writing the PoE1 worker's database overwrite each other's files
const disk = (bucket: string, key: string) => `/desktop/fs/${bucket}/${encodeURIComponent(key)}`;

const cacheGet = async (bucket: string, key: string) => {
  if (DESKTOP) {
    const res = await fetch(disk(bucket, key));
    if (!res.ok) throw new Error(`read ${bucket}/${key}: ${res.status}`);
    return new Uint8Array(await res.arrayBuffer());
  }
  return new Uint8Array(await fs.promises.readFile(`/${bucket}/${key}`));
};

const cacheSet = async (bucket: string, key: string, value: Uint8Array | undefined) => {
  if (!value || !key) return;
  if (DESKTOP) {
    const res = await fetch(disk(bucket, key), { method: 'PUT', body: value });
    if (!res.ok) throw new Error(`write ${bucket}/${key}: ${res.status}`);
    return;
  }
  await fs.promises.writeFile(`/${bucket}/${key}`, value);
};

const cacheExists = async (bucket: string, key: string) => {
  if (DESKTOP) return (await fetch(disk(bucket, key), { method: 'HEAD' })).ok;
  return fs.promises.exists(`/${bucket}/${key}`);
};

const initializeStorage = async () => {
  if (!DESKTOP) await configure({ mounts: { '/cache': await IndexedDB.create({ storeName: 'cache2' }) } });
  await storage.InitializeStorage(
    () => [],
    cacheGet,
    cacheSet,
    cacheExists,
    () => {},
    () => {}
  );
};

class PoB2Worker {
  private booted?: Promise<unknown>;

  boot(wasm: ArrayBuffer) {
    this.booted ??= boot(wasm).then(async () => {
      config.InitLogging(false);
      await initializeStorage();
    });
    return this.booted;
  }

  async Open(xml: string): Promise<Poe2View> {
    await this.booted;
    const calc = exposition.ParseBuild(xml).unwrap();
    if (!calc) throw new Error('the PoE2 build did not parse');
    try {
      const out = (await calc.Calculate()).unwrap();
      if (!out) throw new Error('the PoE2 build did not calculate');
      return { className: out.ClassName, ascendClassName: out.AscendClassName, level: out.Level, output: { ...out.Output } };
    } finally {
      calc.release();
    }
  }

  async ClearDataCache(): Promise<number> {
    await this.booted;
    const names = await fs.promises.readdir('/cache');
    for (const n of names) await fs.promises.rm(`/cache/${n}`, { recursive: true, force: true });
    return names.length;
  }
}

export const worker2Instance = new PoB2Worker();

expose(worker2Instance);

export type Worker2Type = PoB2Worker;
