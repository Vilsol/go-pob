// @vitest-environment jsdom
// Main.lua:1300-1334

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { parseChangelog } from '$lib/design/help';

vi.mock('$app/environment', () => ({ browser: true, dev: false, version: 'old' }));
vi.mock('$app/paths', () => ({ base: '' }));

const files: Record<string, Record<string, string>> = {
  'pob-app-old': { '/changelog.txt': 'VERSION[2.67.2][2026/08/07]\n* running\n' },
  'pob-app-new': { '/changelog.txt': 'VERSION[2.67.3][2026/08/20]\n* Fix the thing (someone)\n\nVERSION[2.67.2][2026/08/07]\n* running\n' },
  'pob-cdn-v1': {}
};
globalThis.caches = {
  keys: () => Promise.resolve(Object.keys(files)),
  match: () => Promise.resolve(undefined),
  open: (name: string) =>
    Promise.resolve({ match: (url: string) => Promise.resolve(files[name]?.[url] === undefined ? undefined : new Response(files[name][url])) })
} as unknown as CacheStorage;

const { readUpdateChanges, updateAvailable, updateChanges } = await import('./index');
const PwaStatus = (await import('./PwaStatus.svelte')).default;

afterEach(cleanup);

describe('the update’s changes', () => {
  it('reads the waiting deployment’s changelog down to the running version', async () => {
    await readUpdateChanges();
    expect(get(updateChanges).map((v) => [v.version, v.date, v.lines.map((l) => l.text).filter(Boolean)])).toEqual([
      ['2.67.3', '2026/08/20', ['* Fix the thing (someone)']]
    ]);
  });

  it('Changes… beside Reload shows them above the prompt; none, no button', async () => {
    updateAvailable.set(true);
    const view = render(PwaStatus);
    const prompt = view.getByRole('alert');
    expect(prompt.textContent).not.toContain('Version 2.67.3');
    await fireEvent.click(view.getByRole('button', { name: 'Changes…' }));
    expect(prompt.textContent).toContain('Version 2.67.3 (2026/08/20)');
    expect(prompt.textContent).toContain('* Fix the thing (someone)');
    updateChanges.set([]);
    await Promise.resolve();
    expect(view.getByRole('alert').textContent).not.toContain('Version');
    expect(view.queryByRole('button', { name: 'Changes…' })).toBeNull();
    updateAvailable.set(false);
  });

  it("an update between two go-pob releases lists go-pob's changes, not Path of Building's", async () => {
    const shipped = readFileSync(resolve(__dirname, '../../../static/changelog.txt'), 'utf8');
    const pob = readFileSync(resolve(__dirname, '../../../static/pob-changelog.txt'), 'utf8');
    const [newest, previous] = parseChangelog(shipped);
    const before = { old: files['pob-app-old'], next: files['pob-app-new'] };
    files['pob-app-old'] = { '/changelog.txt': shipped.slice(shipped.indexOf(`VERSION[${previous.version}]`)), '/pob-changelog.txt': pob };
    files['pob-app-new'] = { '/changelog.txt': shipped, '/pob-changelog.txt': pob };
    try {
      await readUpdateChanges();
      expect(get(updateChanges).map((v) => v.version)).toEqual([newest.version]);
      expect(parseChangelog(shipped).at(-1)?.version, "go-pob's history starts at its first release").toBe('0.1.0');
    } finally {
      files['pob-app-old'] = before.old;
      files['pob-app-new'] = before.next;
    }
  });
});
