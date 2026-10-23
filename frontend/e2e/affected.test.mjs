import { execFileSync } from 'node:child_process';
import { readdirSync } from 'node:fs';
import { resolve } from 'node:path';
import { expect, test } from 'vitest';

import { LINK_ROUTES, model, select } from './affected.mjs';

const m = model();
const pick = (...files) => select(files, m);

test('a change to the harness, the bindings or the shell runs every spec', () => {
  for (const f of [
    'frontend/e2e/support/app.ts',
    'frontend/e2e/playwright.config.ts',
    'frontend/src/lib/types/index.js',
    'frontend/src/lib/go/sync_worker.ts',
    'frontend/src/routes/+layout.svelte',
    'wasm/exposition/main.go'
  ]) {
    const r = pick(f);
    expect(r.everything, f).toEqual([f]);
    expect(r.specs, f).toEqual(Object.keys(m.specs).sort());
  }
});

test('a changed spec runs itself', () => {
  expect(pick('frontend/e2e/notes.spec.ts')).toEqual({ specs: ['e2e/notes.spec.ts'], everything: [], unmapped: [] });
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
  expect(pick('calculator/calc_offence.go')).toEqual({ specs: [], everything: [], unmapped: ['calculator/calc_offence.go'] });
  expect(pick('frontend/src/lib/design/Button.stories.svelte', 'README.md')).toEqual({ specs: [], everything: [], unmapped: [] });
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
