import { execFileSync } from 'node:child_process';
import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { expect, test } from 'vitest';

import { LINK_ROUTES, model, select, WASM_BOOT } from './affected.mjs';

// the import graph alone, so these hold whatever the recorded map says
const m = { ...model(), coverage: undefined };
const all = Object.keys(m.specs).sort();
const noGo = () => new Map();
const pick = (...files) => select(files, m, { max: Infinity, go: noGo });

test('a change to the harness or the build reaches every spec', () => {
  for (const f of ['frontend/e2e/support/app.ts', 'frontend/e2e/playwright.config.ts', 'frontend/static/favicon.png', 'frontend/vite.config.js']) {
    const r = pick(f);
    expect(r.everything, f).toEqual([f]);
    expect(r.specs, f).toEqual(all);
  }
});

test('without a recorded map, a file of the shell reaches every spec', () => {
  for (const f of ['frontend/src/lib/types/index.js', 'frontend/src/lib/go/sync_worker.ts', 'frontend/src/routes/+layout.svelte'])
    expect(pick(f).specs, f).toEqual(all);
});

test("a selection over the cap names its size and runs only the change's own specs", () => {
  const r = select(['frontend/e2e/support/app.ts', 'frontend/e2e/notes.spec.ts'], m, { max: 5, go: noGo });
  expect(r.capped).toEqual({ reached: all.length, max: 5 });
  expect(r.specs).toEqual(['e2e/notes.spec.ts']);
  expect(select(['frontend/src/routes/+layout.svelte'], m, { max: 5, go: noGo }).specs).toEqual([]);
});

test('a changed spec runs itself', () => {
  expect(pick('frontend/e2e/notes.spec.ts')).toEqual({ specs: ['e2e/notes.spec.ts'], everything: [], unmapped: [], capped: undefined });
});

// a map that recorded notes and items but not tree: a shell file only items ran
const recorded = (files) => ({
  ...m,
  coverage: {
    commit: 'fixture',
    files: new Set(['frontend/src/lib/shell/Poe2Page.svelte', 'frontend/src/lib/types/index.js', 'frontend/src/lib/types2/index.js']),
    specs: files
  }
});
const cov = recorded({
  'e2e/notes.spec.ts': new Set(['frontend/src/lib/types/index.js']),
  'e2e/items.spec.ts': new Set(['frontend/src/lib/types/index.js', 'frontend/src/lib/shell/Poe2Page.svelte', 'frontend/src/lib/types2/index.js'])
});

test('a recorded spec is picked by what it ran, an unrecorded one by its imports', () => {
  const r = select(['frontend/src/lib/shell/Poe2Page.svelte'], cov, { max: Infinity, go: noGo });
  expect(r.specs).toContain('e2e/items.spec.ts');
  expect(r.specs).not.toContain('e2e/notes.spec.ts');
  // tree.spec is not in the map, and the file is in the shell's closure
  expect(r.specs).toContain('e2e/tree.spec.ts');
});

test('a file the map never saw is mapped by imports, even for recorded specs', () => {
  const r = select(['frontend/src/routes/+layout.svelte'], cov, { max: Infinity, go: noGo });
  expect(r.specs).toEqual(all);
});

test('a Go file reaches the specs that ran the boot of a wasm that compiles it', () => {
  const go = () =>
    new Map([
      ['calculator2', ['wasm2']],
      ['moddb', ['wasm', 'wasm2']]
    ]);
  const only = (r) => r.specs.filter((s) => s === 'e2e/notes.spec.ts' || s === 'e2e/items.spec.ts');
  expect(only(select(['calculator2/calc_offence.go'], cov, { max: Infinity, go }))).toEqual(['e2e/items.spec.ts']);
  expect(only(select(['moddb/moddb.go'], cov, { max: Infinity, go }))).toEqual(['e2e/items.spec.ts', 'e2e/notes.spec.ts']);
  expect(select(['scripts/gopob/main.go'], cov, { max: Infinity, go })).toMatchObject({ specs: [], unmapped: ['scripts/gopob/main.go'] });
  expect(select(['calculator2/calc_offence_test.go'], cov, { max: Infinity, go })).toMatchObject({ specs: [], unmapped: [] });
});

test("each wasm build's bindings are the file that instantiates it", () => {
  for (const f of Object.values(WASM_BOOT)) expect(readFileSync(resolve(import.meta.dirname, '../..', f), 'utf8'), f).toMatch(/WebAssembly\.instantiate/);
});

test('a page component runs the specs that visit its page, not the others', () => {
  const items = pick('frontend/src/lib/design/ItemsPage.svelte');
  expect(items.specs).toContain('e2e/items.spec.ts');
  expect(items.specs).toContain('e2e/items_ui.spec.ts');
  expect(items.specs).not.toContain('e2e/notes.spec.ts');
  expect(items.specs).not.toContain('e2e/undo.spec.ts');
  const notes = pick('frontend/src/lib/design/NotesPage.svelte');
  expect(notes.specs).toContain('e2e/notes.spec.ts');
  expect(notes.specs).not.toContain('e2e/items_ui.spec.ts');
});

test('a module a spec imports runs that spec', () => {
  expect(pick('frontend/src/lib/skill_tree/heat.ts').specs).toContain('e2e/tree.spec.ts');
});

test('what nothing reaches is named, and only @smoke covers it', () => {
  expect(pick('calculator/calc_offence.go')).toEqual({ specs: [], everything: [], unmapped: ['calculator/calc_offence.go'], capped: undefined });
  expect(pick('frontend/src/lib/design/Button.stories.svelte', 'README.md')).toEqual({ specs: [], everything: [], unmapped: [], capped: undefined });
});

test('every route is visited by some spec, and every navigation link names a route', () => {
  const routes = readdirSync(resolve(import.meta.dirname, '../src/routes'), { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);
  for (const r of routes)
    expect(
      Object.values(m.specs).some((s) => s.routes.includes(r)),
      r
    ).toBe(true);
  for (const r of Object.values(LINK_ROUTES)) expect(routes).toContain(r);
});

test('the grep pattern matches the picked files only, at a file-name boundary', () => {
  const grep = new RegExp(
    execFileSync('node', [resolve(import.meta.dirname, 'affected.mjs'), 'grep', '--', 'frontend/e2e/items.spec.ts'], {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore']
    })
  );
  expect(grep.test('e2e/items.spec.ts paste, equip')).toBe(true);
  expect(grep.test('e2e/party-items.spec.ts drag reorders')).toBe(false);
  expect(grep.test('e2e/journeys.spec.ts change class @smoke')).toBe(true);
});

test('the committed map picks a PoE2 page for the PoE2 specs, not for a PoE1 spec it recorded', () => {
  const real = model();
  expect(real.coverage?.specs['e2e/notes.spec.ts'], 'coverage-map.json records notes.spec').toBeDefined();
  const r = select(['frontend/src/lib/shell/Poe2Page.svelte'], real, { max: Infinity, go: noGo });
  expect(r.specs).toContain('e2e/poe2-pages.spec.ts');
  expect(r.specs).not.toContain('e2e/notes.spec.ts');
  // the control: a file notes.spec does run selects it
  expect(select(['frontend/src/lib/design/NotesPage.svelte'], real, { max: Infinity, go: noGo }).specs).toContain('e2e/notes.spec.ts');
});
