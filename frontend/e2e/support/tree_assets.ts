import { existsSync, mkdirSync, readFileSync, renameSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { createHash } from 'node:crypto';
import type { Page } from '@playwright/test';

import { RESULTS_DIR } from './perf';

export const TREE_ASSET_PATTERN = '^https://data\\.pob\\.gg/data/[^/]+/(tree/assets|raw/Art)/';
export const isTreeAsset = (url: string) => new RegExp(TREE_ASSET_PATTERN).test(url);
const DIR = resolve(RESULTS_DIR, 'tree-assets');

export const FONT_ASSET_PATTERN = '^https://fonts\\.(googleapis|gstatic)\\.com/';
const isFontAsset = (url: string) => new RegExp(FONT_ASSET_PATTERN).test(url);

const ITEM_ART_PATTERN = /^https:\/\/web\.poecdn\.com\//;
// a 1×1 transparent PNG: item and gem icons load without reaching the CDN
const ITEM_ART = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==', 'base64');

const failed = new Set<string>();

const localCopy = async (url: string): Promise<{ file: string; type: string } | undefined> => {
  const file = resolve(DIR, createHash('sha256').update(url).digest('hex'));
  if (existsSync(file)) return { file, type: existsSync(`${file}.type`) ? readFileSync(`${file}.type`, 'utf8') : '' };
  for (let attempt = 1; ; attempt++) {
    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const type = res.headers.get('content-type') ?? '';
      mkdirSync(DIR, { recursive: true });
      const tmp = `${file}.${process.pid}.tmp`;
      writeFileSync(tmp, Buffer.from(await res.arrayBuffer()));
      writeFileSync(`${file}.type`, type);
      renameSync(tmp, file);
      return { file, type };
    } catch {
      if (attempt === 3) return undefined;
      await new Promise((r) => setTimeout(r, 1_000 * attempt));
    }
  }
};

export const routeTreeAssets = async (page: Page) => {
  await page.context().route(ITEM_ART_PATTERN, (route) => route.fulfill({ status: 200, body: ITEM_ART, headers: { 'content-type': 'image/png' } }));
  await page.context().route(isTreeAsset, async (route) => {
    const url = route.request().url();
    const type = url.endsWith('.png') ? 'image/png' : url.endsWith('.webp') ? 'image/webp' : 'image/jpeg';
    const copy = await localCopy(url);
    if (!copy) {
      failed.add(url);
      return route.abort();
    }
    return route.fulfill({ status: 200, body: readFileSync(copy.file), headers: { 'access-control-allow-origin': '*', 'content-type': type } });
  });
  await page.context().route(isFontAsset, async (route) => {
    const url = route.request().url();
    const copy = await localCopy(url);
    if (!copy) {
      failed.add(url);
      return route.abort();
    }
    return route.fulfill({ status: 200, body: readFileSync(copy.file), headers: { 'access-control-allow-origin': '*', 'content-type': copy.type } });
  });
};

export const treeAssetsFailed = () => [...failed];
