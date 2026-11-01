import { fs } from '@zenfs/core';
import { builds, storage } from '../types';
import { DESKTOP } from '$lib/pwa/desktop';
import { TabStorage, type StorageWrite } from '$lib/storage/tabs';
import { remountStore } from './storage_config';

const entry = (name: string, dir: boolean, lastEdit: string, file?: Uint8Array): storage.DirEntry => {
  let summary: builds.SavedBuild | undefined;
  if (file) {
    const parsed = builds.SavedBuildSummary(new TextDecoder().decode(file));
    summary = parsed.ok ? parsed.value : undefined;
  }
  return {
    Name: name,
    Type: dir ? 'dir' : 'file',
    Class: summary?.ClassName || '',
    LastEdit: lastEdit,
    Level: summary?.Level || 0,
    Ascendancy: summary?.Ascendancy ?? '',
    MainSkill: summary?.MainSkill ?? '',
    Game: summary?.Game ?? ''
  };
};

const disk = (bucket: string, path = '') =>
  `/desktop/fs/${bucket}/${path
    .split('/')
    .filter((p) => p !== '')
    .map(encodeURIComponent)
    .join('/')}`;

const failed = async (res: Response, what: string) => new Error(`${what}: ${res.status} ${(await res.text()).trim()}`);

const diskRead = async (bucket: string, key: string) => {
  const res = await fetch(disk(bucket, key));
  if (!res.ok) throw await failed(res, `read ${bucket}/${key}`);
  return new Uint8Array(await res.arrayBuffer());
};

const diskList = async (bucket: string, dir: string) => {
  const res = await fetch(disk(bucket, dir));
  if (!res.ok) throw await failed(res, `list ${bucket}/${dir}`);
  return (await res.json()) as { name: string; dir: boolean; lastEdit: string }[];
};

export const storageFs = DESKTOP
  ? {
      readdir: async (path: string) => {
        const [bucket, ...rest] = path.split('/').filter((p) => p !== '');
        return (await diskList(bucket, rest.join('/'))).map((e) => e.name);
      },
      stat: async (path: string) => {
        const [bucket, ...rest] = path.split('/').filter((p) => p !== '');
        const parent = rest.slice(0, -1).join('/');
        const dir = (await diskList(bucket, parent)).find((e) => e.name === rest.at(-1))?.dir ?? false;
        return { isDirectory: () => dir };
      },
      readFile: async (path: string) => {
        const [bucket, ...rest] = path.split('/').filter((p) => p !== '');
        return diskRead(bucket, rest.join('/'));
      },
      rm: async (path: string, opts?: { force?: boolean }) => {
        const [bucket, ...rest] = path.split('/').filter((p) => p !== '');
        const res = await fetch(disk(bucket, rest.join('/')), { method: 'DELETE' });
        if (!res.ok && !(opts?.force && res.status === 404)) throw await failed(res, `delete ${path}`);
      }
    }
  : fs.promises;

const diskStorage = [
  async (bucket: string, dir: string) =>
    Promise.all(
      (await diskList(bucket, dir)).map(async (e) => entry(e.name, e.dir, e.lastEdit, e.dir ? undefined : await diskRead(bucket, `${dir}/${e.name}`)))
    ),
  async (bucket: string, key: string) => diskRead(bucket, key),
  async (bucket: string, key: string, value: Uint8Array | undefined) => {
    if (!value || !key) {
      return;
    }
    const res = await fetch(disk(bucket, key), { method: 'PUT', body: value });
    if (!res.ok) throw await failed(res, `write ${bucket}/${key}`);
  },
  async (bucket: string, key: string) => (await fetch(disk(bucket, key), { method: 'HEAD' })).ok,
  async (bucket: string, dir: string) => {
    const res = await fetch(disk(bucket, dir), { method: 'POST' });
    if (!res.ok) console.error(await failed(res, `mkdir ${bucket}/${dir}`));
  },
  async (bucket: string, path: string) => {
    const res = await fetch(disk(bucket, path), { method: 'DELETE' });
    if (!res.ok) console.error(await failed(res, `delete ${bucket}/${path}`));
  }
] as const;

// mkdir and delete only log a failure, as they always have; a failed set rejects
const zenfsWrite = async (w: StorageWrite) => {
  const path = w.kind === 'clear' ? `/${w.bucket}` : `/${w.bucket}/${w.key}`;
  const logged = (res: () => void) => (err?: unknown) => {
    if (err) console.error(err);
    res();
  };
  switch (w.kind) {
    case 'set':
      console.log('Writing to storage', path);
      return new Promise<void>((res, rej) => fs.writeFile(path, w.value, (err) => (err ? rej(err) : res())));
    case 'mkdir':
      return new Promise<void>((res) => fs.mkdir(path, 0o777, logged(res)));
    case 'delete':
      return new Promise<void>((res) => fs.unlink(path, logged(res)));
    case 'clear':
      for (const n of await fs.promises.readdir(path)) await fs.promises.rm(`${path}/${n}`, { recursive: true, force: true });
      return;
  }
};

let tabs: TabStorage | undefined;

// the main worker of every tab; resolves with whether this tab writes the databases, before they are mounted
export const startTabStorage = (changed: (bucket: string) => void) => {
  tabs = new TabStorage('go-pob-storage', {
    buckets: ['cache', 'builds', 'settings'],
    apply: zenfsWrite,
    remount: (bucket) => remountStore(`/${bucket}`, bucket),
    changed,
    lazy: { cache: 1000 }
  });
  return tabs.start();
};

export const storageWrite = (w: StorageWrite) => (tabs ? tabs.write(w) : zenfsWrite(w));

export const otherTabsOpen = () => tabs?.othersOpen() ?? Promise.resolve(false);

// a second zenfs instance on the same database allocates inode ids the first already uses: a power worker only reads,
// and a tab that is not the writer forwards its writes to the one that is
export const initializeStorage = async (readOnly = false) => {
  if (DESKTOP) {
    await storage.InitializeStorage(...diskStorage);
    return;
  }
  await storage.InitializeStorage(
    async (bucket: string, dir: string) =>
      new Promise((res, rej) => {
        fs.readdir(`/${bucket}/${dir}`, (err, files) => {
          if (err) {
            console.error(err);
            return rej(err);
          }

          res(
            files?.map((file) => {
              const fullPath = `/${bucket}/${dir}/${file}`;
              const stats = fs.statSync(fullPath);

              let summary: builds.SavedBuild | undefined;
              if (stats.isFile()) {
                const buildFile = fs.readFileSync(fullPath);
                if (buildFile) {
                  const parsed = builds.SavedBuildSummary(buildFile.toString());
                  summary = parsed.ok ? parsed.value : undefined;
                }
              }

              return {
                Name: file,
                Type: stats.isDirectory() ? 'dir' : 'file',
                Class: summary?.ClassName || '',
                LastEdit: stats.mtime.toISOString(),
                Level: summary?.Level || 0,
                Ascendancy: summary?.Ascendancy ?? '',
                MainSkill: summary?.MainSkill ?? '',
                Game: summary?.Game ?? ''
              };
            })
          );
        });
      }),
    async (bucket: string, key: string) =>
      new Promise((res, rej) => {
        fs.readFile(`/${bucket}/${key}`, (err, file) => {
          if (err) {
            console.error(err);
            return rej(err);
          }

          if (file) {
            return res(new Uint8Array(file));
          }

          return res(new Uint8Array(0));
        });
      }),
    async (bucket: string, key: string, value: Uint8Array | undefined) => {
      if (!value || !key || readOnly) {
        return;
      }
      await storageWrite({ kind: 'set', bucket, key, value });
    },
    async (bucket: string, key: string) =>
      new Promise((res) => {
        fs.access(`/${bucket}/${key}`, (err) => {
          if (err) {
            if (err.code !== 'ENOENT') {
              console.error(err);
            }
            return res(false);
          }

          return res(true);
        });
      }),
    async (bucket: string, dir: string) => {
      if (!readOnly) await storageWrite({ kind: 'mkdir', bucket, key: dir });
    },
    async (bucket: string, path: string) => {
      if (!readOnly) await storageWrite({ kind: 'delete', bucket, key: path });
    }
  );
};
