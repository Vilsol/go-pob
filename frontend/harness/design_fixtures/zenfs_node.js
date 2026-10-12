import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, renameSync, statSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

export const cacheDir = process.env.GO_POB_CACHE_DIR || join(process.env.XDG_CACHE_HOME || join(homedir(), '.cache'), 'go-pob', 'bundle-cache');
export const cacheStats = { hits: 0, misses: 0, writes: 0 };

const cacheFile = (path) => {
  const m = /^\/cache\/(.+)$/.exec(path);
  if (!m) throw new Error(`design fixtures: storage outside the game-data cache was touched: ${path}`);
  const key = Buffer.from(m[1], 'base64url').toString('utf8');
  return join(cacheDir, createHash('sha256').update(key).digest('hex'));
};

const enoent = (path) => Object.assign(new Error(`ENOENT: ${path}`), { code: 'ENOENT' });

export const configure = () => Promise.resolve();

export const fs = {
  access(path, cb) {
    const ok = existsSync(cacheFile(path));
    ok ? cacheStats.hits++ : cacheStats.misses++;
    cb(ok ? null : enoent(path));
  },
  readFile(path, cb) {
    try {
      cb(null, readFileSync(cacheFile(path)));
    } catch (e) {
      cb(e);
    }
  },
  writeFile(path, value, cb) {
    try {
      const to = cacheFile(path);
      mkdirSync(cacheDir, { recursive: true });
      const tmp = `${to}.tmp-${process.pid}`;
      writeFileSync(tmp, value);
      renameSync(tmp, to);
      cacheStats.writes++;
      cb(null);
    } catch (e) {
      cb(e);
    }
  },
  statSync(path) {
    return statSync(cacheFile(path));
  },
  readFileSync(path) {
    return readFileSync(cacheFile(path));
  },
  readdir(path, cb) {
    cb(new Error(`design fixtures: readdir is not served: ${path}`));
  },
  mkdir(path, _mode, cb) {
    cb(new Error(`design fixtures: mkdir is not served: ${path}`));
  },
  unlink(path, cb) {
    cb(new Error(`design fixtures: unlink is not served: ${path}`));
  }
};
