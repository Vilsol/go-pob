import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { resolve } from 'node:path';

const ROOT = resolve(import.meta.dirname, '../..');

export const DATA_HOST = 'https://data.pob.gg/';

const cacheDir = () => process.env.GO_POB_CACHE_DIR || resolve(process.env.XDG_CACHE_HOME || resolve(homedir(), '.cache'), 'go-pob', 'bundle-cache');

const dataModuleVersion = (() => {
  const src = readFileSync(resolve(ROOT, '../data/raw/cache_namespace.go'), 'utf8');
  const m = src.match(/DataModuleVersion\s*=\s*"([^"]+)"/);
  if (!m) {
    throw new Error('REFUSING: could not read DataModuleVersion from data/raw/cache_namespace.go');
  }
  return m[1];
})();

const KNOWN_ABSENT = new Set(['raw/SkillTotems']);

export type CachedAsset = { status: 200; body: Buffer } | { status: 404 };

export const cachedAsset = (url: string): CachedAsset => {
  const path = resolve(cacheDir(), createHash('sha256').update(`${dataModuleVersion}|${url}`).digest('hex'));
  if (existsSync(path)) {
    return { status: 200, body: readFileSync(path) };
  }

  const name = url.match(/\/data\/[^/]+\/(.+?)(?:\.(?:t|min)-[0-9a-f]{8})?\.msgpack\.br$/)?.[1];
  if (name && KNOWN_ABSENT.has(name)) {
    return { status: 404 };
  }

  throw new Error(
    `REFUSING: ${url} is not in the Go bundle cache (${cacheDir()}). Populate it by running any ` +
      'Go test that loads game data (e.g. `task gate`), then re-run.'
  );
};
