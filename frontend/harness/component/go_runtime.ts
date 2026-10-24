import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';

import { get } from 'svelte/store';

import { workerInstance } from '$lib/go/sync_worker';
import { builds } from '$lib/types';
import * as goTypes from '$lib/types';
import type { currentBuild as currentBuildStore, BuildChannel } from '$lib/global';
import type { Outputs } from '$lib/custom_types';

import { asRemote } from './as_remote';

const ROOT = resolve(process.cwd());
if (!existsSync(resolve(ROOT, 'svelte.config.js'))) {
  throw new Error(
    `REFUSING: the component harness resolves paths from the working directory, and this one does ` +
      `not look like frontend/ (no svelte.config.js in ${ROOT}). Run vitest from frontend/.`
  );
}

const WASM = resolve(ROOT, 'static/go-pob.wasm');
const CORPUS = resolve(ROOT, '../testdata/corpus-3.29');

const refuse = (what: string, path: string): never => {
  throw new Error(
    `REFUSING: ${what}\n  missing: ${path}\n` +
      `  The component harness cannot run without the real Go module — a mocked worker would make\n` +
      `  every write-path assertion vacuous, which is the defect class this harness exists to see.`
  );
};

export const recordedOutputs: Outputs[] = [];

const assertNotStale = async (mod: Record<string, unknown>) => {
  const genGlue = resolve(ROOT, '../wasm/exposition/crystalline_gen.go');
  if (!existsSync(genGlue)) {
    return;
  }

  const glue = await readFile(genGlue, 'utf8');
  const declared = new Map<string, Set<string>>();
  for (const m of glue.matchAll(/crystallineNamespace\("go-pob",\s*"(\w+)"\)\.Set\("(\w+)"/g)) {
    const [, ns, fn] = m;
    if (!declared.has(ns)) {
      declared.set(ns, new Set());
    }
    declared.get(ns)!.add(fn);
  }

  const total = [...declared.values()].reduce((n, s) => n + s.size, 0);
  if (total === 0) {
    refuse(
      'parsed ZERO function registrations out of crystalline_gen.go, so the staleness check would ' +
        "pass vacuously. The generator's emitted shape has changed; fix the pattern in assertNotStale.",
      genGlue
    );
  }

  const missing: string[] = [];
  for (const [ns, fns] of declared) {
    const live = mod[ns] as Record<string, unknown> | undefined;
    for (const fn of fns) {
      if (!live || live[fn] === undefined) {
        missing.push(`${ns}.${fn}`);
      }
    }
  }

  if (missing.length > 0) {
    refuse(
      `go-pob.wasm is STALE relative to the generated bindings — it is missing ${missing.length} of ` +
        `${total} registered function(s) that crystalline_gen.go declares:\n    ` +
        `${missing.slice(0, 8).join(', ')}${missing.length > 8 ? ', …' : ''}\n` +
        `  The module is gitignored, so a checkout leaves whatever you last built. Run ` +
        `\`mise run build-go\`, or use \`task component-tests\`, which depends on it.`,
      WASM
    );
  }
};

let booting: Promise<typeof workerInstance> | undefined;

export const bootWorker = async (store: typeof currentBuildStore): Promise<typeof workerInstance> => {
  if (booting) {
    return booting;
  }

  booting = (async () => {
    if (!existsSync(WASM)) {
      refuse('the wasm module — run `mise run build-go`', WASM);
    }

    const proxyingChannel: BuildChannel = {
      set: (value) => store.set(value === undefined ? undefined : asRemote(value)),
      touch: () => store.set(get(store)),
      restore: (value) => store.set(asRemote(value))
    };

    const wasm = await readFile(WASM);
    await workerInstance.boot(
      wasm.buffer.slice(wasm.byteOffset, wasm.byteOffset + wasm.byteLength) as ArrayBuffer,
      (out: Outputs | undefined) => {
        if (out) recordedOutputs.push(out);
      },
      proxyingChannel
    );

    await assertNotStale(goTypes as unknown as Record<string, unknown>);

    return workerInstance;
  })();

  return booting;
};

export const remoteWorker = () => asRemote(workerInstance);

export const loadCorpusBuild = async (file: string): Promise<void> => {
  const path = `${CORPUS}/${file}`;
  if (!existsSync(path)) {
    refuse('a corpus build — the corpus is git-LFS, run `git lfs pull`', path);
  }

  const xml = await readFile(path, 'utf8');
  const parsed = builds.ParseBuildStr(xml);
  const build = parsed.unwrap();
  if (!build) {
    throw new Error(`REFUSING: ParseBuildStr returned nothing for ${file}`);
  }

  workerInstance.currentBuild = build;
};
