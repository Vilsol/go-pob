import { existsSync, mkdirSync, readdirSync, rmSync } from 'node:fs';
import { resolve } from 'node:path';
import { RESULTS_DIR } from './support/perf';

// the routed tree art and fonts (support/tree_assets.ts) are keyed by URL and kept, so runs stop fetching them
const KEPT = new Set(['tree-assets']);

export default function globalSetup() {
  if (existsSync(RESULTS_DIR)) {
    for (const name of readdirSync(RESULTS_DIR)) if (!KEPT.has(name)) rmSync(resolve(RESULTS_DIR, name), { recursive: true, force: true });
  }
  mkdirSync(RESULTS_DIR, { recursive: true });
}
