import { existsSync, mkdirSync, readdirSync, rmSync } from 'node:fs';
import { resolve } from 'node:path';
import { RESULTS_DIR } from './support/perf';

// the routed tree art and fonts (support/tree_assets.ts) are keyed by URL and kept, so runs stop fetching them; the
// shard's test list (shard.mjs) is read after this runs
const KEPT = new Set(['tree-assets', 'shard-tests.txt']);

export default function globalSetup() {
  if (existsSync(RESULTS_DIR)) {
    for (const name of readdirSync(RESULTS_DIR)) if (!KEPT.has(name)) rmSync(resolve(RESULTS_DIR, name), { recursive: true, force: true });
  }
  mkdirSync(RESULTS_DIR, { recursive: true });
}
