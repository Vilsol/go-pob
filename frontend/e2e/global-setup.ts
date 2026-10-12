import { mkdirSync, rmSync } from 'node:fs';
import { RESULTS_DIR } from './support/perf';

export default function globalSetup() {
  rmSync(RESULTS_DIR, { recursive: true, force: true });
  mkdirSync(RESULTS_DIR, { recursive: true });
}
